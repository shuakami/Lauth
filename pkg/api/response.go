package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 通用API响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "操作成功",
		Data:    data,
	})
}

// Error 返回错误响应
func Error(c *gin.Context, code int, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Error:   errMsg,
	})
}
