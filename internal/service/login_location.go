package service

import (
	"context"
	"time"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// LoginLocationService 登录位置服务接口
type LoginLocationService interface {
	// 记录登录位置
	RecordLoginLocation(ctx context.Context, appID, userID, ip string) error
	// 获取用户最近的登录位置
	GetLatestLocations(ctx context.Context, userID string, limit int) ([]*types.LocationResponse, error)
	// 获取用户指定时间范围内的登录位置
	GetLocationsByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*types.LocationResponse, error)
}

// loginLocationService 登录位置服务实现
type loginLocationService struct {
	locationRepo repository.LoginLocationRepository
	ipService    IPLocationService
}

// NewLoginLocationService 创建登录位置服务实例
func NewLoginLocationService(
	locationRepo repository.LoginLocationRepository,
	ipService IPLocationService,
) LoginLocationService {
	return &loginLocationService{
		locationRepo: locationRepo,
		ipService:    ipService,
	}
}

// RecordLoginLocation 记录登录位置
func (s *loginLocationService) RecordLoginLocation(ctx context.Context, appID, userID, ip string) error {
	// 解析IP地理位置
	location, err := s.ipService.SearchIP(ctx, ip)
	if err != nil {
		return err
	}

	// 创建登录位置记录
	loginLocation := &model.LoginLocation{
		AppID:     appID,
		UserID:    userID,
		IP:        ip,
		Country:   location.Country,
		Province:  location.Province,
		City:      location.City,
		ISP:       location.ISP,
		LoginTime: time.Now(),
	}

	return s.locationRepo.Create(ctx, loginLocation)
}

// GetLatestLocations 获取用户最近的登录位置
func (s *loginLocationService) GetLatestLocations(ctx context.Context, userID string, limit int) ([]*types.LocationResponse, error) {
	locations, err := s.locationRepo.GetLatestByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	responses := make([]*types.LocationResponse, len(locations))
	for i, loc := range locations {
		responses[i] = &types.LocationResponse{
			ID:        loc.ID,
			IP:        loc.IP,
			Country:   loc.Country,
			Province:  loc.Province,
			City:      loc.City,
			ISP:       loc.ISP,
			LoginTime: loc.LoginTime,
		}
	}

	return responses, nil
}

// GetLocationsByTimeRange 获取用户指定时间范围内的登录位置
func (s *loginLocationService) GetLocationsByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*types.LocationResponse, error) {
	locations, err := s.locationRepo.GetByUserIDAndTimeRange(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	responses := make([]*types.LocationResponse, len(locations))
	for i, loc := range locations {
		responses[i] = &types.LocationResponse{
			ID:        loc.ID,
			IP:        loc.IP,
			Country:   loc.Country,
			Province:  loc.Province,
			City:      loc.City,
			ISP:       loc.ISP,
			LoginTime: loc.LoginTime,
		}
	}

	return responses, nil
}
