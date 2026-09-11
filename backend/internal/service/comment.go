package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// commentFetcher 在线抓取 B 站评论的依赖；*BilibiliOnlineService 实现了它。
// 抽象成接口以便在单测里注入 fake，避免真实 os/exec 调用。
type commentFetcher interface {
	FetchComments(bvid string, limit int) ([]model.ResourceComment, error)
}

// CommentService B 站评论服务：有缓存读缓存，无缓存则实时爬取后落库。
// 只对 B 站来源（source_url 含 bilibili）的资源生效；站内评分评论仍走 RatingService。
type CommentService struct {
	comments repository.CommentRepository
	fetcher  commentFetcher
	limit    int
}

func NewCommentService(comments repository.CommentRepository, fetcher commentFetcher, limit int) *CommentService {
	if limit <= 0 {
		limit = 20
	}
	return &CommentService{comments: comments, fetcher: fetcher, limit: limit}
}

// ListOrFetch 返回资源下的 B 站评论列表（可能触发实时爬取）。
// 爬取失败不向上抛错，返回空列表，由前端展示「评论暂不可用」的空态。
func (s *CommentService) ListOrFetch(ctx context.Context, resource *model.Resource) ([]model.ResourceComment, error) {
	has, err := s.comments.HasByResourceID(resource.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if has {
		return s.comments.ListByResourceID(resource.ID, s.limit)
	}

	bvid := extractBvid(resource.SourceURL)
	if bvid == "" {
		// 非 B 站来源：没有可爬取的评论
		return []model.ResourceComment{}, nil
	}

	fetched, err := s.fetcher.FetchComments(bvid, s.limit)
	if err != nil {
		slog.Warn("B站评论抓取失败", "resource_id", resource.ID, "bvid", bvid, "error", err)
		return []model.ResourceComment{}, nil
	}
	if len(fetched) == 0 {
		return []model.ResourceComment{}, nil
	}

	for i := range fetched {
		fetched[i].ResourceID = resource.ID
	}
	if err := s.comments.BatchCreate(fetched); err != nil {
		return nil, apperror.Internal(err)
	}
	return fetched, nil
}

// extractBvid 从 source_url 提取 BV 号。source_url 形如
// https://www.bilibili.com/video/BV1DgxCzREbM，与 crawler/collect.py 的模板一致。
func extractBvid(sourceURL string) string {
	sourceURL = strings.TrimSpace(sourceURL)
	if !strings.HasPrefix(sourceURL, bilibiliSourceTemplate) {
		return ""
	}
	bvid := strings.TrimPrefix(sourceURL, bilibiliSourceTemplate)
	if i := strings.IndexAny(bvid, "?/"); i >= 0 {
		bvid = bvid[:i]
	}
	return bvid
}
