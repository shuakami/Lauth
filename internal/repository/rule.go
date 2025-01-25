package repository

import (
	"context"
	"errors"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// RuleRepository 规则仓储接口
type RuleRepository interface {
	// 基础CRUD
	Create(ctx context.Context, rule *model.Rule) error
	GetByID(ctx context.Context, id string) (*model.Rule, error)
	Update(ctx context.Context, rule *model.Rule) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByName(ctx context.Context, appID, name string) (*model.Rule, error)
	List(ctx context.Context, appID string, offset, limit int) ([]model.Rule, int64, error)
	ListByType(ctx context.Context, appID string, ruleType model.RuleType) ([]model.Rule, error)

	// 规则条件管理
	AddConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error
	UpdateConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error
	RemoveConditions(ctx context.Context, ruleID string) error
	GetConditions(ctx context.Context, ruleID string) ([]model.RuleCondition, error)

	// 规则验证
	GetActiveRules(ctx context.Context, appID string) ([]model.Rule, error)
}

// ruleRepository 规则仓储实现
type ruleRepository struct {
	db *gorm.DB
}

// NewRuleRepository 创建规则仓储实例
func NewRuleRepository(db *gorm.DB) RuleRepository {
	return &ruleRepository{db: db}
}

// Create 创建规则
func (r *ruleRepository) Create(ctx context.Context, rule *model.Rule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 通过ID获取规则
func (r *ruleRepository) GetByID(ctx context.Context, id string) (*model.Rule, error) {
	var rule model.Rule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// GetByName 通过名称获取规则
func (r *ruleRepository) GetByName(ctx context.Context, appID, name string) (*model.Rule, error) {
	var rule model.Rule
	if err := r.db.WithContext(ctx).Where("app_id = ? AND name = ?", appID, name).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// Update 更新规则
func (r *ruleRepository) Update(ctx context.Context, rule *model.Rule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 删除规则
func (r *ruleRepository) Delete(ctx context.Context, id string) error {
	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除规则的条件
		if err := tx.Where("rule_id = ?", id).Delete(&model.RuleCondition{}).Error; err != nil {
			return err
		}

		// 删除规则
		return tx.Delete(&model.Rule{}, "id = ?", id).Error
	})
}

// List 获取规则列表
func (r *ruleRepository) List(ctx context.Context, appID string, offset, limit int) ([]model.Rule, int64, error) {
	var rules []model.Rule
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Rule{}).Where("app_id = ?", appID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("app_id = ?", appID).Offset(offset).Limit(limit).Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListByType 获取指定类型的规则列表
func (r *ruleRepository) ListByType(ctx context.Context, appID string, ruleType model.RuleType) ([]model.Rule, error) {
	var rules []model.Rule
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND type = ?", appID, ruleType).
		Find(&rules).Error
	return rules, err
}

// AddConditions 为规则添加条件
func (r *ruleRepository) AddConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error {
	for i := range conditions {
		conditions[i].RuleID = ruleID
	}
	return r.db.WithContext(ctx).Create(&conditions).Error
}

// UpdateConditions 更新规则的条件
func (r *ruleRepository) UpdateConditions(ctx context.Context, ruleID string, conditions []model.RuleCondition) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除现有条件
		if err := tx.Where("rule_id = ?", ruleID).Delete(&model.RuleCondition{}).Error; err != nil {
			return err
		}

		// 添加新条件
		for i := range conditions {
			conditions[i].RuleID = ruleID
		}
		return tx.Create(&conditions).Error
	})
}

// RemoveConditions 移除规则的所有条件
func (r *ruleRepository) RemoveConditions(ctx context.Context, ruleID string) error {
	return r.db.WithContext(ctx).Where("rule_id = ?", ruleID).Delete(&model.RuleCondition{}).Error
}

// GetConditions 获取规则的所有条件
func (r *ruleRepository) GetConditions(ctx context.Context, ruleID string) ([]model.RuleCondition, error) {
	var conditions []model.RuleCondition
	err := r.db.WithContext(ctx).Where("rule_id = ?", ruleID).Find(&conditions).Error
	return conditions, err
}

// GetActiveRules 获取所有启用的规则
func (r *ruleRepository) GetActiveRules(ctx context.Context, appID string) ([]model.Rule, error) {
	var rules []model.Rule
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND is_enabled = ?", appID, true).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}
