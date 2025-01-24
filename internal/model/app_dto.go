package model

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateAppRequest 更新应用请求
type UpdateAppRequest struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	Status      AppStatus `json:"status"`
}

// AppResponse 应用响应
type AppResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      AppStatus `json:"status"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// AppCredentialsResponse 应用凭证响应
type AppCredentialsResponse struct {
	AppKey    string `json:"app_key"`
	AppSecret string `json:"app_secret"`
}
