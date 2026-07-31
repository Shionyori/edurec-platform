package model

import "gorm.io/gorm"

// Rating 用户评分与评论
type Rating struct {
	gorm.Model
	UserID     uint   `gorm:"index;not null" json:"user_id"`
	User       User   `gorm:"foreignKey:UserID" json:"user"`
	ResourceID uint   `gorm:"index;not null" json:"resource_id"`
	Resource   Resource `gorm:"foreignKey:ResourceID" json:"-"`
	Score      uint8  `gorm:"not null" json:"score"`     // 1-5
	Comment    string `gorm:"type:text" json:"comment"`
}
