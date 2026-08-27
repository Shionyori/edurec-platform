package service_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

// errUserRepo / errResourceRepo / errRecRepo 复用完整 fake，仅覆盖单个方法以注入错误
type errUserRepo struct {
	*fakeUserRepository
	err error
}

func (f *errUserRepo) FindByIDs([]uint) ([]model.User, error) {
	return nil, f.err
}

type errResourceRepo struct {
	*fakeResourceRepository
	err error
}

func (f *errResourceRepo) FindByIDs([]uint) ([]model.Resource, error) {
	return nil, f.err
}

type errRecRepo struct {
	*fakeRecommendationRepository
	err error
}

func (f *errRecRepo) Replace(uint, string, int64) (*model.Recommendation, error) {
	return nil, f.err
}

func newFakeUsers(ids ...uint) *fakeUserRepository {
	f := &fakeUserRepository{}
	for _, id := range ids {
		f.users = append(f.users, &model.User{Model: gorm.Model{ID: id}})
	}
	return f
}

func newFakeResources(ids ...uint) *fakeResourceRepository {
	f := &fakeResourceRepository{}
	for _, id := range ids {
		f.findByIDsResult = append(f.findByIDsResult, model.Resource{Model: gorm.Model{ID: id}})
	}
	return f
}

func writeRecommendations(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recommendations.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写临时文件失败: %v", err)
	}
	return path
}

func TestImportFiltersAndWritesMatchingData(t *testing.T) {
	recs := &fakeRecommendationRepository{}
	svc := service.NewRecommendationImportService(
		recs,
		newFakeUsers(1, 3),
		newFakeResources(101, 201),
		writeRecommendations(t, `{"1": [101, 999], "3": [201], "99": [101]}`),
	)

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedUsers != 2 {
		t.Fatalf("ImportedUsers = %d, want 2", result.ImportedUsers)
	}
	if result.SkippedUsers != 1 {
		t.Fatalf("SkippedUsers = %d, want 1 (user 99)", result.SkippedUsers)
	}
	if result.ImportedResources != 2 {
		t.Fatalf("ImportedResources = %d, want 2", result.ImportedResources)
	}
	if result.SkippedResources != 1 {
		t.Fatalf("SkippedResources = %d, want 1 (resource 999)", result.SkippedResources)
	}
	// 用户 1、3 各写入过滤后的资源列表（map 遍历顺序不定，按集合断言）
	got := map[uint]string{}
	for _, c := range recs.replaceCalls {
		got[c.userID] = c.ids
	}
	if len(recs.replaceCalls) != 2 || got[1] != `[101]` || got[3] != `[201]` {
		t.Fatalf("Replace() calls = %v, want user1=[101] user3=[201]", got)
	}
}

func TestImportWritesPerUser(t *testing.T) {
	recs := &fakeRecommendationRepository{}
	svc := service.NewRecommendationImportService(
		recs,
		newFakeUsers(1, 2),
		newFakeResources(10, 20, 30),
		writeRecommendations(t, `{"1": [10, 30], "2": [20]}`),
	)

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedUsers != 2 || result.ImportedResources != 3 {
		t.Fatalf("stats = %+v, want 2 users / 3 resources", result)
	}
}

func TestImportSkipsUserWithNoValidResources(t *testing.T) {
	recs := &fakeRecommendationRepository{}
	svc := service.NewRecommendationImportService(
		recs,
		newFakeUsers(1),
		newFakeResources(),
		writeRecommendations(t, `{"1": [999]}`),
	)

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedUsers != 0 || result.SkippedUsers != 1 || result.SkippedResources != 1 {
		t.Fatalf("stats = %+v, want all skipped", result)
	}
	if recs.replaceUserID != 0 {
		t.Fatal("Import() should not write cache when no valid resource")
	}
}

func TestImportEmptyFileReturnsEmptyResult(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&fakeRecommendationRepository{},
		newFakeUsers(),
		newFakeResources(),
		writeRecommendations(t, `{}`),
	)

	result, err := svc.Import()

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedUsers != 0 || result.SkippedUsers != 0 {
		t.Fatalf("stats = %+v, want all zero", result)
	}
}

func TestImportMissingFileReturnsError(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&fakeRecommendationRepository{},
		newFakeUsers(),
		newFakeResources(),
		filepath.Join(t.TempDir(), "not-exist.json"),
	)

	_, err := svc.Import()
	if err == nil {
		t.Fatal("Import() should error on missing file")
	}
}

func TestImportMalformedJSONReturnsError(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&fakeRecommendationRepository{},
		newFakeUsers(),
		newFakeResources(),
		writeRecommendations(t, `{not-json`),
	)

	_, err := svc.Import()
	if err == nil {
		t.Fatal("Import() should error on malformed JSON")
	}
}

func TestImportMapsUserRepoError(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&fakeRecommendationRepository{},
		&errUserRepo{fakeUserRepository: newFakeUsers(1), err: errors.New("db error")},
		newFakeResources(),
		writeRecommendations(t, `{"1": [10]}`),
	)

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestImportMapsResourceRepoError(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&fakeRecommendationRepository{},
		newFakeUsers(1),
		&errResourceRepo{fakeResourceRepository: newFakeResources(), err: errors.New("db error")},
		writeRecommendations(t, `{"1": [10]}`),
	)

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestImportMapsReplaceError(t *testing.T) {
	svc := service.NewRecommendationImportService(
		&errRecRepo{fakeRecommendationRepository: &fakeRecommendationRepository{}, err: errors.New("db error")},
		newFakeUsers(1),
		newFakeResources(10),
		writeRecommendations(t, `{"1": [10]}`),
	)

	_, err := svc.Import()
	assertErrorCode(t, err, apperror.CodeInternal)
}
