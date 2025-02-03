package model

// AuthStatus 认证状态
const (
	AuthStatusPending   = "pending"
	AuthStatusCompleted = "completed"
	AuthStatusFailed    = "failed"
)

// ExtendedLoginResponse 扩展的登录响应(包含插件信息)
type ExtendedLoginResponse struct {
	User         UserResponse        `json:"user"`
	AccessToken  string              `json:"access_token,omitempty"`  // 访问令牌(验证完成时返回)
	RefreshToken string              `json:"refresh_token,omitempty"` // 刷新令牌(验证完成时返回)
	ExpiresIn    int64               `json:"expires_in,omitempty"`    // 过期时间(验证完成时返回)
	AuthStatus   string              `json:"auth_status"`             // 认证状态
	Plugins      []PluginRequirement `json:"plugins,omitempty"`       // 需要的插件列表
	NextPlugin   *PluginRequirement  `json:"next_plugin,omitempty"`   // 下一个需要验证的插件
	SessionID    string              `json:"session_id,omitempty"`    // 验证会话ID(验证未完成时返回)
}

// PendingLoginResponse 未完成验证的登录响应
type PendingLoginResponse ExtendedLoginResponse
