package service

import (
	"context"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// maxImpressionBatch 单次上报的曝光条数上限：曝光请求由前端在列表渲染后触发，
// 是高频、非关键路径，限制批量大小避免一次写入过大。
const maxImpressionBatch = 100

// ResourceImpressionService 推荐曝光打点。
//
// 曝光走独立存储，不写入 user_behaviors——后者会被 engine 当作正样本训练。
// 曝光用于统计 CTR（click / impression）与后续负采样，是推荐系统评估的前提。
type ResourceImpressionService struct {
	impressions repository.ResourceImpressionRepository
	resources   repository.ResourceRepository
}

func NewResourceImpressionService(
	impressions repository.ResourceImpressionRepository,
	resources repository.ResourceRepository,
) *ResourceImpressionService {
	return &ResourceImpressionService{impressions: impressions, resources: resources}
}

// Record 批量记录一次推荐展示。
//
// 语义约定：
//   - `resourceIDs` 的顺序即列表位次，`Position` 取去重后的下标；
//   - 去重、丢弃 0，并对超过上限的部分截断（曝光是统计信号，宁可截断也不报错）；
//   - 只记录平台真实存在的资源，避免脏 ID 落库（与行为上报口径一致）；
//   - 过滤后无可记录项时静默返回成功——埋点不应向用户报错。
func (s *ResourceImpressionService) Record(
	ctx context.Context, userID uint, scene string, resourceIDs []uint,
) error {
	scene = strings.TrimSpace(scene)
	if userID == 0 || scene == "" || len(scene) > 32 {
		return apperror.BadRequest("请求参数错误")
	}

	seen := make(map[uint]struct{}, len(resourceIDs))
	unique := make([]uint, 0, len(resourceIDs))
	for _, id := range resourceIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return apperror.BadRequest("请求参数错误")
	}
	if len(unique) > maxImpressionBatch {
		unique = unique[:maxImpressionBatch]
	}

	existing, err := s.resources.FindByIDs(unique)
	if err != nil {
		return apperror.Internal(err)
	}
	valid := make(map[uint]struct{}, len(existing))
	for i := range existing {
		valid[existing[i].ID] = struct{}{}
	}

	items := make([]model.ResourceImpression, 0, len(unique))
	for pos, id := range unique {
		if _, ok := valid[id]; !ok {
			continue
		}
		items = append(items, model.ResourceImpression{
			UserID:     userID,
			ResourceID: id,
			Scene:      scene,
			Position:   pos,
		})
	}
	if len(items) == 0 {
		return nil
	}
	if err := s.impressions.CreateBatch(items); err != nil {
		return apperror.Internal(err)
	}
	return nil
}
