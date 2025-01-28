package v1

import (
	"net/http"
	"strconv"
	"strings"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	fileService service.FileService
}

// NewFileHandler 创建文件处理器实例
func NewFileHandler(fileService service.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// Register 注册路由
func (h *FileHandler) Register(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	files := group.Group("/files")
	files.Use(authMiddleware.HandleAuth())
	{
		files.POST("", h.UploadFile)
		files.GET("/:id", h.GetFile)
		files.PUT("/:id", h.UpdateFile)
		files.DELETE("/:id", h.DeleteFile)
		files.GET("", h.ListFiles)
		files.GET("/tags", h.ListFilesByTags)
		files.PUT("/:id/custom_data", h.UpdateCustomData)
	}
}

// UploadFile 上传文件
func (h *FileHandler) UploadFile(c *gin.Context) {
	// 获取文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}

	// 获取其他参数
	var req model.FileUploadRequest
	if tagsStr := c.PostForm("tags"); tagsStr != "" {
		req.Tags = strings.Split(tagsStr, ",")
	}
	if customDataStr := c.PostForm("custom_data"); customDataStr != "" {
		var customData map[string]interface{}
		if err := c.ShouldBindJSON(&customData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid custom data format"})
			return
		}
		req.CustomData = customData
	}

	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 上传文件
	uploadedFile, err := h.fileService.UploadFile(c.Request.Context(), claims.UserID, claims.AppID, file, &req)
	if err != nil {
		switch err {
		case service.ErrFileTooLarge:
			c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
		case service.ErrInvalidFileType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, uploadedFile)
}

// GetFile 获取文件信息
func (h *FileHandler) GetFile(c *gin.Context) {
	id := c.Param("id")

	file, err := h.fileService.GetFile(c.Request.Context(), id)
	if err != nil {
		switch err {
		case service.ErrFileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, file)
}

// UpdateFile 更新文件信息
func (h *FileHandler) UpdateFile(c *gin.Context) {
	id := c.Param("id")

	var req model.FileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := h.fileService.UpdateFile(c.Request.Context(), id, &req)
	if err != nil {
		switch err {
		case service.ErrFileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, file)
}

// DeleteFile 删除文件
func (h *FileHandler) DeleteFile(c *gin.Context) {
	id := c.Param("id")

	if err := h.fileService.DeleteFile(c.Request.Context(), id); err != nil {
		switch err {
		case service.ErrFileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListFiles 获取文件列表
func (h *FileHandler) ListFiles(c *gin.Context) {
	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	files, total, err := h.fileService.ListFiles(c.Request.Context(), claims.UserID, claims.AppID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"items": files,
	})
}

// ListFilesByTags 通过标签获取文件列表
func (h *FileHandler) ListFilesByTags(c *gin.Context) {
	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 获取标签参数
	tagsStr := c.Query("tags")
	if tagsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tags parameter is required"})
		return
	}
	tags := strings.Split(tagsStr, ",")

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	files, err := h.fileService.ListFilesByTags(c.Request.Context(), claims.UserID, claims.AppID, tags, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": files,
	})
}

// UpdateCustomData 更新自定义数据
func (h *FileHandler) UpdateCustomData(c *gin.Context) {
	id := c.Param("id")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.fileService.UpdateCustomData(c.Request.Context(), id, data); err != nil {
		switch err {
		case service.ErrFileNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
