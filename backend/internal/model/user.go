package model

import "gorm.io/gorm"

// User 用户信息
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email        string `gorm:"uniqueIndex;size:128;not null" json:"email"`
	PasswordHash string `gorm:"size:256;not null" json:"-"`
	DisplayName  string `gorm:"size:128" json:"display_name"`
	AvatarURL    string `gorm:"size:512" json:"avatar_url"`
}
