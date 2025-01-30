package model

import "time"

// PluginRequirement 插件需求
type PluginRequirement struct {
	Name     string `json:"name"`     // 插件名称
	Required bool   `json:"required"` // 是否必须
	Stage    string `json:"stage"`    // 执行阶段 (pre_login, post_login, pre_register, post_register)
	Status   string `json:"status"`   // 状态 (pending, completed, failed)
}

// PluginStatus 插件状态记录
type PluginStatus struct {
	ID        string    `json:"id"`         // 状态ID
	AppID     string    `json:"app_id"`     // 应用ID
	UserID    string    `json:"user_id"`    // 用户ID
	Action    string    `json:"action"`     // 动作
	Plugin    string    `json:"plugin"`     // 插件名称
	Status    string    `json:"status"`     // 状态
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// PluginStatus 插件状态
const (
	PluginStatusPending   = "pending"
	PluginStatusCompleted = "completed"
	PluginStatusFailed    = "failed"
)

// PluginStage 插件执行阶段
const (
	PluginStagePreLogin     = "pre_login"
	PluginStagePostLogin    = "post_login"
	PluginStagePreRegister  = "pre_register"
	PluginStagePostRegister = "post_register"
)

// PluginConfig 插件配置
type PluginConfig struct {
	ID        string                 `json:"id" gorm:"primaryKey"`                     // 配置ID
	AppID     string                 `json:"app_id" gorm:"index"`                      // 应用ID
	Name      string                 `json:"name"`                                     // 插件名称
	Config    map[string]interface{} `json:"config" gorm:"type:json;serializer:json"`  // 插件配置
	Required  bool                   `json:"required"`                                 // 是否必需
	Stage     string                 `json:"stage"`                                    // 执行阶段
	Actions   []string               `json:"actions" gorm:"type:json;serializer:json"` // 适用的动作列表
	Enabled   bool                   `json:"enabled"`                                  // 是否启用
	CreatedAt time.Time              `json:"created_at"`                               // 创建时间
	UpdatedAt time.Time              `json:"updated_at"`                               // 更新时间
}

// TableName 返回表名
func (PluginConfig) TableName() string {
	return "plugin_configs"
}

// VerificationSession 验证会话
type VerificationSession struct {
	ID        string    `json:"id" gorm:"primaryKey"` // 会话ID
	AppID     string    `json:"app_id" gorm:"index"`  // 应用ID
	UserID    string    `json:"user_id" gorm:"index"` // 用户ID
	Action    string    `json:"action"`               // 动作(login/register等)
	Status    string    `json:"status"`               // 状态
	CreatedAt time.Time `json:"created_at"`           // 创建时间
	UpdatedAt time.Time `json:"updated_at"`           // 更新时间
	ExpiredAt time.Time `json:"expired_at"`           // 过期时间
}

// TableName 返回表名
func (VerificationSession) TableName() string {
	return "verification_sessions"
}
