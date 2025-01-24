package v1

import (
	"net/http"
	"strconv"
	"time"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// AppHandler 应用处理器
type AppHandler struct {
	appService service.AppService
}

// NewAppHandler 创建应用处理器实例
func NewAppHandler(appService service.AppService) *AppHandler {
	return &AppHandler{appService: appService}
}

// Register 注册路由
func (h *AppHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	apps := r.Group("/apps")
	{
		apps.POST("", authMiddleware.HandleAuth(), h.CreateApp)       // 需要认证
		apps.GET("/:id", authMiddleware.HandleAuth(), h.GetApp)       // 需要认证
		apps.PUT("/:id", authMiddleware.HandleAuth(), h.UpdateApp)    // 需要认证
		apps.DELETE("/:id", authMiddleware.HandleAuth(), h.DeleteApp) // 需要认证
		apps.GET("", authMiddleware.HandleAuth(), h.ListApps)         // 需要认证

		// 凭证管理接口
		apps.GET("/:id/credentials", authMiddleware.HandleAuth(), h.GetAppCredentials)    // 需要认证
		apps.POST("/:id/credentials", authMiddleware.HandleAuth(), h.ResetAppCredentials) // 需要认证
	}
}

// toAppResponse 转换为应用响应
func toAppResponse(app *model.App) model.AppResponse {
	return model.AppResponse{
		ID:          app.ID,
		Name:        app.Name,
		Description: app.Description,
		Status:      app.Status,
		CreatedAt:   app.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   app.UpdatedAt.Format(time.RFC3339),
	}
}

// CreateApp 创建应用
func (h *AppHandler) CreateApp(c *gin.Context) {
	var req model.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app := &model.App{
		Name:        req.Name,
		Description: req.Description,
		Status:      model.AppStatusEnabled,
	}

	if err := h.appService.CreateApp(c.Request.Context(), app); err != nil {
		if err == service.ErrAppExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回基本信息（不包含凭证）
	c.JSON(http.StatusCreated, toAppResponse(app))
}

// GetApp 获取应用
func (h *AppHandler) GetApp(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appService.GetApp(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toAppResponse(app))
}

// UpdateApp 更新应用
func (h *AppHandler) UpdateApp(c *gin.Context) {
	id := c.Param("id")
	var req model.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先获取现有应用
	app, err := h.appService.GetApp(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 只更新允许的字段
	app.Name = req.Name
	app.Description = req.Description
	app.Status = req.Status

	if err := h.appService.UpdateApp(c.Request.Context(), app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toAppResponse(app))
}

// DeleteApp 删除应用
func (h *AppHandler) DeleteApp(c *gin.Context) {
	id := c.Param("id")
	if err := h.appService.DeleteApp(c.Request.Context(), id); err != nil {
		if err == service.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListApps 获取应用列表
func (h *AppHandler) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	apps, total, err := h.appService.ListApps(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为响应对象列表
	var responses []model.AppResponse
	for _, app := range apps {
		responses = append(responses, toAppResponse(&app))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"meta": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetAppCredentials 获取应用凭证
func (h *AppHandler) GetAppCredentials(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appService.GetApp(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.AppCredentialsResponse{
		AppKey:    app.AppKey,
		AppSecret: app.AppSecret,
	})
}

// ResetAppCredentials 重置应用凭证
func (h *AppHandler) ResetAppCredentials(c *gin.Context) {
	id := c.Param("id")
	app, err := h.appService.ResetCredentials(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrAppNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.AppCredentialsResponse{
		AppKey:    app.AppKey,
		AppSecret: app.AppSecret,
	})
}
