package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// GeneratedConcept engine 生成的一个概念
type GeneratedConcept struct {
	Name     string  `json:"name"`
	PageRank float64 `json:"page_rank"`
}

// ConceptGenerationRunner 概念生成能力提供方（真实实现是 engine 的 HTTP 服务）
type ConceptGenerationRunner interface {
	Generate(ctx context.Context, point *model.KnowledgePoint) ([]GeneratedConcept, error)
}

// HTTPConceptRunnerConfig 在线 KG 客户端的生产参数。
//
// 这些参数不是可选的“调优项”，而是把在线依赖限制在可控范围内的边界：
// 超时、并发、熔断、鉴权缺一不可。
type HTTPConceptRunnerConfig struct {
	BaseURL          string
	Timeout          time.Duration
	Token            string        // 非空时发送 Authorization: Bearer
	MaxConcurrency   int           // 同时请求 engine 的上限
	FailureThreshold int           // 连续失败多少次后熔断（0 = 不熔断）
	CircuitCooldown  time.Duration // 熔断后的冷却时间
}

// HTTPConceptRunner 调用 engine 侧 kg_server.py 生成概念。
//
// 平台不直接执行 Python：算法与 venv 依赖都在 engine，平台只发 HTTP 请求。
// 在线依赖必须可降级：超时、并发上限、熔断与契约版本校验都在这里统一处理。
type HTTPConceptRunner struct {
	baseURL string
	client  *http.Client
	token   string
	sem     chan struct{}
	breaker *circuitBreaker
}

func NewHTTPConceptRunner(cfg HTTPConceptRunnerConfig) *HTTPConceptRunner {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxConcurrency := cfg.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	cooldown := cfg.CircuitCooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	transport := &http.Transport{
		MaxIdleConns:        16,
		MaxIdleConnsPerHost: maxConcurrency,
		IdleConnTimeout:     90 * time.Second,
	}
	return &HTTPConceptRunner{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		client:  &http.Client{Timeout: timeout, Transport: transport},
		token:   strings.TrimSpace(cfg.Token),
		sem:     make(chan struct{}, maxConcurrency),
		breaker: &circuitBreaker{
			threshold: cfg.FailureThreshold,
			cooldown:  cooldown,
		},
	}
}

type generateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PointCode   string `json:"point_code"`
}

type generateResponse struct {
	ContractVersion int                `json:"contract_version"`
	Concepts        []GeneratedConcept `json:"concepts"`
	Error           string             `json:"error"`
}

// requestIDKey 把调用链的 request id 放进 context，随 HTTP 头传给 engine。
type requestIDKey struct{}

// WithRequestID 注入请求 ID（handler 可从入站请求头取，命令行可自行生成）。
func WithRequestID(ctx context.Context, id string) context.Context {
	if strings.TrimSpace(id) == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, id)
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(requestIDKey{}).(string)
	return strings.TrimSpace(value)
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("kg-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

// circuitBreaker 简单的连续失败熔断：失败达到阈值后，冷却期内直接失败，
// 避免 engine 不可用时把平台的 goroutine/连接池拖垮。
type circuitBreaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	openUntil time.Time
	cooldown  time.Duration
}

func (b *circuitBreaker) before() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openUntil.IsZero() {
		return nil
	}
	if time.Now().Before(b.openUntil) {
		return fmt.Errorf("engine 知识生成服务熔断中，请稍后重试")
	}
	// 冷却结束，放行一次探测（half-open）。
	b.openUntil = time.Time{}
	return nil
}

func (b *circuitBreaker) success() {
	b.mu.Lock()
	b.failures = 0
	b.openUntil = time.Time{}
	b.mu.Unlock()
}

func (b *circuitBreaker) failure() {
	if b.threshold <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.failures = 0
		b.openUntil = time.Now().Add(b.cooldown)
	}
}

func (r *HTTPConceptRunner) Generate(ctx context.Context, point *model.KnowledgePoint) ([]GeneratedConcept, error) {
	if err := r.breaker.before(); err != nil {
		return nil, err
	}

	select {
	case r.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-r.sem }()

	payload, err := json.Marshal(generateRequest{
		Title:       point.Name,
		Description: point.Description,
		PointCode:   point.Code,
	})
	if err != nil {
		return nil, err
	}

	reqID := requestIDFromContext(ctx)
	if reqID == "" {
		reqID = newRequestID()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.baseURL+"/knowledge/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "edurec-platform/kg-client")
	req.Header.Set("X-Request-ID", reqID)
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		// 平台主动取消（关停/任务取消）不算 engine 故障；超时仍然算。
		if !errors.Is(err, context.Canceled) {
			r.breaker.failure()
		}
		return nil, fmt.Errorf("engine 知识生成服务不可用: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		r.breaker.failure()
		return nil, fmt.Errorf("读取 engine 响应失败: %w", err)
	}
	var parsed generateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		r.breaker.failure()
		return nil, fmt.Errorf("engine 响应不是合法 JSON: %w", err)
	}
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		r.breaker.failure()
	} else if resp.StatusCode >= 400 {
		r.breaker.success() // 4xx 是请求问题，不是引擎不可用
	}
	if resp.StatusCode >= 400 {
		message := parsed.Error
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("engine 生成失败（HTTP %d）: %s", resp.StatusCode, message)
	}
	if parsed.ContractVersion != 1 {
		r.breaker.failure()
		return nil, fmt.Errorf("engine 契约版本不支持: %d（期望 1）", parsed.ContractVersion)
	}
	r.breaker.success()
	return parsed.Concepts, nil
}

const (
	// generationJobTTL 已完成任务在内存里保留多久。状态只用于前端轮询，
	// 过期后回落到「idle + 当前概念总数」即可。
	generationJobTTL = time.Hour
	// maxGenerationJobs 内存任务表的上限，防止长期运行后无界增长。
	maxGenerationJobs = 1000
)

// GenerationStatus 一次概念生成任务的状态
type GenerationStatus struct {
	PointCode  string     `json:"point_code"`
	Status     string     `json:"status"` // idle / running / done / failed
	Generated  int        `json:"generated"`
	Total      int        `json:"total"`
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// ConceptGenerator 管理「按学科生成概念」的异步任务：
// 搜索时触发一次，前端轮询状态；同一学科并发触发会复用同一个任务。
type ConceptGenerator struct {
	knowledge repository.KnowledgeRepository
	runner    ConceptGenerationRunner
	cooldown  time.Duration

	mu   sync.Mutex
	jobs map[string]*GenerationStatus
}

func NewConceptGenerator(
	knowledge repository.KnowledgeRepository,
	runner ConceptGenerationRunner,
	cooldown time.Duration,
) *ConceptGenerator {
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	return &ConceptGenerator{
		knowledge: knowledge,
		runner:    runner,
		cooldown:  cooldown,
		jobs:      make(map[string]*GenerationStatus),
	}
}

// Start 异步触发一次生成；正在跑或刚跑过（冷却期内）时直接返回现有状态。
//
// ctx 只用于传递请求作用域的值（如 request id）；任务本身用 WithoutCancel
// 派生，避免 HTTP 请求结束后异步生成被取消。
func (g *ConceptGenerator) Start(ctx context.Context, code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	g.evictLocked(time.Now())
	if current, ok := g.jobs[point.Code]; ok {
		if current.Status == "running" || (current.Status == "done" && current.FinishedAt != nil &&
			time.Since(*current.FinishedAt) < g.cooldown) {
			snapshot := *current
			g.mu.Unlock()
			return &snapshot, nil
		}
	}
	started := time.Now()
	status := &GenerationStatus{PointCode: point.Code, Status: "running", StartedAt: &started}
	g.jobs[point.Code] = status
	snapshot := *status
	g.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	go func() {
		defer cancel()
		g.run(jobCtx, point)
	}()

	return &snapshot, nil
}

// Status 查询生成状态；没有任务时返回 idle（并带上当前概念总数）。
func (g *ConceptGenerator) Status(code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	g.evictLocked(time.Now())
	if current, ok := g.jobs[point.Code]; ok {
		snapshot := *current
		g.mu.Unlock()
		return &snapshot, nil
	}
	g.mu.Unlock()

	total, err := g.totalConcepts(point.Code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &GenerationStatus{PointCode: point.Code, Status: "idle", Total: total}, nil
}

// GenerateNow 同步生成（供批量重建命令使用），不受冷却时间限制。
func (g *ConceptGenerator) GenerateNow(ctx context.Context, code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	g.evictLocked(time.Now())
	if current, ok := g.jobs[point.Code]; ok && current.Status == "running" {
		g.mu.Unlock()
		return nil, apperror.BadRequest("该知识点的生成任务正在进行")
	}
	started := time.Now()
	status := &GenerationStatus{PointCode: point.Code, Status: "running", StartedAt: &started}
	g.jobs[point.Code] = status
	g.mu.Unlock()

	return g.generate(ctx, point)
}

func (g *ConceptGenerator) run(ctx context.Context, point *model.KnowledgePoint) {
	_, _ = g.generate(ctx, point)
}

// generate 调 engine、落库，并更新任务状态。
func (g *ConceptGenerator) generate(ctx context.Context, point *model.KnowledgePoint) (*GenerationStatus, error) {
	concepts, err := g.runner.Generate(ctx, point)

	g.mu.Lock()
	status := g.jobs[point.Code]
	if status == nil {
		// 极端情况下任务被并发清理：补一个状态，避免 nil panic。
		status = &GenerationStatus{PointCode: point.Code, Status: "running"}
		g.jobs[point.Code] = status
	}
	finished := time.Now()
	status.FinishedAt = &finished
	g.mu.Unlock()

	if err != nil {
		g.mu.Lock()
		status.Status = "failed"
		status.Error = err.Error()
		snapshot := *status
		g.mu.Unlock()
		return &snapshot, err
	}

	generated := 0
	for _, concept := range concepts {
		name := strings.TrimSpace(concept.Name)
		if name == "" {
			continue
		}
		if err := g.upsertConcept(point, name, concept.PageRank); err != nil {
			g.mu.Lock()
			status.Status = "failed"
			status.Error = err.Error()
			snapshot := *status
			g.mu.Unlock()
			return &snapshot, err
		}
		generated++
	}

	total, err := g.totalConcepts(point.Code)
	if err != nil {
		return nil, err
	}

	g.mu.Lock()
	status.Status = "done"
	status.Generated = generated
	status.Total = total
	snapshot := *status
	g.mu.Unlock()
	return &snapshot, nil
}

// upsertConcept 按「学科 + 概念名」去重：已有概念只更新 pageRank（保留原来源），新概念标记 generated。
func (g *ConceptGenerator) upsertConcept(point *model.KnowledgePoint, name string, pageRank float64) error {
	existing, err := g.knowledge.FindConceptByNamePoint(name, point.Code)
	if err != nil {
		return err
	}
	if existing != nil {
		return g.knowledge.UpsertConcept(&model.KnowledgeConcept{
			Code:        existing.Code,
			Name:        existing.Name,
			Field:       existing.Field,
			PointCode:   existing.PointCode,
			Explanation: existing.Explanation,
			Source:      existing.Source,
			PageRank:    float32(pageRank),
		})
	}

	return g.knowledge.UpsertConcept(&model.KnowledgeConcept{
		Code:        GeneratedConceptCode(point.Code, name),
		Name:        name,
		Field:       point.Name,
		PointCode:   point.Code,
		Level:       1,
		Explanation: "由 engine 的 generate.py 根据学科标题/描述生成",
		Source:      "generated",
		PageRank:    float32(pageRank),
	})
}

// evictLocked 清理过期的已完成任务，并在总量超限时淘汰已完成任务。
// 调用方必须持有 g.mu。
func (g *ConceptGenerator) evictLocked(now time.Time) {
	for code, status := range g.jobs {
		if status.Status != "running" && status.FinishedAt != nil &&
			now.Sub(*status.FinishedAt) > generationJobTTL {
			delete(g.jobs, code)
		}
	}
	if len(g.jobs) <= maxGenerationJobs {
		return
	}
	for code, status := range g.jobs {
		if status.Status == "running" {
			continue
		}
		delete(g.jobs, code)
		if len(g.jobs) <= maxGenerationJobs {
			return
		}
	}
}

func (g *ConceptGenerator) totalConcepts(pointCode string) (int, error) {
	counts, err := g.knowledge.CountConceptsByPoint()
	if err != nil {
		return 0, err
	}
	return int(counts[pointCode]), nil
}

// GeneratedConceptCode 生成概念的稳定 code：同一学科同一概念名重复生成不会产生新行。
func GeneratedConceptCode(pointCode, name string) string {
	sum := sha1.Sum([]byte(pointCode + "|" + name))
	return fmt.Sprintf("GEN_%s_%s", hex.EncodeToString(sum[:])[:12], pointCode)
}
