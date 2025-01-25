package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"
)

// RoleHandler 角色处理器
type RoleHandler struct {
	roleService service.RoleService
}

// NewRoleHandler 创建角色处理器实例
func NewRoleHandler(roleService service.RoleService) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
	}
}

// Register 注册路由
func (h *RoleHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// 将roles路由注册为apps的子路由，统一使用:id参数
	roles := r.Group("/apps/:id/roles", authMiddleware.HandleAuth())
	{
		roles.POST("", h.Create)
		roles.GET("/:role_id", h.Get)
		roles.PUT("/:role_id", h.Update)
		roles.DELETE("/:role_id", h.Delete)
		roles.GET("", h.List)

		// 角色权限管理
		rolePerms := roles.Group("/:role_id/permissions")
		{
			rolePerms.POST("", h.AddPermissions)
			rolePerms.DELETE("", h.RemovePermissions)
			rolePerms.GET("", h.GetPermissions)
		}

		// 角色用户管理
		roleUsers := roles.Group("/:role_id/users")
		{
			roleUsers.POST("", h.AddUsers)
			roleUsers.DELETE("", h.RemoveUsers)
			roleUsers.GET("", h.GetUsers)
		}
	}
}

// RoleCreateRequest 创建角色请求
type RoleCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
}

// Create 创建角色
func (h *RoleHandler) Create(c *gin.Context) {
	var req RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appID := c.Param("id")
	role := &model.Role{
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    req.IsSystem,
	}

	if err := h.roleService.Create(c.Request.Context(), appID, role); err != nil {
		switch err {
		case service.ErrRoleNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, role)
}

// Get 获取角色
func (h *RoleHandler) Get(c *gin.Context) {
	id := c.Param("role_id")
	role, err := h.roleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// RoleUpdateRequest 更新角色请求
type RoleUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Update 更新角色
func (h *RoleHandler) Update(c *gin.Context) {
	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("role_id")
	role, err := h.roleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	role.Name = req.Name
	role.Description = req.Description

	if err := h.roleService.Update(c.Request.Context(), role); err != nil {
		switch err {
		case service.ErrRoleNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrRoleSystemLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, role)
}

// Delete 删除角色
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("role_id")
	if err := h.roleService.Delete(c.Request.Context(), id); err != nil {
		switch err {
		case service.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrRoleSystemLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// List 获取角色列表
func (h *RoleHandler) List(c *gin.Context) {
	appID := c.Param("id")
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	roles, total, err := h.roleService.List(c.Request.Context(), appID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": roles,
		"total": total,
	})
}

// PermissionRequest 权限请求
type PermissionRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required"`
}

// AddPermissions 为角色添加权限
func (h *RoleHandler) AddPermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleID := c.Param("role_id")
	if err := h.roleService.AddPermissions(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		switch err {
		case service.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrRoleSystemLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// RemovePermissions 移除角色的权限
func (h *RoleHandler) RemovePermissions(c *gin.Context) {
	var req PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleID := c.Param("role_id")
	if err := h.roleService.RemovePermissions(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		switch err {
		case service.ErrRoleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrRoleSystemLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// GetPermissions 获取角色的权限列表
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	roleID := c.Param("role_id")
	permissions, err := h.roleService.GetPermissions(c.Request.Context(), roleID)
	if err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// UserRequest 用户请求
type UserRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

// AddUsers 为角色添加用户
func (h *RoleHandler) AddUsers(c *gin.Context) {
	roleID := c.Param("role_id")
	if roleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色ID不能为空"})
		return
	}

	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID列表不能为空"})
		return
	}

	if err := h.roleService.AddUsers(c.Request.Context(), roleID, req.UserIDs); err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveUsers 移除角色的用户
func (h *RoleHandler) RemoveUsers(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleID := c.Param("role_id")
	if err := h.roleService.RemoveUsers(c.Request.Context(), roleID, req.UserIDs); err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetUsers 获取角色的用户列表
func (h *RoleHandler) GetUsers(c *gin.Context) {
	roleID := c.Param("role_id")
	users, err := h.roleService.GetUsers(c.Request.Context(), roleID)
	if err != nil {
		if err == service.ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}
