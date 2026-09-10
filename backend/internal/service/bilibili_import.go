package service

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// BilibiliImportService 把 backend/crawler 采集的 B 站视频元数据导入资源表。
//
// 视频以 type=video 的普通 Resource 身份入库，与手工录入的资源混在同一列表里；
// 按 source_url 判重，已存在则不新增行、只刷新动态字段，因此可反复执行。
type BilibiliImportService struct {
	resources  repository.ResourceRepository
	categories repository.CategoryRepository
	filePath   string
}

func NewBilibiliImportService(
	resources repository.ResourceRepository,
	categories repository.CategoryRepository,
	filePath string,
) *BilibiliImportService {
	return &BilibiliImportService{resources: resources, categories: categories, filePath: filePath}
}

// BilibiliImportResult 导入统计
type BilibiliImportResult struct {
	CreatedResources  int `json:"created_resources"`
	UpdatedResources  int `json:"updated_resources"`
	SkippedResources  int `json:"skipped_resources"` // 缺必填字段，未落库
	CreatedCategories int `json:"created_categories"`
}

// B 站视频在资源表里的固定类型；model.Resource.Type 允许 course / article / video
const bilibiliResourceType = "video"

// bilibiliSourceTemplate 与 crawler/collect.py 的 SOURCE_URL_TEMPLATE 保持一致
const bilibiliSourceTemplate = "https://www.bilibili.com/video/"

// model.Resource 各字段的 gorm size，超长按字符截断，避免整批导入因单条超长而失败
const (
	maxTitleRunes     = 256
	maxCoverURLRunes  = 512
	maxAuthorRunes    = 128
	maxSourceURLRunes = 512
)

// bilibiliFile 是 crawler/run.py 写出的交接文件结构
type bilibiliFile struct {
	Version int            `json:"version"`
	Source  string         `json:"source"`
	Items   []bilibiliItem `json:"items"`
}

type bilibiliItem struct {
	Bvid        string         `json:"bvid"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	CoverURL    string         `json:"cover_url"`
	Author      string         `json:"author"`
	SourceURL   string         `json:"source_url"`
	Category    string         `json:"category"`
	Tags        []string       `json:"tags"`
	ViewCount   uint           `json:"view_count"`
	Metadata    map[string]any `json:"metadata"`
}

// bilibiliRecord 是校验、截断、序列化之后的待落库记录
type bilibiliRecord struct {
	title        string
	description  string
	coverURL     string
	author       string
	sourceURL    string
	category     string
	tagsJSON     string
	metadataJSON string
	viewCount    uint
}

// Import 读取爬虫输出文件并落库
func (s *BilibiliImportService) Import() (*BilibiliImportResult, error) {
	return s.run(true)
}

// Preview 走与 Import 完全相同的解析、判重与分类逻辑，但不写库，用于 -dry-run
func (s *BilibiliImportService) Preview() (*BilibiliImportResult, error) {
	return s.run(false)
}

func (s *BilibiliImportService) run(write bool) (*BilibiliImportResult, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	var file bilibiliFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, apperror.Internal(err)
	}

	result := &BilibiliImportResult{}
	records := make([]bilibiliRecord, 0, len(file.Items))
	for _, item := range file.Items {
		record, ok := newBilibiliRecord(item)
		if !ok {
			result.SkippedResources++
			continue
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		return result, nil
	}

	// 一次性查出已存在的来源链接，避免逐条查询
	urls := make([]string, 0, len(records))
	for i := range records {
		urls = append(urls, records[i].sourceURL)
	}
	existing, err := s.resources.FindBySourceURLs(urls)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	existingByURL := make(map[string]*model.Resource, len(existing))
	for i := range existing {
		existingByURL[existing[i].SourceURL] = &existing[i]
	}

	categoryIDs := map[string]uint{}
	for i := range records {
		record := &records[i]

		categoryID, err := s.resolveCategory(record.category, categoryIDs, result, write)
		if err != nil {
			return nil, err
		}

		if current, ok := existingByURL[record.sourceURL]; ok {
			// 已存在：不新增行，只刷新动态字段（播放量、扩展信息）
			current.ViewCount = record.viewCount
			current.Metadata = record.metadataJSON
			if write {
				if err := s.resources.Update(current); err != nil {
					return nil, apperror.Internal(err)
				}
			}
			result.UpdatedResources++
			continue
		}

		resource := record.toResource(categoryID)
		if write {
			if err := s.resources.Create(resource); err != nil {
				return nil, apperror.Internal(err)
			}
		}
		// 文件内若出现重复的 source_url，后续条目按「已存在」处理
		existingByURL[record.sourceURL] = resource
		result.CreatedResources++
	}

	return result, nil
}

// resolveCategory 按名称取分类 ID，不存在则创建；同名分类在单次导入内只查一次
func (s *BilibiliImportService) resolveCategory(
	name string,
	cache map[string]uint,
	result *BilibiliImportResult,
	write bool,
) (uint, error) {
	if id, ok := cache[name]; ok {
		return id, nil
	}

	category, err := s.categories.FindByName(name)
	if err == nil {
		cache[name] = category.ID
		return category.ID, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, apperror.Internal(err)
	}

	if !write {
		cache[name] = 0
		result.CreatedCategories++
		return 0, nil
	}

	created := &model.Category{Name: name}
	if err := s.categories.Create(created); err != nil {
		return 0, apperror.Internal(err)
	}
	cache[name] = created.ID
	result.CreatedCategories++
	return created.ID, nil
}

// newBilibiliRecord 校验必填字段并归一化；缺字段返回 ok=false，由调用方计入跳过
func newBilibiliRecord(item bilibiliItem) (bilibiliRecord, bool) {
	title := strings.TrimSpace(item.Title)
	bvid := strings.TrimSpace(item.Bvid)
	category := strings.TrimSpace(item.Category)
	if title == "" || bvid == "" || category == "" {
		return bilibiliRecord{}, false
	}

	sourceURL := strings.TrimSpace(item.SourceURL)
	if sourceURL == "" {
		sourceURL = bilibiliSourceTemplate + bvid
	}

	// Tags 列是 JSON 数组，nil 会被 Marshal 成 null，这里兜底成 []
	if item.Tags == nil {
		item.Tags = []string{}
	}
	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return bilibiliRecord{}, false
	}

	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return bilibiliRecord{}, false
	}

	return bilibiliRecord{
		title:        truncateRunes(title, maxTitleRunes),
		description:  strings.TrimSpace(item.Description),
		coverURL:     truncateRunes(strings.TrimSpace(item.CoverURL), maxCoverURLRunes),
		author:       truncateRunes(strings.TrimSpace(item.Author), maxAuthorRunes),
		sourceURL:    truncateRunes(sourceURL, maxSourceURLRunes),
		category:     category,
		tagsJSON:     string(tagsJSON),
		metadataJSON: string(metadataJSON),
		viewCount:    item.ViewCount,
	}, true
}

func (r bilibiliRecord) toResource(categoryID uint) *model.Resource {
	return &model.Resource{
		Title:       r.title,
		Description: r.description,
		CoverURL:    r.coverURL,
		Type:        bilibiliResourceType,
		CategoryID:  categoryID,
		Tags:        r.tagsJSON,
		Metadata:    r.metadataJSON,
		Author:      r.author,
		SourceURL:   r.sourceURL,
		ViewCount:   r.viewCount,
	}
}

// truncateRunes 按字符截断（而非字节），与 MySQL varchar 的长度语义一致
func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
