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
	// Interests 冷启动兴趣：分类 ID 的 JSON 数组字符串（如 "[1,3]"）。
	// 新用户无行为历史时，平台兜底推荐优先取这些分类下的资源。
	Interests string `gorm:"type:json" json:"interests"`
}

// BeforeCreate keeps MySQL's JSON column valid for new users. GORM would
// otherwise insert the zero value "" into a JSON column, which MySQL rejects
// with "Invalid JSON text: The document is empty".
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.Interests == "" {
		u.Interests = "[]"
	}
	return nil
}
