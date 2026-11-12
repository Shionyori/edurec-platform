package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecommendationUpsert 覆盖写某用户推荐缓存的入参。
// 用结构体而非散参，便于后续增加字段（如 scores/曝光）而不反复改签名。
type RecommendationUpsert struct {
	UserID      uint
	ResourceIDs string // JSON 数组字符串
	RunID       string // 产出本次结果的运行标识，空表示平台兜底生成
	Reasons     string // JSON 数组字符串，与 ResourceIDs 一一对应；可为空
	Now         int64  // Unix 秒
}

// RecommendationRepository 推荐结果缓存数据访问接口
type RecommendationRepository interface {
	// FindByUserID 按用户查询推荐缓存，未命中时返回 (nil, nil)
	FindByUserID(userID uint) (*model.Recommendation, error)
	// Replace 整行替换某用户的推荐缓存（每个用户只保留一份）
	Replace(input RecommendationUpsert) (*model.Recommendation, error)
	// ReplaceMany 在一个事务里批量覆盖多个用户的缓存。
	// 导入是「一批结果一起生效」的操作，逐用户删除+插入既慢又可能留下半批状态。
	ReplaceMany(inputs []RecommendationUpsert) error
}

type RecommendationRepo struct {
	db *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) *RecommendationRepo {
	return &RecommendationRepo{db: db}
}

func (r *RecommendationRepo) FindByUserID(userID uint) (*model.Recommendation, error) {
	rec := &model.Recommendation{}
	err := r.db.Where("user_id = ?", userID).First(rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询推荐缓存失败: %w", err)
	}
	return rec, nil
}

func (r *RecommendationRepo) Replace(input RecommendationUpsert) (*model.Recommendation, error) {
	if err := r.ReplaceMany([]RecommendationUpsert{input}); err != nil {
		return nil, err
	}
	return toRecommendation(input), nil
}

// ReplaceMany 批量覆盖写：利用 user_id 唯一索引做一次 UPSERT。
//
// 为什么不是「逐用户先删后插」：导入一份全量结果可能有数万用户，逐条往返
// 既慢，又会在中途失败时留下「一部分用户已更新、另一部分还是旧结果」的半批状态。
// CreateInBatches + ON DUPLICATE KEY UPDATE 把**本次传入的这一批**放进一个事务；
// 调用方（导入服务）再按 ≤500 用户切分，避免单事务长时间持锁。
// 语义：每一批内部要么全部生效，要么全部不生效。
func (r *RecommendationRepo) ReplaceMany(inputs []RecommendationUpsert) error {
	if len(inputs) == 0 {
		return nil
	}
	rows := make([]model.Recommendation, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, *toRecommendation(input))
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"resource_ids", "run_id", "reasons", "created_at", "updated_at",
			}),
		}).CreateInBatches(&rows, 200).Error
	})
	if err != nil {
		return fmt.Errorf("批量保存推荐缓存失败: %w", err)
	}
	return nil
}

// toRecommendation 把入参归一成模型；Reasons 为空时写 `[]`，
// 保证读侧可以无条件 JSON 解析。
func toRecommendation(input RecommendationUpsert) *model.Recommendation {
	if input.Reasons == "" {
		input.Reasons = "[]"
	}
	return &model.Recommendation{
		UserID:      input.UserID,
		ResourceIDs: input.ResourceIDs,
		RunID:       input.RunID,
		Reasons:     input.Reasons,
		CreatedAt:   input.Now,
		UpdatedAt:   input.Now,
	}
}
