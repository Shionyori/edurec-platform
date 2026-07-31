package model

import "gorm.io/gorm"

// UserBehavior 用户行为日志
type UserBehavior struct {
	gorm.Model
	UserID     uint   `gorm:"index;not null" json:"user_id"`
	User       User   `gorm:"foreignKey:UserID" json:"-"`
	ResourceID uint   `gorm:"index;not null" json:"resource_id"`
	Resource   Resource `gorm:"foreignKey:ResourceID" json:"-"`
	Action     string `gorm:"size:20;not null" json:"action"` // view / click / favorite
}
