package model

// Admin 管理员（独立建表，关联 User）
type Admin struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"uniqueIndex;not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`
}
