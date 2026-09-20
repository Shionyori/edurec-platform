package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// writeDataset 把内容写成临时数据集文件；扩展名决定 format=auto 的判定结果
func writeDataset(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写临时数据集失败: %v", err)
	}
	return path
}

// moocDataset 一份把慕课风格字段映射到平台契约的配置（对应 configs/config.yaml 的 datasets.mooc）
func moocDataset(path string) config.DatasetConfig {
	return config.DatasetConfig{
		File:            path,
		ResourceType:    "course",
		DefaultCategory: "慕课课程",
		Fields: map[string][]string{
			config.DatasetFieldTitle:       {"title", "name"},
			config.DatasetFieldDescription: {"short_description", "description"},
			config.DatasetFieldCoverURL:    {"pic"},
			config.DatasetFieldAuthor:      {"author_nickname"},
			config.DatasetFieldSourceURL:   {"target_url"},
			config.DatasetFieldCategory:    {"category"},
			config.DatasetFieldTags:        {"tags"},
			config.DatasetFieldViewCount:   {"numbers"},
		},
		Metadata: []string{"obj_id", "price"},
	}
}

func TestLoadDatasetMapsFieldsAndMetadata(t *testing.T) {
	path := writeDataset(t, "courses.json", `{
      "data": {"list": [
        {"obj_id": 1273, "title": "Python 工程师进阶", "short_description": "面向零基础",
         "pic": "//img1.sycdn.imooc.com/a.jpg", "author_nickname": "雨衡",
         "target_url": "https://www.imooc.com/learn/1273", "numbers": "25612",
         "tags": ["零基础", "Python"], "price": 0.0}
      ]}
    }`)
	cfg := moocDataset(path)
	cfg.ItemsPath = "data.list"

	result, err := service.LoadDataset(cfg)

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Skipped() != 0 {
		t.Fatalf("result = %+v, want 1 条全部映射成功", result)
	}

	got := result.Items[0]
	if got.Title != "Python 工程师进阶" || got.Description != "面向零基础" {
		t.Fatalf("Title/Description = %q/%q", got.Title, got.Description)
	}
	// // 前缀补 https:，与 collect.py 的封面归一化语义一致
	if got.CoverURL != "https://img1.sycdn.imooc.com/a.jpg" {
		t.Fatalf("CoverURL = %q, want 补全 https:", got.CoverURL)
	}
	if got.Author != "雨衡" || got.SourceURL != "https://www.imooc.com/learn/1273" {
		t.Fatalf("Author/SourceURL = %q/%q", got.Author, got.SourceURL)
	}
	if got.Category != "慕课课程" {
		t.Fatalf("Category = %q, want 落到 default_category", got.Category)
	}
	if got.ViewCount != 25612 {
		t.Fatalf("ViewCount = %d, want 25612", got.ViewCount)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "零基础" {
		t.Fatalf("Tags = %v", got.Tags)
	}
	if got.Metadata["obj_id"] != float64(1273) {
		t.Fatalf("Metadata = %v, want 收进 obj_id", got.Metadata)
	}
	if _, ok := got.Metadata["target_url"]; ok {
		t.Fatalf("Metadata = %v, 只应收录配置点名的字段", got.Metadata)
	}
}

func TestLoadDatasetFallsBackThroughFieldCandidates(t *testing.T) {
	// title 存在但为空 → 回退到 name；author 走点号路径
	path := writeDataset(t, "courses.json", `[
      {"title": "   ", "name": "备用标题", "target_url": "https://x/1",
       "author": {"nickname": "嵌套作者"}}
    ]`)
	cfg := moocDataset(path)
	cfg.Fields[config.DatasetFieldAuthor] = []string{"author_nickname", "author.nickname"}

	result, err := service.LoadDataset(cfg)

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if got := result.Items[0].Title; got != "备用标题" {
		t.Fatalf("Title = %q, want 回退到 name 候选", got)
	}
	if got := result.Items[0].Author; got != "嵌套作者" {
		t.Fatalf("Author = %q, want 走点号路径取嵌套字段", got)
	}
}

func TestLoadDatasetDescriptionFallsBackToTitle(t *testing.T) {
	path := writeDataset(t, "courses.json", `[
      {"title": "只有标题", "target_url": "https://x/1"}
    ]`)

	result, err := service.LoadDataset(moocDataset(path))

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	// 卡片与详情页直接展示 description，留空会很难看
	if got := result.Items[0].Description; got != "只有标题" {
		t.Fatalf("Description = %q, want 回退到标题", got)
	}
}

func TestLoadDatasetRendersSourceURLTemplate(t *testing.T) {
	// 数据集没有 URL 字段，但有课程 id：用模板拼出 source_url
	path := writeDataset(t, "courses.json", `[
      {"title": "课程甲", "obj_id": 998}
    ]`)
	cfg := moocDataset(path)
	cfg.Fields[config.DatasetFieldSourceURL] = nil
	cfg.SourceURLTemplate = "https://www.imooc.com/learn/{obj_id}"

	result, err := service.LoadDataset(cfg)

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if got := result.Items[0].SourceURL; got != "https://www.imooc.com/learn/998" {
		t.Fatalf("SourceURL = %q, want 由模板渲染", got)
	}
}

func TestLoadDatasetSkipsRowsWithClearReasons(t *testing.T) {
	// 三条各缺一样必需信息，且只缺一样——校验按 标题→分类→来源链接 的顺序报第一个缺失项，
	// 因此每条都要补齐其余字段，否则报出的原因不唯一、无法断言
	path := writeDataset(t, "courses.json", `[
      {"title": "   ", "category": "慕课课程", "target_url": "https://x/1"},
      {"title": "缺链接也没模板", "category": "慕课课程"},
      {"title": "缺分类", "target_url": "https://x/3"}
    ]`)
	cfg := moocDataset(path)
	cfg.DefaultCategory = "" // 去掉兜底分类，让第 3 条因缺分类被跳过

	result, err := service.LoadDataset(cfg)

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if result.Total != 3 || len(result.Items) != 0 {
		t.Fatalf("result = %+v, want 3 条全部跳过", result)
	}
	for _, reason := range []string{
		service.ReasonMissingTitle,
		service.ReasonMissingSourceURL,
		service.ReasonMissingCategory,
	} {
		if result.SkipReasons[reason] != 1 {
			t.Fatalf("SkipReasons = %v, want 每个原因各 1 条（缺 %q）", result.SkipReasons, reason)
		}
	}
}

func TestLoadDatasetParsesJSONL(t *testing.T) {
	path := writeDataset(t, "courses.jsonl", `{"title": "第一条", "target_url": "https://x/1"}
{"title": "第二条", "target_url": "https://x/2"}

`)

	result, err := service.LoadDataset(moocDataset(path))

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("result = %+v, want 解析 2 行且跳过空行", result)
	}
	if result.Items[1].Title != "第二条" {
		t.Fatalf("Items[1].Title = %q", result.Items[1].Title)
	}
}

func TestLoadDatasetParsesCSVWithBOM(t *testing.T) {
	// Windows 导出的 CSV 常带 UTF-8 BOM；不剥掉首列名会多出 U+FEFF 而映射不上
	path := writeDataset(t, "courses.csv", "\ufeff"+
		"title,target_url,author_nickname,numbers,tags\n"+
		"数据分析入门,https://x/csv1,李老师,\"1.2万\",\"入门,统计\"\n")

	result, err := service.LoadDataset(moocDataset(path))

	if err != nil {
		t.Fatalf("LoadDataset() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = %+v, want 1 条（BOM 不应导致首列映射失败）", result)
	}

	got := result.Items[0]
	if got.Title != "数据分析入门" {
		t.Fatalf("Title = %q, want 剥掉 BOM 后取到首列", got.Title)
	}
	if got.ViewCount != 12000 {
		t.Fatalf("ViewCount = %d, want 1.2万 → 12000", got.ViewCount)
	}
	if len(got.Tags) != 2 || got.Tags[1] != "统计" {
		t.Fatalf("Tags = %v, want 按逗号拆成两个", got.Tags)
	}
}

func TestNormalizeViewCountVariants(t *testing.T) {
	cases := []struct {
		raw  string
		want uint
	}{
		{`"25612"`, 25612},
		{`"25,612"`, 25612},
		{`"2.5万"`, 25000},
		{`"1.2w"`, 12000},
		{`"3k"`, 3000},
		{`25612`, 25612},
		{`"暂无"`, 0},
		{`""`, 0},
		{`null`, 0},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			path := writeDataset(t, "courses.json",
				`[{"title": "课程", "target_url": "https://x/1", "numbers": `+tc.raw+`}]`)

			result, err := service.LoadDataset(moocDataset(path))
			if err != nil {
				t.Fatalf("LoadDataset() error = %v", err)
			}
			if got := result.Items[0].ViewCount; got != tc.want {
				t.Fatalf("ViewCount = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNormalizeDatasetTagsVariants(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"数组", `["A", "B", "A"]`, []string{"A", "B"}},
		{"逗号分隔字符串", `"A,B"`, []string{"A", "B"}},
		{"null", `null`, []string{}},
		{"空字符串", `""`, []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeDataset(t, "courses.json",
				`[{"title": "课程", "target_url": "https://x/1", "tags": `+tc.raw+`}]`)

			result, err := service.LoadDataset(moocDataset(path))
			if err != nil {
				t.Fatalf("LoadDataset() error = %v", err)
			}
			got := result.Items[0].Tags
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Fatalf("Tags = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLoadDatasetRejectsBadItemsPath(t *testing.T) {
	path := writeDataset(t, "courses.json", `{"data": {"list": []}}`)
	cfg := moocDataset(path)
	cfg.ItemsPath = "data.missing"

	_, err := service.LoadDataset(cfg)

	// items_path 指错时必须报错，不能静默返回 0 条让人以为数据集是空的
	if err == nil {
		t.Fatal("items_path 指向不存在的节点，期望报错")
	}
}

func TestLoadDatasetMissingFileReturnsError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-exist.json")
	cfg := moocDataset(missing)

	_, err := service.LoadDataset(cfg)

	if err == nil {
		t.Fatal("文件不存在，期望报错")
	}
	// 路径配错是最常见的失败，报错必须点名是哪个文件，否则无从下手
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error = %v, want 带上出错的文件路径", err)
	}
}
