package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrAppNotFound = errors.New("app not found")
	ErrAppExists   = errors.New("app already exists")
)

// AppService 应用服务接口
type AppService interface {
	CreateApp(ctx context.Context, app *model.App) error
	GetApp(ctx context.Context, id string) (*model.App, error)
	UpdateApp(ctx context.Context, app *model.App) error
	DeleteApp(ctx context.Context, id string) error
	ListApps(ctx context.Context, page, pageSize int) ([]model.App, int64, error)
	ValidateApp(ctx context.Context, appKey, appSecret string) (*model.App, error)
	ResetCredentials(ctx context.Context, id string) (*model.App, error)
}

// appService 应用服务实现
type appService struct {
	repo repository.AppRepository
}

// NewAppService 创建应用服务实例
func NewAppService(repo repository.AppRepository) AppService {
	return &appService{repo: repo}
}

// CreateApp 创建应用
func (s *appService) CreateApp(ctx context.Context, app *model.App) error {
	// 检查应用名是否已存在
	existingApp, err := s.repo.GetByAppKey(ctx, app.AppKey)
	if err != nil {
		return err
	}
	if existingApp != nil {
		return ErrAppExists
	}

	return s.repo.Create(ctx, app)
}

// GetApp 获取应用
func (s *appService) GetApp(ctx context.Context, id string) (*model.App, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}
	return app, nil
}

// UpdateApp 更新应用
func (s *appService) UpdateApp(ctx context.Context, app *model.App) error {
	existingApp, err := s.repo.GetByID(ctx, app.ID)
	if err != nil {
		return err
	}
	if existingApp == nil {
		return ErrAppNotFound
	}

	return s.repo.Update(ctx, app)
}

// DeleteApp 删除应用
func (s *appService) DeleteApp(ctx context.Context, id string) error {
	existingApp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existingApp == nil {
		return ErrAppNotFound
	}

	return s.repo.Delete(ctx, id)
}

// ListApps 获取应用列表
func (s *appService) ListApps(ctx context.Context, page, pageSize int) ([]model.App, int64, error) {
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, offset, pageSize)
}

// ValidateApp 验证应用凭证
func (s *appService) ValidateApp(ctx context.Context, appKey, appSecret string) (*model.App, error) {
	app, err := s.repo.GetByAppKey(ctx, appKey)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}

	if app.AppSecret != appSecret {
		return nil, errors.New("invalid app secret")
	}

	if app.Status == model.AppStatusDisabled {
		return nil, errors.New("app is disabled")
	}

	return app, nil
}

// ResetCredentials 重置应用凭证
func (s *appService) ResetCredentials(ctx context.Context, id string) (*model.App, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}

	// 生成新的凭证
	app.AppKey = uuid.New().String()
	app.AppSecret = uuid.NewString()

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}
