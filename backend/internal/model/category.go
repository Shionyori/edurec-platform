package model

import "gorm.io/gorm"

// Category 资源分类
type Category struct {
	gorm.Model
	Name        string `gorm:"size:64;not null" json:"name"`
	Description string `gorm:"size:256" json:"description"`
}
