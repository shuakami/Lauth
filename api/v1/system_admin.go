package v1

import (
	"lauth/internal/service"
	"lauth/pkg/api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuperAdminHandler 超级管理员处理器
type SuperAdminHandler struct {
	superAdminService service.SuperAdminService
	userService       service.UserService
}

// NewSuperAdminHandler 创建超级管理员处理器
func NewSuperAdminHandler(superAdminService service.SuperAdminService, userService service.UserService) *SuperAdminHandler {
	return &SuperAdminHandler{
		superAdminService: superAdminService,
		userService:       userService,
	}
}

// SuperAdminUserRequest 添加/删除超级管理员的请求
type SuperAdminUserRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// AddSuperAdmin 添加超级管理员
// @Summary 添加超级管理员
// @Description 将用户添加为超级管理员
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body SuperAdminUserRequest true "用户ID"
// @Success 200 {object} api.Response "成功添加超级管理员"
// @Failure 400 {object} api.Response "请求参数错误"
// @Failure 404 {object} api.Response "用户不存在"
// @Failure 409 {object} api.Response "用户已经是超级管理员"
// @Failure 500 {object} api.Response "服务器内部错误"
// @Router /api/v1/system/super-admins [post]
func (h *SuperAdminHandler) AddSuperAdmin(c *gin.Context) {
	var req SuperAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 添加超级管理员
	err := h.superAdminService.AddSuperAdmin(c.Request.Context(), req.UserID)
	if err != nil {
		switch err {
		case service.ErrUserNotExists:
			api.Error(c, http.StatusNotFound, "用户不存在", err)
		case service.ErrSuperAdminExists:
			api.Error(c, http.StatusConflict, "该用户已经是超级管理员", err)
		default:
			api.Error(c, http.StatusInternalServerError, "添加超级管理员失败", err)
		}
		return
	}

	api.Success(c, gin.H{
		"message": "成功添加超级管理员",
		"user_id": req.UserID,
	})
}

// RemoveSuperAdmin 移除超级管理员
// @Summary 移除超级管理员
// @Description 移除超级管理员权限
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param user_id path string true "用户ID"
// @Success 200 {object} api.Response "成功移除超级管理员"
// @Failure 404 {object} api.Response "超级管理员不存在"
// @Failure 400 {object} api.Response "不能删除最后一个超级管理员"
// @Failure 500 {object} api.Response "服务器内部错误"
// @Router /api/v1/system/super-admins/{user_id} [delete]
func (h *SuperAdminHandler) RemoveSuperAdmin(c *gin.Context) {
	userID := c.Param("user_id")

	// 移除超级管理员
	err := h.superAdminService.RemoveSuperAdmin(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case service.ErrSuperAdminNotExists:
			api.Error(c, http.StatusNotFound, "该用户不是超级管理员", err)
		case service.ErrLastSuperAdmin:
			api.Error(c, http.StatusBadRequest, "不能删除最后一个超级管理员", err)
		default:
			api.Error(c, http.StatusInternalServerError, "移除超级管理员失败", err)
		}
		return
	}

	api.Success(c, gin.H{
		"message": "成功移除超级管理员",
		"user_id": userID,
	})
}

// ListSuperAdmins 获取所有超级管理员
// @Summary 获取所有超级管理员
// @Description 获取系统中所有超级管理员列表
// @Tags 系统管理
// @Produce json
// @Success 200 {object} api.Response{data=[]model.SuperAdmin} "超级管理员列表"
// @Failure 500 {object} api.Response "服务器内部错误"
// @Router /api/v1/system/super-admins [get]
func (h *SuperAdminHandler) ListSuperAdmins(c *gin.Context) {
	// 获取所有超级管理员
	superAdmins, err := h.superAdminService.ListSuperAdmins(c.Request.Context())
	if err != nil {
		api.Error(c, http.StatusInternalServerError, "获取超级管理员列表失败", err)
		return
	}

	api.Success(c, gin.H{
		"total": len(superAdmins),
		"data":  superAdmins,
	})
}

// CheckSuperAdmin 检查用户是否是超级管理员
// @Summary 检查用户是否是超级管理员
// @Description 检查指定用户是否拥有超级管理员权限
// @Tags 系统管理
// @Produce json
// @Param user_id path string true "用户ID"
// @Success 200 {object} api.Response "用户的超级管理员状态"
// @Failure 500 {object} api.Response "服务器内部错误"
// @Router /api/v1/system/super-admins/check/{user_id} [get]
func (h *SuperAdminHandler) CheckSuperAdmin(c *gin.Context) {
	userID := c.Param("user_id")

	// 检查用户是否是超级管理员
	isSuperAdmin, err := h.superAdminService.IsSuperAdmin(c.Request.Context(), userID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, "检查超级管理员权限失败", err)
		return
	}

	status := "非超级管理员"
	if isSuperAdmin {
		status = "超级管理员"
	}

	api.Success(c, gin.H{
		"user_id":        userID,
		"is_super_admin": isSuperAdmin,
		"status":         status,
	})
}
