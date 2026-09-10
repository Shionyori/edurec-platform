package model

import "gorm.io/gorm"

// ResourceComment 资源下的 B 站评论（爬取自 B 站公开评论区，独立于站内 Rating 评分）
type ResourceComment struct {
	gorm.Model
	ResourceID  uint   `gorm:"index;not null" json:"resource_id"`
	Resource    Resource `gorm:"foreignKey:ResourceID" json:"-"`
	Bvid        string `gorm:"size:64;index" json:"bvid"` // 来源视频 BV 号，便于追溯
	AuthorName  string `gorm:"size:128" json:"author_name"`
	Content     string `gorm:"type:text" json:"content"`
	LikeCount   uint   `gorm:"default:0" json:"like_count"`
	Floor       int    `gorm:"default:0" json:"floor"`
	PublishedAt int64  `gorm:"default:0" json:"published_at"` // B 站 ctime，Unix 秒
}
