package model

import "gorm.io/gorm"

// 知识点来源，用于将来区分产出方（换成 LLM 打标时下游不用改）
const (
	KnowledgePointSourceTag  = "tag"  // 由 resources.tags 聚合而来
	KnowledgePointSourceSeed = "seed" // 人工预置
	KnowledgePointSourceLLM  = "llm"  // 离线 LLM 打标
)

// KnowledgePoint 学习知识点：学习目标被拆解后的一块主题，挂载若干真实资源。
//
// 本轮知识点由资源标签聚合而来（Source = "tag"），Name 与 resources.tags 里的
// 字符串按原值对应，服务层据此用 JSON_CONTAINS 过滤资源。Aliases 不参与资源
// 过滤，只参与「目标文本 → 知识点」的匹配。
type KnowledgePoint struct {
	gorm.Model
	Name        string   `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string   `gorm:"size:512" json:"description"`
	Aliases     string   `gorm:"type:json" json:"aliases"` // JSON 数组字符串，仅用于目标匹配
	CategoryID  uint     `gorm:"index" json:"category_id"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
	Source      string   `gorm:"size:32;index" json:"source"`
	SortOrder   int      `gorm:"default:0" json:"sort_order"`
}
