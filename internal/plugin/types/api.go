package types

// APIInfo API接口信息
type APIInfo struct {
	Method      string            `json:"method"`      // HTTP方法
	Path        string            `json:"path"`        // 路径
	Description string            `json:"description"` // 接口描述
	Parameters  map[string]string `json:"parameters"`  // 参数说明
	Returns     map[string]string `json:"returns"`     // 返回值说明
}

// APIResponse API响应
type APIResponse struct {
	Code    int         `json:"code"`    // 响应码
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}

// NewAPIResponse 创建API响应
func NewAPIResponse(code int, message string, data interface{}) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *APIResponse {
	return NewAPIResponse(0, "success", data)
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *APIResponse {
	return NewAPIResponse(code, message, nil)
}
