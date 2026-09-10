package service_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, `{
      "items": [{"bvid": "BV1xx411c7mD", "title": "无来源链接", "category": "人工智能"}]
    }`))

	if _, err := svc.Import(); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if got := resources.created[0].SourceURL; got != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Fatalf("SourceURL = %q, want 由 bvid 推导", got)
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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, `{
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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t,
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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, `{"items": []}`))

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if *result != (service.BilibiliImportResult{}) {
		t.Fatalf("stats = %+v, want 全 0", result)
	}
}

func TestBilibiliImportMissingFileReturnsError(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := service.NewBilibiliImportService(resources, categories,
		filepath.Join(t.TempDir(), "not-exist.json"))

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestBilibiliImportMalformedJSONReturnsError(t *testing.T) {
	resources, categories := newBilibiliRepos()
	svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, `{not-json`))

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestBilibiliImportMapsRepoErrors(t *testing.T) {
	boom := errors.New("db error")

	t.Run("FindBySourceURLs", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		resources.findByURLsErr = boom
		svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("Create", func(t *testing.T) {
		resources, categories := newBilibiliRepos()
		resources.createErr = boom
		svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
		svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})

	t.Run("FindByName", func(t *testing.T) {
		resources, _ := newBilibiliRepos()
		categories := &errCategoryRepo{fakeCategoryRepository: &fakeCategoryRepository{}, findErr: boom}
		svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

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
		svc := service.NewBilibiliImportService(resources, categories, writeBilibiliFile(t, oneItemFile))

		_, err := svc.Import()
		assertErrorCode(t, err, apperror.CodeInternal)
	})
}
