package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"
)

// PermissionHandler 权限处理器
type PermissionHandler struct {
	permissionService service.PermissionService
}

// NewPermissionHandler 创建权限处理器实例
func NewPermissionHandler(permissionService service.PermissionService) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
	}
}

// Register 注册路由
func (h *PermissionHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// 将permissions路由注册为apps的子路由，统一使用:id参数
	permissions := r.Group("/apps/:id/permissions", authMiddleware.HandleAuth())
	{
		permissions.POST("", h.Create)
		permissions.GET("/:permission_id", h.Get)
		permissions.PUT("/:permission_id", h.Update)
		permissions.DELETE("/:permission_id", h.Delete)
		permissions.GET("", h.List)

		// 资源权限管理
		permissions.GET("/resource/:type", h.ListByResourceType)
	}

	// 用户权限管理
	r.GET("/apps/:id/users/:user_id/permissions", authMiddleware.HandleAuth(), h.ListUserPermissions)
}

// PermissionCreateRequest 创建权限请求
type PermissionCreateRequest struct {
	Name         string             `json:"name" binding:"required"`
	Description  string             `json:"description"`
	ResourceType model.ResourceType `json:"resource_type" binding:"required"`
	Action       model.ActionType   `json:"action" binding:"required"`
}

// Create 创建权限
func (h *PermissionHandler) Create(c *gin.Context) {
	var req PermissionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appID := c.Param("id")
	permission := &model.Permission{
		Name:         req.Name,
		Description:  req.Description,
		ResourceType: req.ResourceType,
		Action:       req.Action,
	}

	if err := h.permissionService.Create(c.Request.Context(), appID, permission); err != nil {
		switch err {
		case service.ErrPermissionNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, permission)
}

// Get 获取权限
func (h *PermissionHandler) Get(c *gin.Context) {
	id := c.Param("permission_id")
	permission, err := h.permissionService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrPermissionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permission)
}

// PermissionUpdateRequest 更新权限请求
type PermissionUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Update 更新权限
func (h *PermissionHandler) Update(c *gin.Context) {
	var req PermissionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("permission_id")
	permission, err := h.permissionService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrPermissionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	permission.Name = req.Name
	permission.Description = req.Description

	if err := h.permissionService.Update(c.Request.Context(), permission); err != nil {
		switch err {
		case service.ErrPermissionNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, permission)
}

// Delete 删除权限
func (h *PermissionHandler) Delete(c *gin.Context) {
	id := c.Param("permission_id")
	if err := h.permissionService.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrPermissionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// List 获取权限列表
func (h *PermissionHandler) List(c *gin.Context) {
	appID := c.Param("id")
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	permissions, total, err := h.permissionService.List(c.Request.Context(), appID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": permissions,
		"total": total,
	})
}

// ListByResourceType 获取资源类型的权限列表
func (h *PermissionHandler) ListByResourceType(c *gin.Context) {
	appID := c.Param("id")
	resourceType := model.ResourceType(c.Param("type"))

	permissions, err := h.permissionService.GetResourcePermissions(c.Request.Context(), appID, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// ListUserPermissions 获取用户的权限列表
func (h *PermissionHandler) ListUserPermissions(c *gin.Context) {
	appID := c.Param("id")
	userID := c.Param("user_id")

	permissions, err := h.permissionService.GetUserPermissions(c.Request.Context(), userID, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}
