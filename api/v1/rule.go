package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"
)

// RuleHandler 规则处理器
type RuleHandler struct {
	ruleService service.RuleService
}

// NewRuleHandler 创建规则处理器实例
func NewRuleHandler(ruleService service.RuleService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
	}
}

// Register 注册路由
func (h *RuleHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	// 将rules路由注册为apps的子路由，统一使用:id参数
	rules := r.Group("/apps/:id/rules", authMiddleware.HandleAuth())
	{
		rules.POST("", h.Create)
		rules.GET("/:rule_id", h.Get)
		rules.PUT("/:rule_id", h.Update)
		rules.DELETE("/:rule_id", h.Delete)
		rules.GET("", h.List)
		rules.GET("/active", h.ListActive)
		rules.POST("/validate", h.ValidateRule)

		// 规则条件管理
		ruleConditions := rules.Group("/:rule_id/conditions")
		{
			ruleConditions.POST("", h.AddConditions)
			ruleConditions.PUT("", h.UpdateConditions)
			ruleConditions.DELETE("", h.RemoveConditions)
			ruleConditions.GET("", h.GetConditions)
		}
	}
}

// RuleCreateRequest 创建规则请求
type RuleCreateRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Type        model.RuleType `json:"type" binding:"required"`
	Priority    int            `json:"priority"`
	IsEnabled   bool           `json:"is_enabled"`
}

// Create 创建规则
func (h *RuleHandler) Create(c *gin.Context) {
	var req RuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appID := c.Param("id")
	rule := &model.Rule{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		IsEnabled:   req.IsEnabled,
	}

	if err := h.ruleService.Create(c.Request.Context(), appID, rule); err != nil {
		switch err {
		case service.ErrRuleNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// Get 获取规则
func (h *RuleHandler) Get(c *gin.Context) {
	id := c.Param("rule_id")
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// RuleUpdateRequest 更新规则请求
type RuleUpdateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	IsEnabled   bool   `json:"is_enabled"`
}

// Update 更新规则
func (h *RuleHandler) Update(c *gin.Context) {
	var req RuleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("rule_id")
	rule, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rule.Name = req.Name
	rule.Description = req.Description
	rule.Priority = req.Priority
	rule.IsEnabled = req.IsEnabled

	if err := h.ruleService.Update(c.Request.Context(), rule); err != nil {
		switch err {
		case service.ErrRuleNameExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, rule)
}

// Delete 删除规则
func (h *RuleHandler) Delete(c *gin.Context) {
	id := c.Param("rule_id")
	if err := h.ruleService.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// List 获取规则列表
func (h *RuleHandler) List(c *gin.Context) {
	appID := c.Param("id")
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	rules, total, err := h.ruleService.List(c.Request.Context(), appID, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": rules,
		"total": total,
	})
}

// ListActive 获取启用的规则列表
func (h *RuleHandler) ListActive(c *gin.Context) {
	appID := c.Param("id")
	rules, err := h.ruleService.GetActiveRules(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// ConditionRequest 规则条件请求
type ConditionRequest struct {
	Conditions []model.RuleCondition `json:"conditions" binding:"required"`
}

// AddConditions 添加规则条件
func (h *RuleHandler) AddConditions(c *gin.Context) {
	var req ConditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ruleID := c.Param("rule_id")
	if err := h.ruleService.AddConditions(c.Request.Context(), ruleID, req.Conditions); err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateConditions 更新规则条件
func (h *RuleHandler) UpdateConditions(c *gin.Context) {
	var req ConditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ruleID := c.Param("rule_id")
	if err := h.ruleService.UpdateConditions(c.Request.Context(), ruleID, req.Conditions); err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveConditions 删除规则条件
func (h *RuleHandler) RemoveConditions(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if err := h.ruleService.RemoveConditions(c.Request.Context(), ruleID); err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetConditions 获取规则条件
func (h *RuleHandler) GetConditions(c *gin.Context) {
	ruleID := c.Param("rule_id")
	conditions, err := h.ruleService.GetConditions(c.Request.Context(), ruleID)
	if err != nil {
		if err == service.ErrRuleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conditions)
}

// ValidateRuleRequest 验证规则请求
type ValidateRuleRequest struct {
	Data map[string]interface{} `json:"data" binding:"required"`
}

// ValidateRule 验证规则
func (h *RuleHandler) ValidateRule(c *gin.Context) {
	var req ValidateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appID := c.Param("id")
	result, err := h.ruleService.ValidateRule(c.Request.Context(), appID, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
