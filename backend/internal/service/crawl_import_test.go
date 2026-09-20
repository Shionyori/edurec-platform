package service_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

// newCrawlImportService 按当前构造签名建服务。
// 不传 courseRules → 零值经 CourseRulesOrDefault() 回退到默认归类规则
//（标题带课程/合集标记且时长 ≥ 120 分钟判为 course），与生产配置一致。
func newCrawlImportService(
	resources *recordingResourceRepo,
	categories repository.CategoryRepository,
	filePath string,
	allowedTypenames ...[]string,
) *service.CrawlImportService {
	var typed []string
	if len(allowedTypenames) > 0 {
		typed = allowedTypenames[0]
	}
	return service.NewCrawlImportService(resources, categories, filePath, typed, config.ContentRulesConfig{})
}

// recordingResourceRepo 复用完整 fake，额外记录全部写入，便于断言字段映射
type recordingResourceRepo struct {
	*fakeResourceRepository
	created []*model.Resource
	updated []*model.Resource
}

func (f *recordingResourceRepo) Create(resource *model.Resource) error {
	if err := f.fakeResourceRepository.Create(resource); err != nil {
		return err
	}
	f.created = append(f.created, resource)
	return nil
}

func (f *recordingResourceRepo) Update(resource *model.Resource) error {
	if err := f.fakeResourceRepository.Update(resource); err != nil {
		return err
	}
	f.updated = append(f.updated, resource)
	return nil
}

type errCategoryRepo struct {
	*fakeCategoryRepository
	findErr   error
	createErr error
}

func (f *errCategoryRepo) FindByName(string) (*model.Category, error) {
	return nil, f.findErr
}

func (f *errCategoryRepo) Create(category *model.Category) error {
	return f.createErr
}

func newBilibiliRepos() (*recordingResourceRepo, *fakeCategoryRepository) {
	return &recordingResourceRepo{fakeResourceRepository: &fakeResourceRepository{}},
		&fakeCategoryRepository{}
}

func writeBilibiliFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "latest.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}
	return path
}

const oneItemFile = `{
  "version": 1,
  "source": "bilibili",
  "items": [
    {
      "bvid": "BV1DgxCzREbM",
      "title": "机器学习入门",
      "description": "面向零基础的机器学习导论",
      "cover_url": "https://i1.hdslb.com/bfs/archive/cover.jpg",
      "author": "AI精品课程",
      "source_url": "https://www.bilibili.com/video/BV1DgxCzREbM",
      "category": "人工智能",
      "tags": ["机器学习", "AI"],
      "view_count": 170306,
      "metadata": {"bvid": "BV1DgxCzREbM", "duration": "16:08", "like": 4757}
    }
  ]
}`

func TestBilibiliImportCreatesResource(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedResources != 1 || result.SkippedResources != 0 {
		t.Fatalf("stats = %+v, want 1 created / 0 skipped", result)
	}
	if result.CreatedCategories != 1 {
		t.Fatalf("CreatedCategories = %d, want 1", result.CreatedCategories)
	}
	if len(resources.created) != 1 {
		t.Fatalf("created = %d 条, want 1", len(resources.created))
	}

	got := resources.created[0]
	if got.Type != "video" {
		t.Fatalf("Type = %q, want video", got.Type)
	}
	if got.Title != "机器学习入门" || got.Author != "AI精品课程" {
		t.Fatalf("Title/Author = %q/%q", got.Title, got.Author)
	}
	if got.CoverURL != "https://i1.hdslb.com/bfs/archive/cover.jpg" {
		t.Fatalf("CoverURL = %q", got.CoverURL)
	}
	if got.SourceURL != "https://www.bilibili.com/video/BV1DgxCzREbM" {
		t.Fatalf("SourceURL = %q", got.SourceURL)
	}
	if got.ViewCount != 170306 {
		t.Fatalf("ViewCount = %d, want 170306", got.ViewCount)
	}
	if got.CategoryID != 1 {
		t.Fatalf("CategoryID = %d, want 1", got.CategoryID)
	}
	if got.Tags != `["机器学习","AI"]` {
		t.Fatalf("Tags = %s", got.Tags)
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(got.Metadata), &metadata); err != nil {
		t.Fatalf("Metadata 不是合法 JSON: %v (%s)", err, got.Metadata)
	}
	if metadata["duration"] != "16:08" || metadata["like"] != float64(4757) {
		t.Fatalf("Metadata = %v", metadata)
	}
}

func TestBilibiliImportDerivesSourceURLFromBvid(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{
      "items": [{"bvid": "BV1xx411c7mD", "title": "无来源链接", "category": "人工智能"}]
    }`))

	if _, err := svc.Import(); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := resources.created[0].SourceURL; got != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Fatalf("SourceURL = %q, want 由 bvid 推导", got)
	}
}

// 非 B 站来源（数据集导入）用 source_id 作身份键、不透传 bvid，
// 且落库 type 与 URL 模板都由调用方通过 CrawlImportOptions 指定。
func TestCrawlImportAcceptsSourceIDAndCustomOptions(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, "")

	result, imported, err := svc.ImportItems([]service.CrawlItem{{
		SourceID:  "1273",
		Title:     "Python 工程师进阶",
		Category:  "慕课课程",
		Tags:      []string{"零基础"},
		ViewCount: 25612,
	}}, service.CrawlImportOptions{
		ResourceType:      "course",
		SourceURLTemplate: "https://www.imooc.com/learn/",
	}, true)

	if err != nil {
		t.Fatalf("ImportItems() error = %v", err)
	}
	if result.CreatedResources != 1 || result.SkippedResources != 0 {
		t.Fatalf("stats = %+v, want 1 created / 0 skipped", result)
	}
	if len(imported) != 1 {
		t.Fatalf("imported = %d 条, want 1", len(imported))
	}

	got := resources.created[0]
	if got.Type != "course" {
		t.Fatalf("Type = %q, want course（由 options 指定，不再写死 video）", got.Type)
	}
	if got.SourceURL != "https://www.imooc.com/learn/1273" {
		t.Fatalf("SourceURL = %q, want 由 source_id + 模板推导", got.SourceURL)
	}
}

// source_id 与 bvid 都为空时无法判重，必须跳过而非插入（否则每次导入都会重复建行）
func TestCrawlImportSkipsItemWithoutIdentity(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, "")

	result, imported, err := svc.ImportItems([]service.CrawlItem{{
		Title:    "没有身份键",
		Category: "慕课课程",
	}}, service.CrawlImportOptions{
		ResourceType:      "course",
		SourceURLTemplate: "https://www.imooc.com/learn/",
	}, true)

	if err != nil {
		t.Fatalf("ImportItems() error = %v", err)
	}
	if result.SkippedResources != 1 || result.CreatedResources != 0 {
		t.Fatalf("stats = %+v, want 1 skipped / 0 created", result)
	}
	if len(imported) != 0 || len(resources.created) != 0 {
		t.Fatal("无身份键的条目不应落库")
	}
}

func TestBilibiliImportRefreshesExistingResource(t *testing.T) {
	resources, categories := newBilibiliRepos()
	resources.findByURLsResult = []model.Resource{{
		Model:     gorm.Model{ID: 7},
		Title:     "旧标题",
		SourceURL: "https://www.bilibili.com/video/BV1DgxCzREbM",
		ViewCount: 1,
		Metadata:  "{}",
	}}
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.UpdatedResources != 1 || result.CreatedResources != 0 {
		t.Fatalf("stats = %+v, want 1 updated / 0 created", result)
	}
	if len(resources.created) != 0 {
		t.Fatalf("created = %d 条, want 0（判重命中不应新增行）", len(resources.created))
	}
	if len(resources.updated) != 1 {
		t.Fatalf("updated = %d 条, want 1", len(resources.updated))
	}

	got := resources.updated[0]
	if got.ID != 7 {
		t.Fatalf("更新了 id=%d, want 7", got.ID)
	}
	if got.ViewCount != 170306 {
		t.Fatalf("ViewCount = %d, want 刷新为 170306", got.ViewCount)
	}
	if !strings.Contains(got.Metadata, `"duration":"16:08"`) {
		t.Fatalf("Metadata = %s, want 刷新为新的扩展信息", got.Metadata)
	}
	if got.Title != "旧标题" {
		t.Fatalf("Title = %q, want 保持原值（只刷新动态字段）", got.Title)
	}
}

func TestBilibiliImportSkipsInvalidItems(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{
      "items": [
        {"bvid": "BV1", "title": "缺少分类", "category": ""},
        {"bvid": "BV2", "title": "  ", "category": "人工智能"},
        {"bvid": "", "title": "缺少 bvid", "category": "人工智能"}
      ]
    }`))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.SkippedResources != 3 || result.CreatedResources != 0 {
		t.Fatalf("stats = %+v, want 3 skipped / 0 created", result)
	}
	if len(resources.created) != 0 {
		t.Fatalf("created = %d 条, want 0", len(resources.created))
	}
}

func TestBilibiliImportReusesExistingCategory(t *testing.T) {
	resources, categories := newBilibiliRepos()
	categories.categories = []model.Category{{Model: gorm.Model{ID: 42}, Name: "人工智能"}}
	categories.nextID = 42
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedCategories != 0 {
		t.Fatalf("CreatedCategories = %d, want 0（分类已存在）", result.CreatedCategories)
	}
	if resources.created[0].CategoryID != 42 {
		t.Fatalf("CategoryID = %d, want 42", resources.created[0].CategoryID)
	}
}

func TestBilibiliImportTruncatesLongFields(t *testing.T) {
	resources, categories := newBilibiliRepos()
	long := strings.Repeat("测", 300)
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t,
		`{"items": [{"bvid": "BV1", "title": "`+long+`", "author": "`+long+`", "category": "人工智能"}]}`))

	if _, err := svc.Import(); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	got := resources.created[0]
	if n := len([]rune(got.Title)); n != 256 {
		t.Fatalf("Title 长度 = %d, want 256", n)
	}
	if n := len([]rune(got.Author)); n != 128 {
		t.Fatalf("Author 长度 = %d, want 128", n)
	}
}

func TestBilibiliImportPreviewDoesNotWrite(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

	result, err := svc.Preview()

	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if result.CreatedResources != 1 || result.CreatedCategories != 1 {
		t.Fatalf("stats = %+v, want 与 Import 相同的统计", result)
	}
	if len(resources.created) != 0 || len(resources.updated) != 0 || len(categories.categories) != 0 {
		t.Fatal("Preview() 不应写库")
	}
}

func TestBilibiliImportEmptyFileReturnsEmptyResult(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{"items": []}`))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if *result != (service.CrawlImportResult{}) {
		t.Fatalf("stats = %+v, want 全 0", result)
	}
}

func TestBilibiliImportMissingFileReturnsError(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories,
		filepath.Join(t.TempDir(), "not-exist.json"))

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestBilibiliImportMalformedJSONReturnsError(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{not-json`))

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestBilibiliImportMapsRepoErrors(t *testing.T) {
	boom := errors.New("db error")

	t.Run("FindBySourceURLs", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		resources.findByURLsErr = boom
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("Create", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		resources.createErr = boom
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("Update", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		resources.findByURLsResult = []model.Resource{{
			Model:     gorm.Model{ID: 7},
			SourceURL: "https://www.bilibili.com/video/BV1DgxCzREbM",
		}}
		resources.updateErr = boom
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("FindByName", func(t *testing.T) {
		resources, _ := newBilibiliRepos()
		categories := &errCategoryRepo{fakeCategoryRepository: &fakeCategoryRepository{}, findErr: boom}
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("CategoryCreate", func(t *testing.T) {
		resources, _ := newBilibiliRepos()
		categories := &errCategoryRepo{
			fakeCategoryRepository: &fakeCategoryRepository{},
			findErr:                errors.New("not found"),
			createErr:              boom,
		}
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})
}

// 分区白名单：B 站搜索结果会带上非教育分区（影视/娱乐/游戏等），必须在落库前拦掉
func TestCrawlImportRejectsDisallowedTypenames(t *testing.T) {
	file := `{
  "version": 1,
  "source": "bilibili",
  "items": [
    {"bvid": "BVedu", "title": "高等数学 全程教学", "source_url": "https://www.bilibili.com/video/BVedu",
     "category": "B站视频", "metadata": {"typename": "校园学习"}},
    {"bvid": "BVdrama", "title": "杨真真黑化复仇", "source_url": "https://www.bilibili.com/video/BVdrama",
     "category": "B站视频", "metadata": {"typename": "影视剪辑"}},
    {"bvid": "BVgame", "title": "某游戏实况", "source_url": "https://www.bilibili.com/video/BVgame",
     "category": "B站视频", "metadata": {"typename": "网络游戏"}}
  ]
}`

	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, file), []string{"校园学习"})

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedResources != 1 {
		t.Fatalf("CreatedResources = %d, want 1（只有校园学习应入库）", result.CreatedResources)
	}
	if result.SkippedTypenames != 2 {
		t.Fatalf("SkippedTypenames = %d, want 2（影视剪辑 + 网络游戏）", result.SkippedTypenames)
	}
	if len(resources.created) != 1 || resources.created[0].Title != "高等数学 全程教学" {
		t.Fatalf("created = %+v, want 仅高等数学", resources.created)
	}
}

// 白名单为空 = 不过滤（保持既有行为）；大小写与前后空格应被忽略
func TestCrawlImportTypenameAllowlistEdgeCases(t *testing.T) {
	t.Run("白名单为空时不过滤", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{
  "version": 1, "source": "bilibili", "items": [
    {"bvid": "BVdrama", "title": "影视剪辑内容", "source_url": "https://www.bilibili.com/video/BVdrama",
     "category": "B站视频", "metadata": {"typename": "影视剪辑"}}
  ]}`))

		result, err := svc.Import()
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}
		if result.CreatedResources != 1 || result.SkippedTypenames != 0 {
			t.Fatalf("stats = %+v, want 1 created / 0 typename-skipped", result)
		}
	})

	t.Run("大小写与空格无关", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{
  "version": 1, "source": "bilibili", "items": [
    {"bvid": "BVx", "title": "科技区内容", "source_url": "https://www.bilibili.com/video/BVx",
     "category": "B站视频", "metadata": {"typename": "  科学科普  "}}
  ]}`), []string{" 科学科普 "})

		result, err := svc.Import()
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}
		if result.CreatedResources != 1 || result.SkippedTypenames != 0 {
			t.Fatalf("stats = %+v, want 1 created（空白与大小写应被忽略）", result)
		}
	})

	t.Run("条目没有 typename 时放行（第三方数据集无此字段）", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, `{
  "version": 1, "source": "mooc", "items": [
    {"source_id": "course-1", "title": "数据结构", "source_url": "https://example.com/c/1",
     "category": "慕课课程", "metadata": {"obj_id": "c1"}}
  ]}`), []string{"校园学习"})

		result, err := svc.Import()
		if err != nil {
			t.Fatalf("Import() error = %v", err)
		}
		if result.CreatedResources != 1 || result.SkippedTypenames != 0 {
			t.Fatalf("stats = %+v, want 1 created（无 typename 不应被分区过滤拦下）", result)
		}
	})
}

// 导入时按规则自动判类型：B 站长合集落 course，短片仍落 video。
// 这条保证「每次新内容进来都会判一次」，不用事后跑迁移脚本。
func TestCrawlImportInfersCourseTypeForLongCompilations(t *testing.T) {
	file := `{
  "version": 1,
  "source": "bilibili",
  "items": [
    {"bvid": "BVlong", "title": "数学分析（第三版）-复旦大学-陈纪修教授(全198集)",
     "source_url": "https://www.bilibili.com/video/BVlong", "category": "B站视频",
     "metadata": {"typename": "校园学习", "duration": "9003:20"}},
    {"bvid": "BVshort", "title": "17分钟看懂所有机器学习算法",
     "source_url": "https://www.bilibili.com/video/BVshort", "category": "B站视频",
     "metadata": {"typename": "校园学习", "duration": "17:30"}},
    {"bvid": "BVplain", "title": "2026数学建模国赛B题可视化",
     "source_url": "https://www.bilibili.com/video/BVplain", "category": "B站视频",
     "metadata": {"typename": "校园学习", "duration": "5000:00"}}
  ]
}`

	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, file), []string{"校园学习"})

	if _, err := svc.Import(); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(resources.created) != 3 {
		t.Fatalf("created = %d 条, want 3", len(resources.created))
	}

	byTitle := map[string]string{}
	for _, r := range resources.created {
		byTitle[r.Title] = r.Type
	}

	if got := byTitle["数学分析（第三版）-复旦大学-陈纪修教授(全198集)"]; got != "course" {
		t.Errorf("长合集 Type = %q, want course（含集数标注且时长 9003 分钟）", got)
	}
	if got := byTitle["17分钟看懂所有机器学习算法"]; got != "video" {
		t.Errorf("短片 Type = %q, want video（时长不足阈值，标题也不含课程/合集标记）", got)
	}
	// 长但没有任何课程/合集标记：不归入课程，避免把长直播录像也算成课
	if got := byTitle["2026数学建模国赛B题可视化"]; got != "video" {
		t.Errorf("无标记长视频 Type = %q, want video", got)
	}
}

// 非 B 站来源（无 typename，如第三方数据集）不参与「长合集」判定，沿用 opts.ResourceType
func TestCrawlImportDoesNotInferTypeForNonBilibili(t *testing.T) {
	file := `{
  "version": 1, "source": "mooc", "items": [
    {"source_id": "c1", "title": "高等数学（全200集）课程讲解", "source_url": "https://example.com/c/1",
     "category": "慕课课程", "metadata": {"obj_id": "c1", "duration": "9003:20"}}
  ]
}`

	resources, categories := newBilibiliRepos()
	svc := newCrawlImportService(resources, categories, writeBilibiliFile(t, file))

	if _, err := svc.Import(); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(resources.created) != 1 {
		t.Fatalf("created = %d 条, want 1", len(resources.created))
	}
	// 即使标题与时长都像长课程，数据集来源仍按自身声明的 type 落库
	if got := resources.created[0].Type; got != "video" {
		t.Errorf("Type = %q, want video（非 B 站来源不做长合集判定）", got)
	}
}
