package model

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname string     `json:"nickname"`
	Email    string     `json:"email"`
	Phone    string     `json:"phone"`
	Status   UserStatus `json:"status"`
}

// UpdatePasswordRequest 更新密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID        string     `json:"id"`
	AppID     string     `json:"app_id"`
	Username  string     `json:"username"`
	Nickname  string     `json:"nickname"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	Status    UserStatus `json:"status"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
