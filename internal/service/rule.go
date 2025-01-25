package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/pkg/engine"
)

var (
	ErrRuleNotFound   = errors.New("规则不存在")
	ErrRuleNameExists = errors.New("规则名称已存在")
)

// RuleService 规则服务接口
type RuleService interface {
	// 基础CRUD
	Create(ctx context.Context, appID string, rule *model.Rule) error
	GetByID(ctx context.Context, id string) (*model.Rule, error)
	Update(ctx context.Context, rule *model.Rule) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, appID string, offset, limit int) ([]model.Rule, int64, error)

	// 规则条件管理
	AddConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error
	UpdateConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error
	RemoveConditions(ctx context.Context, ruleID string) error
	GetConditions(ctx context.Context, ruleID string) ([]model.RuleCondition, error)

	// 规则验证和执行
	ValidateRule(ctx context.Context, appID string, data map[string]interface{}) (*engine.Result, error)
	GetActiveRules(ctx context.Context, appID string) ([]model.Rule, error)
}

// ruleService 规则服务实现
type ruleService struct {
	ruleRepo repository.RuleRepository
	engine   engine.Engine
}

// NewRuleService 创建规则服务实例
func NewRuleService(ruleRepo repository.RuleRepository, engine engine.Engine) RuleService {
	return &ruleService{
		ruleRepo: ruleRepo,
		engine:   engine,
	}
}

// Create 创建规则
func (s *ruleService) Create(ctx context.Context, appID string, rule *model.Rule) error {
	// 检查规则名是否已存在
	existing, err := s.ruleRepo.GetByName(ctx, appID, rule.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrRuleNameExists
	}

	rule.AppID = appID
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, appID)
}

// GetByID 获取规则
func (s *ruleService) GetByID(ctx context.Context, id string) (*model.Rule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrRuleNotFound
	}
	return rule, nil
}

// Update 更新规则
func (s *ruleService) Update(ctx context.Context, rule *model.Rule) error {
	existing, err := s.ruleRepo.GetByID(ctx, rule.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRuleNotFound
	}

	// 检查新名称是否与其他规则冲突
	if rule.Name != existing.Name {
		nameExists, err := s.ruleRepo.GetByName(ctx, rule.AppID, rule.Name)
		if err != nil {
			return err
		}
		if nameExists != nil && nameExists.ID != rule.ID {
			return ErrRuleNameExists
		}
	}

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, rule.AppID)
}

// Delete 删除规则
func (s *ruleService) Delete(ctx context.Context, id string) error {
	existing, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRuleNotFound
	}

	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, existing.AppID)
}

// List 获取规则列表
func (s *ruleService) List(ctx context.Context, appID string, offset, limit int) ([]model.Rule, int64, error) {
	return s.ruleRepo.List(ctx, appID, offset, limit)
}

// AddConditions 为规则添加条件
func (s *ruleService) AddConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error {
	// 检查规则是否存在
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return ErrRuleNotFound
	}

	if err := s.ruleRepo.AddConditions(ctx, ruleID, conditions); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, rule.AppID)
}

// UpdateConditions 更新规则的条件
func (s *ruleService) UpdateConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error {
	// 检查规则是否存在
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return ErrRuleNotFound
	}

	if err := s.ruleRepo.UpdateConditions(ctx, ruleID, conditions); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, rule.AppID)
}

// RemoveConditions 移除规则的所有条件
func (s *ruleService) RemoveConditions(ctx context.Context, ruleID string) error {
	// 检查规则是否存在
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return ErrRuleNotFound
	}

	if err := s.ruleRepo.RemoveConditions(ctx, ruleID); err != nil {
		return err
	}

	// 使规则缓存失效
	return s.engine.InvalidateCache(ctx, rule.AppID)
}

// GetConditions 获取规则的所有条件
func (s *ruleService) GetConditions(ctx context.Context, ruleID string) ([]model.RuleCondition, error) {
	// 检查规则是否存在
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, ErrRuleNotFound
	}

	return s.ruleRepo.GetConditions(ctx, ruleID)
}

// ValidateRule 验证规则
func (s *ruleService) ValidateRule(ctx context.Context, appID string, data map[string]interface{}) (*engine.Result, error) {
	return s.engine.Evaluate(ctx, appID, data)
}

// GetActiveRules 获取所有启用的规则
func (s *ruleService) GetActiveRules(ctx context.Context, appID string) ([]model.Rule, error) {
	return s.ruleRepo.GetActiveRules(ctx, appID)
}
