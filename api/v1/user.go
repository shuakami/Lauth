package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService       service.UserService
	authService       service.AuthService
	superAdminService service.SuperAdminService
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, authService service.AuthService, superAdminService service.SuperAdminService) *UserHandler {
	return &UserHandler{
		userService:       userService,
		authService:       authService,
		superAdminService: superAdminService,
	}
}

// Register 注册路由
func (h *UserHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// 将users路由注册为apps的子路由，统一使用:id参数
	r.POST("/apps/:id/users", h.CreateUser)                                                              // 创建用户不需要认证
	r.GET("/apps/:id/users/:user_id", authMiddleware.HandleAuth(), h.GetUser)                            // 需要认证
	r.PUT("/apps/:id/users/:user_id", authMiddleware.HandleAuth(), h.UpdateUser)                         // 需要认证
	r.DELETE("/apps/:id/users/:user_id", authMiddleware.HandleAuth(), h.DeleteUser)                      // 需要认证
	r.GET("/apps/:id/users", authMiddleware.HandleAuth(), h.ListUsers)                                   // 需要认证
	r.PUT("/apps/:id/users/:user_id/password", authMiddleware.HandleAuth(), h.UpdatePassword)            // 需要认证
	r.PUT("/apps/:id/users/:user_id/first-password", authMiddleware.HandleAuth(), h.FirstChangePassword) // 首次修改密码，需要认证
}

// toUserResponse 转换为用户响应
func toUserResponse(user *model.User, profile *model.Profile) model.UserResponse {
	var lastLoginStr *string
	if user.LastLoginAt != nil {
		formatted := user.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		lastLoginStr = &formatted
	}

	// 转换密码过期时间
	var passwordExpiresStr *string
	if user.PasswordExpiresAt != nil {
		formatted := user.PasswordExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		passwordExpiresStr = &formatted
	}

	// 只检查是否是首次登录，不再检查密码是否过期
	// 密码过期信息由前端根据PasswordExpiresAt自行判断
	needChangePassword := user.IsFirstLogin

	return model.UserResponse{
		ID:                 user.ID,
		AppID:              user.AppID,
		Username:           user.Username,
		Nickname:           user.Nickname,
		Email:              user.Email,
		Phone:              user.Phone,
		Status:             user.Status,
		Profile:            profile,
		IsFirstLogin:       user.IsFirstLogin,
		LastLoginAt:        lastLoginStr,
		PasswordExpiresAt:  passwordExpiresStr,
		NeedChangePassword: needChangePassword,
		CreatedAt:          user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// CreateUser 创建用户
func (h *UserHandler) CreateUser(c *gin.Context) {
	fmt.Println("=== 开始创建用户 ===")
	appID := c.Param("id") // 使用统一的id参数
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("请求参数错误:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("调用Register方法")
	response, err := h.authService.Register(c.Request.Context(), appID, &req)
	fmt.Printf("Register返回: response=%+v, err=%v\n", response, err)

	if err != nil {
		fmt.Println("处理Register错误")
		switch err {
		case service.ErrAppNotFound:
			fmt.Println("应用不存在")
			c.JSON(http.StatusNotFound, gin.H{"error": "app not found"})
		case service.ErrUserExists:
			fmt.Println("用户已存在")
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrPluginRequired:
			fmt.Println("需要验证，返回202")
			// 需要验证时返回202状态码和验证信息
			c.JSON(http.StatusAccepted, gin.H{
				"auth_status": response.AuthStatus,
				"plugins":     response.Plugins,
				"next_plugin": response.NextPlugin,
				"session_id":  response.SessionID,
			})
			return
		default:
			fmt.Println("其他错误:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	fmt.Println("开始获取用户档案")
	// 获取用户档案
	_, profile, err := h.userService.GetUserWithProfile(c.Request.Context(), response.User.ID)
	if err != nil && err != service.ErrProfileNotFound {
		fmt.Println("获取用户档案失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("构建用户响应")
	// 构建用户响应
	userResponse := model.UserResponse{
		ID:        response.User.ID,
		AppID:     response.User.AppID,
		Username:  response.User.Username,
		Nickname:  response.User.Nickname,
		Email:     response.User.Email,
		Phone:     response.User.Phone,
		Status:    response.User.Status,
		Profile:   profile,
		CreatedAt: response.User.CreatedAt,
		UpdatedAt: response.User.UpdatedAt,
	}

	fmt.Println("返回创建成功响应")
	c.JSON(http.StatusCreated, userResponse)
}

// GetUser 获取用户
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("user_id")
	user, profile, err := h.userService.GetUserWithProfile(c.Request.Context(), id)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user, profile))
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("user_id")
	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 获取更新后的用户档案
	_, profile, err := h.userService.GetUserWithProfile(c.Request.Context(), user.ID)
	if err != nil && err != service.ErrProfileNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user, profile))
}

// UpdatePassword 更新密码
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	id := c.Param("user_id")

	// 获取当前登录用户
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 安全检查：只有用户本人或超级管理员才能修改密码
	if claims.UserID != id {
		// 检查当前用户是否是超级管理员
		isSuperAdmin, err := h.superAdminService.IsSuperAdmin(c.Request.Context(), claims.UserID)
		if err != nil || !isSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied: only the user or super admin can change password"})
			c.Abort()
			return
		}
	}

	var req model.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.UpdatePassword(c.Request.Context(), id, &req); err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrInvalidPassword:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 密码更新成功，返回204 No Content
	c.Status(http.StatusNoContent)
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("user_id")
	if err := h.userService.DeleteUser(c.Request.Context(), id); err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers 获取用户列表
func (h *UserHandler) ListUsers(c *gin.Context) {
	appID := c.Param("id") // 使用统一的id参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, total, err := h.userService.ListUsers(c.Request.Context(), appID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var responses []model.UserResponse
	for _, user := range users {
		// 获取用户档案
		_, profile, err := h.userService.GetUserWithProfile(c.Request.Context(), user.ID)
		if err != nil && err != service.ErrProfileNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		responses = append(responses, toUserResponse(&user, profile))
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

// GetUserInfo 获取当前用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 从上下文中获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// 获取用户详细信息
	user, profile, err := h.userService.GetUserWithProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user, profile))
}

// FirstChangePassword 首次修改密码（不需要旧密码）- 仅限超级管理员使用
func (h *UserHandler) FirstChangePassword(c *gin.Context) {
	// 不使用appID变量，避免linter错误
	_ = c.Param("id")
	id := c.Param("user_id")

	// 获取当前登录用户
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 安全检查：只有超级管理员才能使用此接口修改密码
	isSuperAdmin, err := h.superAdminService.IsSuperAdmin(c.Request.Context(), claims.UserID)
	if err != nil || !isSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied: only super admin can use this endpoint"})
		c.Abort()
		return
	}

	var req model.FirstChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.FirstChangePassword(c.Request.Context(), id, &req); err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update password: %v", err)})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
