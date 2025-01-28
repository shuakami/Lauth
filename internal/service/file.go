package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	// ErrFileNotFound 文件不存在
	ErrFileNotFound = errors.New("file not found")
	// ErrInvalidFileType 无效的文件类型
	ErrInvalidFileType = errors.New("invalid file type")
	// ErrFileTooLarge 文件太大
	ErrFileTooLarge = errors.New("file too large")
)

const (
	// MaxFileSize 最大文件大小 (10MB)
	MaxFileSize = 10 * 1024 * 1024
	// UploadDir 上传目录
	UploadDir = "uploads"
)

// FileService 文件服务接口
type FileService interface {
	// UploadFile 上传文件
	UploadFile(ctx context.Context, userID, appID string, file *multipart.FileHeader, req *model.FileUploadRequest) (*model.File, error)
	// UpdateFile 更新文件信息
	UpdateFile(ctx context.Context, id string, req *model.FileUpdateRequest) (*model.File, error)
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, id string) error
	// GetFile 获取文件信息
	GetFile(ctx context.Context, id string) (*model.File, error)
	// ListFiles 获取文件列表
	ListFiles(ctx context.Context, userID, appID string, page, pageSize int) ([]*model.File, int64, error)
	// ListFilesByTags 通过标签获取文件列表
	ListFilesByTags(ctx context.Context, userID, appID string, tags []string, page, pageSize int) ([]*model.File, error)
	// UpdateCustomData 更新自定义数据
	UpdateCustomData(ctx context.Context, id string, data map[string]interface{}) error
}

// fileService 文件服务实现
type fileService struct {
	fileRepo repository.FileRepository
}

// NewFileService 创建文件服务实例
func NewFileService(fileRepo repository.FileRepository) FileService {
	return &fileService{fileRepo: fileRepo}
}

// UploadFile 上传文件
func (s *fileService) UploadFile(ctx context.Context, userID, appID string, fileHeader *multipart.FileHeader, req *model.FileUploadRequest) (*model.File, error) {
	// 检查文件大小
	if fileHeader.Size > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// 打开源文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// 计算文件哈希
	hash := sha256.New()
	if _, err := io.Copy(hash, src); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 重置文件指针
	if _, err := src.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// 创建上传目录
	uploadPath := filepath.Join(UploadDir, appID, userID)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("%s_%s", time.Now().Format("20060102150405"), fileHeader.Filename)
	filePath := filepath.Join(uploadPath, fileName)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err = io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// 创建文件记录
	file := &model.File{
		UserID:     userID,
		AppID:      appID,
		Name:       fileHeader.Filename,
		Type:       fileHeader.Header.Get("Content-Type"),
		Size:       fileHeader.Size,
		Path:       filePath,
		URL:        fmt.Sprintf("/files/%s/%s/%s", appID, userID, fileName),
		Hash:       fileHash,
		Tags:       req.Tags,
		CustomData: req.CustomData,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		// 删除已上传的文件
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return file, nil
}

// UpdateFile 更新文件信息
func (s *fileService) UpdateFile(ctx context.Context, id string, req *model.FileUpdateRequest) (*model.File, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	file, err := s.fileRepo.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}

	if req.Name != nil {
		file.Name = *req.Name
	}
	if req.Tags != nil {
		file.Tags = req.Tags
	}
	if req.CustomData != nil {
		file.CustomData = req.CustomData
	}

	if err := s.fileRepo.Update(ctx, objectID, file); err != nil {
		return nil, err
	}

	return file, nil
}

// DeleteFile 删除文件
func (s *fileService) DeleteFile(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	file, err := s.fileRepo.GetByID(ctx, objectID)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrFileNotFound
	}

	// 删除物理文件
	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete physical file: %w", err)
	}

	// 删除文件记录
	return s.fileRepo.Delete(ctx, objectID)
}

// GetFile 获取文件信息
func (s *fileService) GetFile(ctx context.Context, id string) (*model.File, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	file, err := s.fileRepo.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}

	return file, nil
}

// ListFiles 获取文件列表
func (s *fileService) ListFiles(ctx context.Context, userID, appID string, page, pageSize int) ([]*model.File, int64, error) {
	offset := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	files, err := s.fileRepo.List(ctx, userID, appID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.fileRepo.Count(ctx, userID, appID)
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// ListFilesByTags 通过标签获取文件列表
func (s *fileService) ListFilesByTags(ctx context.Context, userID, appID string, tags []string, page, pageSize int) ([]*model.File, error) {
	offset := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	return s.fileRepo.ListByTags(ctx, userID, appID, tags, offset, limit)
}

// UpdateCustomData 更新自定义数据
func (s *fileService) UpdateCustomData(ctx context.Context, id string, data map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	return s.fileRepo.UpdateCustomData(ctx, objectID, data)
}
