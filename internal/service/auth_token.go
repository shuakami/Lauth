package service

import (
	"context"
	"fmt"

	"lauth/internal/model"
	"lauth/internal/repository"
)

// authTokenService 令牌管理服务
type authTokenService struct {
	userRepo     repository.UserRepository
	tokenService TokenService
}

// newAuthTokenService 创建令牌管理服务实例
func newAuthTokenService(
	userRepo repository.UserRepository,
	tokenService TokenService,
) *authTokenService {
	return &authTokenService{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

// RefreshToken 刷新访问令牌
func (s *authTokenService) RefreshToken(ctx context.Context, refreshToken string) (*model.ExtendedLoginResponse, error) {
	// 使用TokenService刷新令牌
	tokenPair, err := s.tokenService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	claims, err := s.tokenService.ValidateToken(ctx, tokenPair.RefreshToken, model.RefreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if user.Status == model.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	// 构造响应
	return &model.ExtendedLoginResponse{
		User: model.UserResponse{
			ID:        user.ID,
			AppID:     user.AppID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Phone:     user.Phone,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(tokenPair.AccessTokenExpireIn.Seconds()),
		AuthStatus:   model.PluginStatusCompleted,
	}, nil
}

// Logout 用户登出
func (s *authTokenService) Logout(ctx context.Context, accessToken string) error {
	// 验证并吊销访问令牌
	claims, err := s.tokenService.ValidateToken(ctx, accessToken, model.AccessToken)
	if err != nil {
		return err
	}

	// 吊销访问令牌
	if err := s.tokenService.RevokeToken(ctx, accessToken, model.AccessToken); err != nil {
		return err
	}

	// 获取并吊销关联的刷新令牌
	refreshKey := fmt.Sprintf("refresh_token:%s", claims.UserID)
	refreshToken, err := s.tokenService.(*tokenService).redis.Get(ctx, refreshKey)
	if err == nil && refreshToken != "" {
		_ = s.tokenService.RevokeToken(ctx, refreshToken, model.RefreshToken)
	}

	return nil
}
