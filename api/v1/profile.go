package v1

import (
	"net/http"
	"strconv"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// ProfileHandler Profile处理器
type ProfileHandler struct {
	profileService service.ProfileService
}

// NewProfileHandler 创建Profile处理器实例
func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// Register 注册路由
func (h *ProfileHandler) Register(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	profiles := group.Group("/profiles")
	profiles.Use(authMiddleware.HandleAuth())
	{
		profiles.POST("", h.CreateProfile)
		profiles.GET("/:id", h.GetProfile)
		profiles.PUT("/:id", h.UpdateProfile)
		profiles.DELETE("/:id", h.DeleteProfile)
		profiles.GET("", h.ListProfiles)
		profiles.GET("/me", h.GetMyProfile)
		profiles.PUT("/:id/custom_data", h.UpdateCustomData)
	}
}

// CreateProfile 创建用户档案
func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	var req model.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.CreateProfile(c.Request.Context(), claims.UserID, claims.AppID, &req)
	if err != nil {
		switch err {
		case service.ErrProfileExists:
			c.JSON(http.StatusConflict, gin.H{"error": "profile already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, profile)
}

// GetProfile 获取用户档案
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	id := c.Param("id")

	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), id)
	if err != nil {
		switch err {
		case service.ErrProfileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateProfile 更新用户档案
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	id := c.Param("id")

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.profileService.UpdateProfileByUserID(c.Request.Context(), id, &req)
	if err != nil {
		switch err {
		case service.ErrProfileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, profile)
}

// DeleteProfile 删除用户档案
func (h *ProfileHandler) DeleteProfile(c *gin.Context) {
	id := c.Param("id")

	if err := h.profileService.DeleteProfile(c.Request.Context(), id); err != nil {
		switch err {
		case service.ErrProfileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListProfiles 获取用户档案列表
func (h *ProfileHandler) ListProfiles(c *gin.Context) {
	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	profiles, total, err := h.profileService.ListProfiles(c.Request.Context(), claims.AppID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"items": profiles,
	})
}

// GetMyProfile 获取当前用户的档案
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), claims.UserID)
	if err != nil {
		switch err {
		case service.ErrProfileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateCustomData 更新自定义数据
func (h *ProfileHandler) UpdateCustomData(c *gin.Context) {
	id := c.Param("id")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.profileService.UpdateCustomData(c.Request.Context(), id, data); err != nil {
		switch err {
		case service.ErrProfileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
