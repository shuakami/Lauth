package model

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username   string                `json:"username" binding:"required"`
	Password   string                `json:"password" binding:"required"`
	Nickname   string                `json:"nickname"`
	Email      string                `json:"email"`
	Phone      string                `json:"phone"`
	Profile    *CreateProfileRequest `json:"profile"`
	ClientIP   string                `json:"client_ip,omitempty"`   // 客户端IP
	DeviceID   string                `json:"device_id,omitempty"`   // 设备ID
	DeviceType string                `json:"device_type,omitempty"` // 设备类型
	UserAgent  string                `json:"user_agent,omitempty"`  // User-Agent
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname string                `json:"nickname"`
	Email    string                `json:"email"`
	Phone    string                `json:"phone"`
	Status   UserStatus            `json:"status"`
	Profile  *UpdateProfileRequest `json:"profile"`
}

// UpdatePasswordRequest 更新密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// FirstChangePasswordRequest 首次修改密码请求（不需要旧密码）
type FirstChangePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID                 string     `json:"id"`
	AppID              string     `json:"app_id"`
	Username           string     `json:"username"`
	Nickname           string     `json:"nickname"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Status             UserStatus `json:"status"`
	Profile            *Profile   `json:"profile,omitempty"`
	IsFirstLogin       bool       `json:"is_first_login"`
	LastLoginAt        *string    `json:"last_login_at,omitempty"`
	PasswordExpiresAt  *string    `json:"password_expires_at,omitempty"` // 密码过期时间
	NeedChangePassword bool       `json:"need_change_password"`
	CreatedAt          string     `json:"created_at"`
	UpdatedAt          string     `json:"updated_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码

	// 验证上下文信息
	ClientIP   string `json:"client_ip,omitempty"`   // 客户端IP
	DeviceID   string `json:"device_id,omitempty"`   // 设备ID
	DeviceType string `json:"device_type,omitempty"` // 设备类型
	UserAgent  string `json:"user_agent,omitempty"`  // User-Agent
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // 过期时间（秒）
}

// LoginCookieResponse Cookie认证的登录响应
type LoginCookieResponse struct {
	User      UserResponse `json:"user"`
	ExpiresIn int64        `json:"expires_in"` // 过期时间（秒）
}
