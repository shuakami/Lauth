package service

import (
	"context"
	"errors"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/pkg/engine"
)

// ValidateTokenAndRuleResponse 组合验证响应
type ValidateTokenAndRuleResponse struct {
	User         *model.User    `json:"user"`
	RuleResult   *engine.Result `json:"rule_result"`
	ValidateTime time.Time      `json:"validate_time"`
	Status       bool           `json:"status"`
}

// authValidationService 验证服务
type authValidationService struct {
	userRepo     repository.UserRepository
	tokenService TokenService
	ruleService  RuleService
}

// newAuthValidationService 创建验证服务实例
func newAuthValidationService(
	userRepo repository.UserRepository,
	tokenService TokenService,
	ruleService RuleService,
) *authValidationService {
	return &authValidationService{
		userRepo:     userRepo,
		tokenService: tokenService,
		ruleService:  ruleService,
	}
}

// ValidateTokenAndGetUser 验证Token并获取用户信息（快速接口）
func (s *authValidationService) ValidateTokenAndGetUser(ctx context.Context, token string) (*model.TokenUserInfo, error) {
	// 验证Token
	claims, err := s.tokenService.ValidateToken(ctx, token, model.AccessToken)
	if err != nil {
		return nil, err
	}

	// 构造快速响应
	return &model.TokenUserInfo{
		UserID:   claims.UserID,
		AppID:    claims.AppID,
		Username: claims.Username,
	}, nil
}

// ValidateTokenAndRuleWithUser 组合验证令牌和规则并返回用户信息
func (s *authValidationService) ValidateTokenAndRuleWithUser(ctx context.Context, token string, data map[string]interface{}) (*ValidateTokenAndRuleResponse, error) {
	// 先验证令牌并获取用户信息
	userInfo, err := s.ValidateTokenAndGetUser(ctx, token)
	if err != nil {
		return nil, err
	}

	// 将 token 中的用户信息添加到验证数据中
	data["token_user_id"] = userInfo.UserID
	data["token_app_id"] = userInfo.AppID
	data["token_username"] = userInfo.Username

	// 如果请求中包含 user_id，验证是否与 token 用户匹配
	if requestUserID, ok := data["user_id"].(string); ok {
		if requestUserID != userInfo.UserID {
			return nil, errors.New("user_id mismatch with token")
		}
	}

	// 验证规则
	ruleResult, ruleErr := s.ruleService.ValidateRule(ctx, userInfo.AppID, data)

	// 获取完整的用户信息
	user, err := s.userRepo.GetByID(ctx, userInfo.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 构造响应
	response := &ValidateTokenAndRuleResponse{
		User:         user,
		RuleResult:   ruleResult,
		ValidateTime: time.Now(),
		Status:       ruleErr == nil,
	}

	return response, ruleErr
}
