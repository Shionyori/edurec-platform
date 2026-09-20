package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/config"
)

// 条目被跳过的原因。这些字符串会直接展示给使用者，用于反查 YAML 里的字段映射，
// 因此措辞要能指明「改哪里」而不是只说「数据不对」。
const (
	ReasonMissingTitle     = "缺少标题（检查 fields.title）"
	ReasonMissingCategory  = "缺少分类（检查 fields.category 或 default_category）"
	ReasonMissingSourceURL = "无法确定来源链接（检查 fields.source_url 或 source_url_template）"
)

// maxDatasetTags 单条标签数量上限，与 collect.py 的 MAX_TAGS 语义一致
const maxDatasetTags = 10

// DatasetLoadResult 数据集解析与字段映射的结果。
// Total 是文件里的记录条数（映射前），SkipReasons 按原因汇总被跳过的条数。
type DatasetLoadResult struct {
	Items       []CrawlItem
	Total       int
	SkipReasons map[string]int
}

// Skipped 被跳过的条数
func (r *DatasetLoadResult) Skipped() int {
	total := 0
	for _, count := range r.SkipReasons {
		total += count
	}
	return total
}

// LoadDataset 读取数据集文件并按 cfg.Fields 的声明式映射整理成 CrawlItem。
//
// 本函数只做「外部格式 → 平台采集契约」的转换，不碰数据库；落库交给
// CrawlImportService.ImportItems，因此可以先用 cmd/import_dataset --preview 看映射结果。
//
// cfg 的合法性由调用方用 DatasetConfig.Validate 先行校验（只有调用方知道数据集的名字，
// 报错信息里带上名字才查得动 YAML）。此处不再校验：字段映射写错时会体现为
// 「整批条目因缺某个字段被跳过」，跳过原因已指明该改哪个 key，不会静默出错。
func LoadDataset(cfg config.DatasetConfig) (*DatasetLoadResult, error) {
	// 有的数据集（尤其 Windows 上导出的 CSV）带 UTF-8 BOM，不剥掉会让首个
	// 列名/字段名多出一个 U+FEFF 前缀，导致映射静默取不到值
	raw, err := os.ReadFile(cfg.File)
	if err != nil {
		// 路径写错是最常见的失败，报错要直接点名是哪个文件；
		// 这属于配置问题而非服务端故障，故按 BadRequest 而非 Internal 返回
		return nil, apperror.BadRequest(fmt.Sprintf("读取数据集文件 %q 失败: %v", cfg.File, err))
	}
	data := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	records, err := parseDataset(cfg, data)
	if err != nil {
		return nil, err
	}

	result := &DatasetLoadResult{
		Items:       make([]CrawlItem, 0, len(records)),
		Total:       len(records),
		SkipReasons: map[string]int{},
	}
	for _, record := range records {
		item, reason := mapDatasetRecord(record, cfg)
		if reason != "" {
			result.SkipReasons[reason]++
			continue
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// parseDataset 按容器格式把文件解析成「字段名 → 值」的记录列表
func parseDataset(cfg config.DatasetConfig, data []byte) ([]map[string]any, error) {
	switch resolveDatasetFormat(cfg, data) {
	case config.DatasetFormatJSON:
		return parseJSONDataset(cfg, data)
	case config.DatasetFormatCSV:
		return parseCSVDataset(data)
	default:
		return parseJSONLDataset(data)
	}
}

// resolveDatasetFormat 决定用哪种解析器。配置写死优先，其余按扩展名判定，
// 扩展名不认识时再看首字符：{ 或 [ 视为 JSON，否则按逐行 JSON 试。
func resolveDatasetFormat(cfg config.DatasetConfig, data []byte) string {
	if format := cfg.FormatOrDefault(); format != config.DatasetFormatAuto {
		return format
	}
	switch strings.ToLower(filepath.Ext(cfg.File)) {
	case ".jsonl", ".ndjson":
		return config.DatasetFormatJSONL
	case ".csv":
		return config.DatasetFormatCSV
	case ".json":
		return config.DatasetFormatJSON
	}
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 &&
		(trimmed[0] == '[' || trimmed[0] == '{') {
		return config.DatasetFormatJSON
	}
	return config.DatasetFormatJSONL
}

func parseJSONDataset(cfg config.DatasetConfig, data []byte) ([]map[string]any, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, apperror.Internal(fmt.Errorf(
			"解析 JSON 失败（format=json；若文件是每行一个 JSON 对象，请设 format: jsonl）: %w", err))
	}

	node := walkPath(root, cfg.ItemsPath)
	// 单个对象也接受：省去使用者为一条记录去包一层数组
	if _, ok := node.(map[string]any); ok {
		return []map[string]any{toRecord(node)}, nil
	}

	array, ok := node.([]any)
	if !ok {
		hint := "根节点"
		if cfg.ItemsPath != "" {
			hint = fmt.Sprintf("items_path=%q 指向的节点", cfg.ItemsPath)
		}
		return nil, apperror.BadRequest(fmt.Sprintf(
			"数据集%s不是数组，请检查 items_path", hint))
	}

	records := make([]map[string]any, 0, len(array))
	for i, entry := range array {
		record := toRecord(entry)
		if record == nil {
			return nil, apperror.BadRequest(fmt.Sprintf(
				"数据集第 %d 条不是对象（items_path=%q）", i+1, cfg.ItemsPath))
		}
		records = append(records, record)
	}
	return records, nil
}

func parseJSONLDataset(data []byte) ([]map[string]any, error) {
	records := make([]map[string]any, 0)
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, apperror.Internal(fmt.Errorf("解析 JSONL 第 %d 行失败: %w", i+1, err))
		}
		record := toRecord(entry)
		if record == nil {
			return nil, apperror.BadRequest(fmt.Sprintf("JSONL 第 %d 行不是对象", i+1))
		}
		records = append(records, record)
	}
	return records, nil
}

func parseCSVDataset(data []byte) ([]map[string]any, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	// 容忍列数不一致的行：数据集常有末尾缺列，不值得为此整批失败
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("解析 CSV 失败: %w", err))
	}
	if len(rows) == 0 {
		return nil, apperror.BadRequest("CSV 没有表头行")
	}

	header := rows[0]
	records := make([]map[string]any, 0, len(rows)-1)
	for _, row := range rows[1:] {
		record := make(map[string]any, len(header))
		for j, name := range header {
			name = strings.TrimSpace(name)
			if name == "" || j >= len(row) {
				continue
			}
			record[name] = row[j]
		}
		if len(record) == 0 {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

// toRecord 把 any 断言成 map；不是对象则返回 nil
func toRecord(value any) map[string]any {
	record, _ := value.(map[string]any)
	return record
}

// walkPath 按点号路径取值。优先把整条 path 当作字面量键（数据集里有 "a.b" 这种
// 扁平键的情况不少），命中不了再逐层下钻。任一层不是对象则返回 nil。
func walkPath(root any, path string) any {
	if path == "" {
		return root
	}
	if object, ok := root.(map[string]any); ok {
		if value, ok := object[path]; ok {
			return value
		}
	}

	segment, rest, nested := strings.Cut(path, ".")
	object, ok := root.(map[string]any)
	if !ok {
		return nil
	}
	value, ok := object[segment]
	if !ok {
		return nil
	}
	if !nested {
		return value
	}
	return walkPath(value, rest)
}

// mapDatasetRecord 把一条外部记录映射成 CrawlItem。
// 返回非空 reason 表示该条不可用，由调用方计入跳过统计。
func mapDatasetRecord(record map[string]any, cfg config.DatasetConfig) (CrawlItem, string) {
	title := strings.TrimSpace(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldTitle])))
	if title == "" {
		return CrawlItem{}, ReasonMissingTitle
	}

	category := strings.TrimSpace(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldCategory])))
	if category == "" {
		category = strings.TrimSpace(cfg.DefaultCategory)
	}
	if category == "" {
		return CrawlItem{}, ReasonMissingCategory
	}

	// source_url 是判重的唯一依据，必须能确定下来，否则每次导入都会重复建行
	sourceURL := strings.TrimSpace(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldSourceURL])))
	if sourceURL == "" {
		sourceURL = renderSourceURLTemplate(record, cfg.SourceURLTemplate)
	}
	if sourceURL == "" {
		return CrawlItem{}, ReasonMissingSourceURL
	}

	// 描述缺失时回退到标题：卡片与详情页都直接展示 description，留空会很难看
	description := strings.TrimSpace(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldDescription])))
	if description == "" {
		description = title
	}

	tags := normalizeDatasetTags(
		firstNonEmpty(record, cfg.Fields[config.DatasetFieldTags]),
		cfg.TagSeparatorOrDefault(),
	)

	return CrawlItem{
		Title:       title,
		Description: description,
		CoverURL:    normalizeCoverURL(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldCoverURL]))),
		Author:      strings.TrimSpace(asString(firstNonEmpty(record, cfg.Fields[config.DatasetFieldAuthor]))),
		SourceURL:   sourceURL,
		Category:    category,
		Tags:        tags,
		ViewCount:   normalizeViewCount(firstNonEmpty(record, cfg.Fields[config.DatasetFieldViewCount])),
		Metadata:    buildDatasetMetadata(record, cfg.Metadata),
	}, ""
}

// firstNonEmpty 按候选外部字段顺序取值，返回第一个有内容的结果。
// 空字符串、空数组、空对象都视为「没值」，以便回退到下一个候选字段。
func firstNonEmpty(record map[string]any, paths []string) any {
	for _, path := range paths {
		value := walkPath(record, path)
		if isEmptyDatasetValue(value) {
			continue
		}
		return value
	}
	return nil
}

func isEmptyDatasetValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	}
	return false
}

// asString 把外部值归一成字符串。数字去掉无意义的小数尾零，数组/对象按 JSON 序列化。
func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case json.Number:
		return typed.String()
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

// normalizeCoverURL 归一化封面地址：// 前缀补 https:，http:// 升 https://。
// 与 crawler/collect.py 的 _normalize_cover 保持同一语义。
func normalizeCoverURL(raw string) string {
	text := strings.TrimSpace(raw)
	switch {
	case text == "":
		return ""
	case strings.HasPrefix(text, "//"):
		return "https:" + text
	case strings.HasPrefix(text, "http://"):
		return "https://" + strings.TrimPrefix(text, "http://")
	}
	return text
}

// normalizeDatasetTags 兼容三种形态：JSON 数组、分隔符字符串、null。去重并截断。
func normalizeDatasetTags(value any, separator string) []string {
	var raw []string
	switch typed := value.(type) {
	case nil:
		return []string{}
	case []any:
		raw = make([]string, 0, len(typed))
		for _, entry := range typed {
			raw = append(raw, asString(entry))
		}
	case string:
		raw = strings.Split(typed, separator)
	default:
		raw = strings.Split(asString(typed), separator)
	}

	tags := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, entry := range raw {
		tag := strings.TrimSpace(entry)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
		if len(tags) >= maxDatasetTags {
			break
		}
	}
	return tags
}

// normalizeViewCount 兼容 25612 / "25612" / "25,612" / "2.5万" / "1.2w" / "3k"。
// 识别不了的一律记 0：播放量是展示字段，不值得为它让整条记录失败。
func normalizeViewCount(value any) uint {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		if typed < 0 {
			return 0
		}
		return uint(typed)
	case string:
		text := strings.ReplaceAll(strings.TrimSpace(typed), ",", "")
		if text == "" {
			return 0
		}
		multiplier := 1.0
		switch {
		case strings.HasSuffix(text, "万"):
			multiplier, text = 10000, strings.TrimSuffix(text, "万")
		case strings.HasSuffix(text, "w"), strings.HasSuffix(text, "W"):
			multiplier, text = 10000, text[:len(text)-1]
		case strings.HasSuffix(text, "k"), strings.HasSuffix(text, "K"):
			multiplier, text = 1000, text[:len(text)-1]
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || number < 0 {
			return 0
		}
		return uint(number * multiplier)
	}
	return 0
}

// buildDatasetMetadata 把配置点名的外部字段原样收进 metadata，供将来换模型时取用
func buildDatasetMetadata(record map[string]any, fields []string) map[string]any {
	metadata := make(map[string]any, len(fields))
	for _, field := range fields {
		if value := walkPath(record, field); !isEmptyDatasetValue(value) {
			metadata[field] = value
		}
	}
	return metadata
}

// sourceURLPlaceholder 匹配 source_url_template 里的 {外部字段名}
var sourceURLPlaceholder = regexp.MustCompile(`\{([^{}]+)\}`)

// renderSourceURLTemplate 用记录里的外部字段渲染 source_url 模板。
// 任一占位符取不到值就整体返回空——拼出半截 URL 会让判重失效、每次导入重复建行。
func renderSourceURLTemplate(record map[string]any, template string) string {
	if strings.TrimSpace(template) == "" {
		return ""
	}

	missing := false
	rendered := sourceURLPlaceholder.ReplaceAllStringFunc(template, func(match string) string {
		key := match[1 : len(match)-1]
		value := strings.TrimSpace(asString(walkPath(record, key)))
		if value == "" {
			missing = true
			return ""
		}
		// 模板拼的是 URL 路径片段，值可能含中文或斜杠，按路径段转义
		return url.PathEscape(value)
	})
	if missing {
		return ""
	}
	return rendered
}
