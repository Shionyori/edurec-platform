package model

import "gorm.io/gorm"

// Resource 教育资源（统一抽象，type 区分类型）
type Resource struct {
	gorm.Model
	Title       string `gorm:"size:256;not null;index" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	CoverURL    string `gorm:"size:512" json:"cover_url"`
	Type        string `gorm:"size:20;not null;index" json:"type"` // course / article / video
	CategoryID  uint   `gorm:"index" json:"category_id"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
	Tags        string `gorm:"type:json" json:"tags"`     // JSON 数组字符串
	Metadata    string `gorm:"type:json" json:"metadata"` // JSON 扩展字段
	Author      string `gorm:"size:128" json:"author"`
	SourceURL   string `gorm:"size:512" json:"source_url"`
	AvgRating   float32 `gorm:"type:decimal(2,1);default:0" json:"avg_rating"`
	ViewCount   uint `gorm:"default:0" json:"view_count"`
}
