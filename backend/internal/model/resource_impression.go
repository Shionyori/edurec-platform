package model

import "gorm.io/gorm"

// ResourceImpression 推荐曝光日志（资源「被展示」的记录）。
//
// 刻意与 UserBehavior 分开建模：曝光是系统展示行为，不是用户的正向行为。
// engine 侧把 behaviors 全部当作正样本参与训练（见 engine 的 build_interactions），
// 若把曝光混进 user_behaviors，会把「只是被展示」误当成「用户感兴趣」，污染训练数据。
// 曝光数据用于统计点击率（CTR = click / impression）与后续的负采样。
type ResourceImpression struct {
	gorm.Model
	UserID     uint   `gorm:"index;not null" json:"user_id"`
	User       User   `gorm:"foreignKey:UserID" json:"-"`
	ResourceID uint   `gorm:"index;not null" json:"resource_id"`
	Resource   Resource `gorm:"foreignKey:ResourceID" json:"-"`
	Scene      string `gorm:"size:32;not null;index" json:"scene"` // 场景：home / search / detail ...
	Position   int    `gorm:"not null;default:0" json:"position"`  // 在推荐列表中的位次（0 基）
}
