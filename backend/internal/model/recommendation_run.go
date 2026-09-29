package model

// RecommendationRun 一次推荐推理运行的记录（可追溯）。
//
// engine 推理后写出旁挂信封 recommendations.meta.json；平台导入时读它并落一条记录，
// 于是「这批推荐是哪份快照、哪个模型、何时生成、覆盖多少用户」在库层面可查，
// 弥补「Recommendation 只有导入时间、说不清来源」的缺口。
type RecommendationRun struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	RunID             string `gorm:"size:32;index" json:"run_id"`    // engine 推理运行标识（信封 run_id）
	SnapshotRunID     string `gorm:"size:32" json:"snapshot_run_id"` // 所用数据快照的 run_id
	ModelName         string `gorm:"size:64" json:"model_name"`      // 模型名（信封 model.name）
	ModelVersion      string `gorm:"size:32" json:"model_version"`   // 模型版本
	Encoder           string `gorm:"size:160" json:"encoder"`        // 文本编码器版本
	GeneratedAt       int64  `json:"generated_at"`                   // 结果真实生成时间（信封 generated_at）
	TopN              int    `json:"top_n"`                          // 每用户输出上限
	UsersCount        int    `json:"users_count"`                    // 引擎覆盖的用户数
	ImportedUsers     int    `json:"imported_users"`                 // 实际导入用户数
	SkippedUsers      int    `json:"skipped_users"`                  // 跳过用户数
	ImportedResources int    `json:"imported_resources"`             // 实际导入资源条目数
	SkippedResources  int    `json:"skipped_resources"`              // 跳过资源条目数
	CreatedAt         int64  `json:"created_at"`                     // 平台导入时间（Unix 秒）
}
