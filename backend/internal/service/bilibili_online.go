package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
)

// pythonTimeout 在线采集单次调用上限：bootstrap + 若干次带限流的请求
const pythonTimeout = 30 * time.Second

// BilibiliOnlineService 在线实时采集 B 站内容：搜索落库 + 评论抓取。
// 通过 os/exec 调用 backend/crawler/online.py，复用其 WBI 签名/搜索/风控逻辑。
type BilibiliOnlineService struct {
	importer     *BilibiliImportService
	pythonPath   string
	crawlerDir   string
	category     string
	searchLimit  int
	commentLimit int
}

func NewBilibiliOnlineService(importer *BilibiliImportService, cfg config.BilibiliConfig) *BilibiliOnlineService {
	pythonPath := strings.TrimSpace(cfg.PythonPath)
	if pythonPath == "" {
		pythonPath = "python"
	}
	crawlerDir := strings.TrimSpace(cfg.CrawlerDir)
	if crawlerDir == "" {
		crawlerDir = "crawler"
	}
	searchLimit := cfg.SearchLimit
	if searchLimit <= 0 {
		searchLimit = 10
	}
	commentLimit := cfg.CommentLimit
	if commentLimit <= 0 {
		commentLimit = 20
	}
	return &BilibiliOnlineService{
		importer:     importer,
		pythonPath:   pythonPath,
		crawlerDir:   crawlerDir,
		category:     strings.TrimSpace(cfg.Category),
		searchLimit:  searchLimit,
		commentLimit: commentLimit,
	}
}

// onlineSearchResponse 对应 online.py search 子命令的 stdout JSON
type onlineSearchResponse struct {
	Items []bilibiliItem `json:"items"`
}

// onlineCommentsResponse 对应 online.py comments 子命令的 stdout JSON
type onlineCommentsResponse struct {
	Comments []onlineComment `json:"comments"`
}

// onlineComment 对应 collect.normalize_comment 的输出
type onlineComment struct {
	AuthorName  string `json:"author_name"`
	Content     string `json:"content"`
	LikeCount   uint   `json:"like_count"`
	Floor       int    `json:"floor"`
	PublishedAt int64  `json:"published_at"`
}

// SearchAndImport 按关键词实时搜索 B 站视频并落库（复用离线导入逻辑）
func (s *BilibiliOnlineService) SearchAndImport(keyword string) (*BilibiliImportResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return &BilibiliImportResult{}, nil
	}

	args := []string{"online.py", "search", "--keyword", keyword, "--limit", strconv.Itoa(s.searchLimit)}
	if s.category != "" {
		args = append(args, "--category", s.category)
	}
	stdout, err := s.runPython(args)
	if err != nil {
		return nil, err
	}

	var resp onlineSearchResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("解析在线搜索输出失败: %w", err)
	}
	return s.importer.ImportItems(resp.Items, true)
}

// FetchComments 按 BV 号实时抓取视频主评论区评论（ResourceID 由调用方填充）
func (s *BilibiliOnlineService) FetchComments(bvid string, limit int) ([]model.ResourceComment, error) {
	bvid = strings.TrimSpace(bvid)
	if bvid == "" {
		return []model.ResourceComment{}, nil
	}
	if limit <= 0 {
		limit = s.commentLimit
	}

	stdout, err := s.runPython([]string{"online.py", "comments", "--bvid", bvid, "--limit", strconv.Itoa(limit)})
	if err != nil {
		return nil, err
	}

	var resp onlineCommentsResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("解析在线评论输出失败: %w", err)
	}

	comments := make([]model.ResourceComment, 0, len(resp.Comments))
	for _, c := range resp.Comments {
		comments = append(comments, model.ResourceComment{
			Bvid:        bvid,
			AuthorName:  c.AuthorName,
			Content:     c.Content,
			LikeCount:   c.LikeCount,
			Floor:       c.Floor,
			PublishedAt: c.PublishedAt,
		})
	}
	return comments, nil
}

// runPython 执行 online.py，stdout 只含 JSON（UTF-8），失败时把 stderr 并入错误
func (s *BilibiliOnlineService) runPython(args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pythonTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.pythonPath, args...)
	cmd.Dir = s.crawlerDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("调用 online.py 失败: %s", detail)
	}
	return stdout.Bytes(), nil
}
