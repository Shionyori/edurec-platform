package model

// Recommendation 推荐结果缓存
type Recommendation struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"uniqueIndex;not null" json:"user_id"`    // 每个用户只存一份
	ResourceIDs string `gorm:"type:json;not null" json:"resource_ids"` // JSON 数组
	RunID       string `gorm:"size:32;index" json:"run_id"`            // 产出本次结果的运行标识（可追溯）
	Reasons     string `gorm:"type:json" json:"reasons"`               // JSON 数组，与 resource_ids 一一对应
	CreatedAt   int64  `json:"created_at"`                             // Unix 时间戳，首次生成时间
	UpdatedAt   int64  `json:"updated_at"`                             // Unix 时间戳，最近生成时间
}
