package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

type fakeCommentRepository struct {
	has        bool
	hasErr     error
	list       []model.ResourceComment
	listErr    error
	lastListID uint
	created    []model.ResourceComment
	createErr  error
}

func (f *fakeCommentRepository) HasByResourceID(resourceID uint) (bool, error) {
	if f.hasErr != nil {
		return false, f.hasErr
	}
	return f.has, nil
}

func (f *fakeCommentRepository) ListByResourceID(resourceID uint, limit int) ([]model.ResourceComment, error) {
	f.lastListID = resourceID
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.list, nil
}

func (f *fakeCommentRepository) BatchCreate(comments []model.ResourceComment) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, comments...)
	return nil
}

type fakeCommentFetcher struct {
	comments []model.ResourceComment
	err      error
	lastBvid string
}

func (f *fakeCommentFetcher) FetchComments(bvid string, limit int) ([]model.ResourceComment, error) {
	f.lastBvid = bvid
	if f.err != nil {
		return nil, f.err
	}
	return f.comments, nil
}

func bilibiliResource() *model.Resource {
	return &model.Resource{
		Model:     gorm.Model{ID: 6},
		Type:      "video",
		SourceURL: "https://www.bilibili.com/video/BV1DgxCzREbM",
	}
}

func TestCommentListOrFetchReturnsCached(t *testing.T) {
	repo := &fakeCommentRepository{
		has:  true,
		list: []model.ResourceComment{{Model: gorm.Model{ID: 1}, AuthorName: "小明"}},
	}
	fetcher := &fakeCommentFetcher{}
	svc := service.NewCommentService(repo, fetcher, 20)

	comments, err := svc.ListOrFetch(context.Background(), bilibiliResource())

	if err != nil {
		t.Fatalf("ListOrFetch() error = %v", err)
	}
	if len(comments) != 1 || comments[0].AuthorName != "小明" {
		t.Fatalf("comments = %+v, want 缓存中的 1 条", comments)
	}
	if fetcher.lastBvid != "" {
		t.Fatal("缓存命中时不应调用在线抓取")
	}
}

func TestCommentListOrFetchSkipsNonBilibili(t *testing.T) {
	repo := &fakeCommentRepository{}
	fetcher := &fakeCommentFetcher{}
	svc := service.NewCommentService(repo, fetcher, 20)
	resource := &model.Resource{Model: gorm.Model{ID: 1}, Type: "course", SourceURL: "https://example.com/course/ml"}

	comments, err := svc.ListOrFetch(context.Background(), resource)

	if err != nil {
		t.Fatalf("ListOrFetch() error = %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want 空（非 B 站来源）", comments)
	}
	if fetcher.lastBvid != "" {
		t.Fatal("非 B 站来源不应触发在线抓取")
	}
}

func TestCommentListOrFetchFetchesAndPersists(t *testing.T) {
	repo := &fakeCommentRepository{}
	fetcher := &fakeCommentFetcher{
		comments: []model.ResourceComment{
			{AuthorName: "小明", Content: "讲得好", Floor: 1},
			{AuthorName: "阿强", Content: "赞", Floor: 2},
		},
	}
	svc := service.NewCommentService(repo, fetcher, 20)

	comments, err := svc.ListOrFetch(context.Background(), bilibiliResource())

	if err != nil {
		t.Fatalf("ListOrFetch() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("comments len = %d, want 2", len(comments))
	}
	if fetcher.lastBvid != "BV1DgxCzREbM" {
		t.Fatalf("FetchComments bvid = %q, want BV1DgxCzREbM", fetcher.lastBvid)
	}
	if len(repo.created) != 2 {
		t.Fatalf("created = %d 条, want 2", len(repo.created))
	}
	for _, c := range repo.created {
		if c.ResourceID != 6 {
			t.Fatalf("created ResourceID = %d, want 6", c.ResourceID)
		}
	}
}

func TestCommentListOrFetchSwallowsFetchError(t *testing.T) {
	repo := &fakeCommentRepository{}
	fetcher := &fakeCommentFetcher{err: errors.New("risk control")}
	svc := service.NewCommentService(repo, fetcher, 20)

	comments, err := svc.ListOrFetch(context.Background(), bilibiliResource())

	if err != nil {
		t.Fatalf("ListOrFetch() error = %v, want 抓取失败降级为空", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want 空", comments)
	}
	if len(repo.created) != 0 {
		t.Fatal("抓取失败不应写库")
	}
}

func TestCommentListOrFetchMapsRepoError(t *testing.T) {
	repo := &fakeCommentRepository{hasErr: errors.New("db error")}
	svc := service.NewCommentService(repo, &fakeCommentFetcher{}, 20)

	_, err := svc.ListOrFetch(context.Background(), bilibiliResource())
	assertErrorCode(t, err, apperror.CodeInternal)
}
