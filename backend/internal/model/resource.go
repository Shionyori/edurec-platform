package model

import "gorm.io/gorm"

// 资源类型的合法取值，与 resources.type 的枚举语义一致
const (
	ResourceTypeCourse  = "course"
	ResourceTypeArticle = "article"
	ResourceTypeVideo   = "video"
)

// IsValidResourceType 判断是否为合法资源类型。
// 外部来源（爬虫、数据集配置）落库前用它校验，避免写入枚举外的值。
func IsValidResourceType(resourceType string) bool {
	switch resourceType {
	case ResourceTypeCourse, ResourceTypeArticle, ResourceTypeVideo:
		return true
	}
	return false
}

// 合法学习难度取值
const (
	DifficultyBeginner     = "beginner"
	DifficultyIntermediate = "intermediate"
	DifficultyAdvanced     = "advanced"
)

// IsValidDifficulty 判断是否为合法难度；空串表示未知，也视为合法。
func IsValidDifficulty(difficulty string) bool {
	switch difficulty {
	case "", DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced:
		return true
	}
	return false
}

// Resource 教育资源（统一抽象，type 区分类型）
type Resource struct {
	gorm.Model
	Title       string   `gorm:"size:256;not null;index" json:"title"`
	Description string   `gorm:"type:text" json:"description"`
	CoverURL    string   `gorm:"size:512" json:"cover_url"`
	Type        string   `gorm:"size:20;not null;index" json:"type"` // course / article / video
	CategoryID  uint     `gorm:"index" json:"category_id"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
	Tags        string   `gorm:"type:json" json:"tags"`     // JSON 数组字符串
	Metadata    string   `gorm:"type:json" json:"metadata"` // JSON 扩展字段
	Author      string   `gorm:"size:128" json:"author"`
	SourceURL   string   `gorm:"size:512;index" json:"source_url"`
	AvgRating   float32  `gorm:"type:decimal(2,1);default:0" json:"avg_rating"`
	ViewCount   uint     `gorm:"default:0" json:"view_count"`
	// Difficulty 学习难度：beginner / intermediate / advanced，空串表示未知。
	// 作为结构化特征参与推荐排序（难度匹配是教育推荐区别于通用推荐的核心之一）。
	Difficulty string `gorm:"size:16;index" json:"difficulty"`
	// DurationMinutes 资源时长（分钟），0 表示未知。
	DurationMinutes uint `gorm:"default:0" json:"duration_minutes"`
}
