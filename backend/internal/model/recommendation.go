package model

// Recommendation 推荐结果缓存
type Recommendation struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"uniqueIndex;not null" json:"user_id"` // 每个用户只存一份
	ResourceIDs string `gorm:"type:json;not null" json:"resource_ids"` // JSON 数组
	CreatedAt   int64  `json:"created_at"` // Unix 时间戳
}
