package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// RecommendationStats 推荐效果统计（曝光 / 点击 / 收藏 / 覆盖用户）。
// 曝光来自独立的 resource_impressions 表，与用户行为分开统计——
// CTR = 点击 / 曝光，是推荐系统评估的基础指标。
type RecommendationStats struct {
	Impressions         int64
	Clicks              int64
	Favorites           int64
	Views               int64
	RecommendationUsers int64
}

type StatsRepository interface {
	RecommendationStats() (*RecommendationStats, error)
}

type StatsRepo struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

func (r *StatsRepo) RecommendationStats() (*RecommendationStats, error) {
	stats := &RecommendationStats{}
	if err := r.db.Model(&model.ResourceImpression{}).Count(&stats.Impressions).Error; err != nil {
		return nil, fmt.Errorf("统计曝光失败: %w", err)
	}

	countAction := func(action string) (int64, error) {
		var n int64
		err := r.db.Model(&model.UserBehavior{}).Where("action = ?", action).Count(&n).Error
		return n, err
	}
	var err error
	if stats.Clicks, err = countAction("click"); err != nil {
		return nil, fmt.Errorf("统计点击失败: %w", err)
	}
	if stats.Favorites, err = countAction("favorite"); err != nil {
		return nil, fmt.Errorf("统计收藏失败: %w", err)
	}
	if stats.Views, err = countAction("view"); err != nil {
		return nil, fmt.Errorf("统计浏览失败: %w", err)
	}
	if err := r.db.Model(&model.Recommendation{}).Count(&stats.RecommendationUsers).Error; err != nil {
		return nil, fmt.Errorf("统计推荐覆盖用户失败: %w", err)
	}
	return stats, nil
}
