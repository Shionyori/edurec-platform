package service_test

import (
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

type fakeKnowledgePointRepository struct {
	points []model.KnowledgePoint
	err    error
}

func (f *fakeKnowledgePointRepository) ListAll() ([]model.KnowledgePoint, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.points, nil
}

func (f *fakeKnowledgePointRepository) UpsertByName(points []model.KnowledgePoint) (int, int, error) {
	return len(points), 0, nil
}

// fakePlanResourceRepository 按「查询里带的标签」返回资源，并记录收到的查询。
//
// 知识点分桶检索走的是 AnyTags（OR 语义，见 repository.ResourceListQuery 注释），
// 所以这里读 q.AnyTags；若实现误传 AND 语义的 q.Tags，视为「没带标签」而落回
// byTag[""]，测试随即失败——替身依然能分辨这两种写法。
type fakePlanResourceRepository struct {
	byTag   map[string][]model.Resource
	totals  map[string]int64
	queries []repository.ResourceListQuery
}

func (f *fakePlanResourceRepository) List(q repository.ResourceListQuery) (*repository.ResourceListResult, error) {
	f.queries = append(f.queries, q)
	if len(q.AnyTags) == 0 {
		items := f.byTag[""]
		return &repository.ResourceListResult{Items: items, Total: int64(len(items))}, nil
	}
	tag := q.AnyTags[0]
	items := f.byTag[tag]
	total := f.totals[tag]
	if total == 0 {
		total = int64(len(items))
	}
	return &repository.ResourceListResult{Items: items, Total: total}, nil
}

func (f *fakePlanResourceRepository) Create(*model.Resource) error { return nil }
func (f *fakePlanResourceRepository) FindByID(uint) (*model.Resource, error) {
	return nil, repository.ErrNotFound
}

// FindByIDs 必须真的按 id 返回资源：RecommendationService.Get 靠它把缓存的
// 推荐 ID 列表还原成资源顺序，返回 nil 会让「engine 推荐序优先」这条测试假通过。
func (f *fakePlanResourceRepository) FindByIDs(ids []uint) ([]model.Resource, error) {
	byID := make(map[uint]model.Resource)
	for _, items := range f.byTag {
		for _, r := range items {
			byID[r.ID] = r
		}
	}
	out := make([]model.Resource, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakePlanResourceRepository) ListTopByCategories([]uint, int) ([]model.Resource, error) {
	return nil, nil
}
func (f *fakePlanResourceRepository) FindBySourceURLs([]string) ([]model.Resource, error) {
	return nil, nil
}
func (f *fakePlanResourceRepository) Update(*model.Resource) error { return nil }
func (f *fakePlanResourceRepository) Delete(uint) error            { return nil }

func point(id uint, name, aliases string, categoryID uint) model.KnowledgePoint {
	p := model.KnowledgePoint{Name: name, Aliases: aliases, CategoryID: categoryID}
	p.ID = id
	if categoryID != 0 {
		p.Category = model.Category{Name: "人工智能"}
		p.Category.ID = categoryID
	}
	return p
}

func resource(id uint, rating float32, views uint) model.Resource {
	r := model.Resource{AvgRating: rating, ViewCount: views}
	r.ID = id
	return r
}

func newPlanService(
	points []model.KnowledgePoint,
	resources *fakePlanResourceRepository,
	recRepo repository.RecommendationRepository,
) *service.KnowledgePointService {
	return service.NewKnowledgePointService(
		&fakeKnowledgePointRepository{points: points},
		resources,
		service.NewRecommendationService(recRepo, resources, &fakeUserRepository{}),
	)
}

func TestPlanForGoal_MatchByName(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
		"线性代数": {resource(2, 4.0, 50)},
	}}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
		point(2, "线性代数", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if !plan.Matched {
		t.Fatal("Matched = false, want true")
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "机器学习" {
		t.Fatalf("groups = %+v, want 仅 机器学习", plan.Groups)
	}
}

// 首轮知识点由中文标签聚合，aliases 全空，因此英文目标必然走兜底。
// 这条测试把「兜底而不是报错/空页」钉死。
func TestPlanForGoal_FallsBackWhenNoAliasMatches(t *testing.T) {
	resources := &fakePlanResourceRepository{
		byTag:  map[string][]model.Resource{"": {resource(9, 4.0, 10)}},
		totals: map[string]int64{},
	}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "machine learning", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if plan.Matched {
		t.Error("Matched = true, want false（英文目标应走兜底）")
	}
	if len(plan.Groups) != 1 {
		t.Fatalf("兜底应返回单一分组，got %d", len(plan.Groups))
	}
	if len(plan.Groups[0].Resources) == 0 {
		t.Error("兜底分组不该是空的——绝不能返回空白页")
	}
}

func TestPlanForGoal_MatchByAliasIgnoreCase(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `["Machine Learning"]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想学 machine learning", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups) != 1 {
		t.Fatalf("拉丁文别名应忽略大小写命中，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_MatchByCategoryWhenNameMisses(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	// 知识点名与目标文本没有公共子串，但分类名「人工智能」在目标里
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门人工智能", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups) != 1 {
		t.Fatalf("应按分类名命中，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_ShortNameIsNotMatched(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{"": {resource(9, 4, 1)}}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "C", `[]`, 0)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想学 C 语言和 Python", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	// 单字知识点名会命中一切目标，必须挡掉；命中不了就兜底
	if plan.Matched {
		t.Error("单字符知识点名不该被匹配")
	}
}

func TestPlanForGoal_SkipsKnowledgePointWithoutResources(t *testing.T) {
	resources := &fakePlanResourceRepository{
		byTag:  map[string][]model.Resource{"机器学习": {}, "线性代数": {resource(2, 4.0, 50)}},
		totals: map[string]int64{"机器学习": 0, "线性代数": 1},
	}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
		point(2, "线性代数", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门机器学习和线性代数", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "线性代数" {
		t.Fatalf("无资源的知识点应被剔除，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_RanksByEngineRecommendationOrder(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {
			resource(1, 5.0, 1000), // 评分最高
			resource(2, 3.0, 10),   // 评分最低，但在 engine 推荐里排第一
		},
	}}
	// 字段名是 found（见 recommendation_test.go 里 fakeRecommendationRepository 的定义）
	recRepo := &fakeRecommendationRepository{found: &model.Recommendation{
		UserID: 7, ResourceIDs: `[2,1]`, UpdatedAt: 1,
	}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources, recRepo)

	plan, err := svc.PlanForGoal(7, "机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	got := plan.Groups[0].Resources
	if len(got) != 2 || got[0].ID != 2 {
		t.Fatalf("engine 推荐序优先，want 资源 2 在前，got %+v", got)
	}
}

func TestPlanForGoal_CapsResourcesPerPoint(t *testing.T) {
	items := make([]model.Resource, 0, 10)
	for i := 1; i <= 10; i++ {
		items = append(items, resource(uint(i), float32(i), uint(i)))
	}
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{"机器学习": items}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups[0].Resources) != service.DefaultResourcePerPoint {
		t.Errorf("组内资源 = %d, want %d", len(plan.Groups[0].Resources), service.DefaultResourcePerPoint)
	}
	if plan.Groups[0].Total != 10 {
		t.Errorf("Total = %d, want 10（总数不受展示上限影响）", plan.Groups[0].Total)
	}
}

func TestPlanForGoal_RejectsBadGoal(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{}}
	svc := newPlanService(nil, resources, &fakeRecommendationRepository{})

	t.Run("空目标", func(t *testing.T) {
		if _, err := svc.PlanForGoal(7, "   ", 8); err == nil {
			t.Error("空目标应当报错")
		}
	})

	t.Run("超长目标", func(t *testing.T) {
		long := ""
		for i := 0; i < 101; i++ {
			long += "学"
		}
		if _, err := svc.PlanForGoal(7, long, 8); err == nil {
			t.Error("超长目标应当报错")
		}
	})
}

func TestPlanForGoal_RepositoryErrorIsInternal(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{}}
	svc := service.NewKnowledgePointService(
		&fakeKnowledgePointRepository{err: errors.New("db down")},
		resources,
		service.NewRecommendationService(&fakeRecommendationRepository{}, resources, &fakeUserRepository{}),
	)
	if _, err := svc.PlanForGoal(7, "机器学习", 8); err == nil {
		t.Error("仓储报错应当向上传递")
	}
}

// 已接受的已知行为：子串匹配不是语义匹配。把行为钉死，避免将来被当成 bug 反复「修」。
func TestPlanForGoal_SubstringMatchIgnoresNegation(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我不想学机器学习了", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if !plan.Matched {
		t.Error("已知行为：子串匹配不识别否定，仍应命中「机器学习」")
	}
}
