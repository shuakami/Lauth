package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	// ErrProfileNotFound 档案不存在
	ErrProfileNotFound = errors.New("profile not found")
	// ErrProfileExists 档案已存在
	ErrProfileExists = errors.New("profile already exists")
)

// ProfileService Profile服务接口
type ProfileService interface {
	// CreateProfile 创建用户档案
	CreateProfile(ctx context.Context, userID, appID string, req *model.CreateProfileRequest) (*model.Profile, error)
	// UpdateProfile 更新用户档案
	UpdateProfile(ctx context.Context, id string, req *model.UpdateProfileRequest) (*model.Profile, error)
	// UpdateProfileByUserID 通过用户ID更新档案
	UpdateProfileByUserID(ctx context.Context, userID string, req *model.UpdateProfileRequest) (*model.Profile, error)
	// DeleteProfile 删除用户档案
	DeleteProfile(ctx context.Context, id string) error
	// GetProfile 获取用户档案
	GetProfile(ctx context.Context, id string) (*model.Profile, error)
	// GetProfileByUserID 通过用户ID获取档案
	GetProfileByUserID(ctx context.Context, userID string) (*model.Profile, error)
	// ListProfiles 获取用户档案列表
	ListProfiles(ctx context.Context, appID string, page, pageSize int) ([]*model.Profile, int64, error)
	// UpdateCustomData 更新自定义数据
	UpdateCustomData(ctx context.Context, id string, data map[string]interface{}) error
}

// profileService Profile服务实现
type profileService struct {
	profileRepo repository.ProfileRepository
	fileRepo    repository.FileRepository
}

// NewProfileService 创建Profile服务实例
func NewProfileService(profileRepo repository.ProfileRepository, fileRepo repository.FileRepository) ProfileService {
	return &profileService{
		profileRepo: profileRepo,
		fileRepo:    fileRepo,
	}
}

// CreateProfile 创建用户档案
func (s *profileService) CreateProfile(ctx context.Context, userID, appID string, req *model.CreateProfileRequest) (*model.Profile, error) {
	// 检查是否已存在档案
	existingProfile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existingProfile != nil {
		return nil, ErrProfileExists
	}

	// 创建新档案
	profile := &model.Profile{
		UserID:     userID,
		AppID:      appID,
		Avatar:     req.Avatar,
		Nickname:   req.Nickname,
		RealName:   req.RealName,
		Gender:     req.Gender,
		Birthday:   req.Birthday,
		Email:      req.Email,
		Phone:      req.Phone,
		Address:    req.Address,
		Social:     req.Social,
		CustomData: req.CustomData,
	}

	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateProfile 更新用户档案
func (s *profileService) UpdateProfile(ctx context.Context, id string, req *model.UpdateProfileRequest) (*model.Profile, error) {
	// 解析ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// 获取现有档案
	profile, err := s.profileRepo.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	// 更新字段
	if req.Avatar != nil {
		profile.Avatar = *req.Avatar
	}
	if req.Nickname != nil {
		profile.Nickname = *req.Nickname
	}
	if req.RealName != nil {
		profile.RealName = *req.RealName
	}
	if req.Gender != nil {
		profile.Gender = *req.Gender
	}
	if req.Birthday != nil {
		profile.Birthday = req.Birthday
	}
	if req.Email != nil {
		profile.Email = *req.Email
	}
	if req.Phone != nil {
		profile.Phone = *req.Phone
	}
	if req.Address != nil {
		profile.Address = req.Address
	}
	if req.Social != nil {
		profile.Social = req.Social
	}
	if req.CustomData != nil {
		profile.CustomData = req.CustomData
	}

	// 更新档案
	if err := s.profileRepo.Update(ctx, objectID, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// DeleteProfile 删除用户档案
func (s *profileService) DeleteProfile(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	return s.profileRepo.Delete(ctx, objectID)
}

// GetProfile 获取用户档案
func (s *profileService) GetProfile(ctx context.Context, id string) (*model.Profile, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.GetByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	return profile, nil
}

// GetProfileByUserID 通过用户ID获取档案
func (s *profileService) GetProfileByUserID(ctx context.Context, userID string) (*model.Profile, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	return profile, nil
}

// ListProfiles 获取用户档案列表
func (s *profileService) ListProfiles(ctx context.Context, appID string, page, pageSize int) ([]*model.Profile, int64, error) {
	offset := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	profiles, err := s.profileRepo.List(ctx, appID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.profileRepo.Count(ctx, appID)
	if err != nil {
		return nil, 0, err
	}

	return profiles, total, nil
}

// UpdateCustomData 更新自定义数据
func (s *profileService) UpdateCustomData(ctx context.Context, id string, data map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	return s.profileRepo.UpdateCustomData(ctx, objectID, data)
}

// UpdateProfileByUserID 通过用户ID更新档案
func (s *profileService) UpdateProfileByUserID(ctx context.Context, userID string, req *model.UpdateProfileRequest) (*model.Profile, error) {
	// 先通过userID获取profile
	profile, err := s.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	// 更新字段
	if req.Avatar != nil {
		profile.Avatar = *req.Avatar
	}
	if req.Nickname != nil {
		profile.Nickname = *req.Nickname
	}
	if req.RealName != nil {
		profile.RealName = *req.RealName
	}
	if req.Gender != nil {
		profile.Gender = *req.Gender
	}
	if req.Birthday != nil {
		profile.Birthday = req.Birthday
	}
	if req.Email != nil {
		profile.Email = *req.Email
	}
	if req.Phone != nil {
		profile.Phone = *req.Phone
	}
	if req.Address != nil {
		profile.Address = req.Address
	}
	if req.Social != nil {
		profile.Social = req.Social
	}
	if req.CustomData != nil {
		profile.CustomData = req.CustomData
	}

	// 更新档案
	if err := s.profileRepo.Update(ctx, profile.ID, profile); err != nil {
		return nil, err
	}

	return profile, nil
}
