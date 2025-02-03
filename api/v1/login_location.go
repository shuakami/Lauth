package v1

import (
	"net/http"
	"time"

	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// LoginLocationHandler 登录位置处理器
type LoginLocationHandler struct {
	locationService service.LoginLocationService
}

// NewLoginLocationHandler 创建登录位置处理器实例
func NewLoginLocationHandler(locationService service.LoginLocationService) *LoginLocationHandler {
	return &LoginLocationHandler{
		locationService: locationService,
	}
}

// Register 注册路由
func (h *LoginLocationHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	locations := r.Group("/apps/:id/users/:user_id/locations", authMiddleware.HandleAuth())
	{
		locations.GET("", h.GetLatestLocations)
		locations.GET("/history", h.GetLocationHistory)
	}
}

// GetLatestLocationsRequest 获取最近登录位置请求
type GetLatestLocationsRequest struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// GetLatestLocations 获取最近登录位置
func (h *LoginLocationHandler) GetLatestLocations(c *gin.Context) {
	var req GetLatestLocationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// 如果没有指定limit，使用默认值10
	if req.Limit <= 0 {
		req.Limit = 10
	}

	locations, err := h.locationService.GetLatestLocations(c.Request.Context(), userID, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, locations)
}

// GetLocationHistoryRequest 获取登录位置历史请求
type GetLocationHistoryRequest struct {
	StartTime time.Time `form:"start_time" binding:"omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime   time.Time `form:"end_time" binding:"omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
}

// GetLocationHistory 获取登录位置历史
func (h *LoginLocationHandler) GetLocationHistory(c *gin.Context) {
	var req GetLocationHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// 如果没有指定时间范围，默认查询最近7天
	if req.StartTime.IsZero() {
		req.StartTime = time.Now().AddDate(0, 0, -7) // 7天前
	}
	if req.EndTime.IsZero() {
		req.EndTime = time.Now()
	}

	// 验证时间范围
	if req.EndTime.Before(req.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_time must be after start_time"})
		return
	}

	// 限制最大查询范围为30天
	if req.EndTime.Sub(req.StartTime) > 30*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "time range cannot exceed 30 days"})
		return
	}

	locations, err := h.locationService.GetLocationsByTimeRange(c.Request.Context(), userID, req.StartTime, req.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, locations)
}
