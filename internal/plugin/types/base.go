package types

import (
	"context"
	"fmt"
	"lauth/internal/model"
	"lauth/pkg/container"
	"sync"

	"github.com/gin-gonic/gin"
)

// SmartPluginBase 智能插件基类
type SmartPluginBase struct {
	mu sync.RWMutex

	// 基础属性
	name         string             // 插件名称
	version      string             // 插件版本
	metadata     *PluginMetadata    // 插件元数据
	stateManager *StateManager      // 状态管理器
	config       interface{}        // 插件配置(泛型)
	hooks        PluginHooks        // 插件钩子
	errorHandler ErrorHandler       // 错误处理器
	validators   []ConfigValidator  // 配置验证器列表
	middleware   []PluginMiddleware // 中间件列表

	// 可选组件
	logger       Logger           // 日志组件
	metrics      MetricsCollector // 指标收集器
	storage      Storage          // 存储接口
	eventEmitter EventEmitter     // 事件发射器

	// API相关
	apiInfo []APIInfo // API信息列表
}

// NewSmartPlugin 创建智能插件实例
func NewSmartPlugin(opts ...Option) *SmartPluginBase {
	p := &SmartPluginBase{
		validators:   make([]ConfigValidator, 0),
		middleware:   make([]PluginMiddleware, 0),
		stateManager: NewStateManager(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Option 插件选项函数
type Option func(*SmartPluginBase)

// WithName 设置插件名称
func WithName(name string) Option {
	return func(p *SmartPluginBase) {
		p.name = name
	}
}

// WithVersion 设置插件版本
func WithVersion(version string) Option {
	return func(p *SmartPluginBase) {
		p.version = version
	}
}

// WithMetadata 设置插件元数据
func WithMetadata(metadata *PluginMetadata) Option {
	return func(p *SmartPluginBase) {
		p.metadata = metadata
	}
}

// WithHooks 设置插件钩子
func WithHooks(hooks PluginHooks) Option {
	return func(p *SmartPluginBase) {
		p.hooks = hooks
	}
}

// WithErrorHandler 设置错误处理器
func WithErrorHandler(handler ErrorHandler) Option {
	return func(p *SmartPluginBase) {
		p.errorHandler = handler
	}
}

// WithLogger 设置日志组件
func WithLogger(logger Logger) Option {
	return func(p *SmartPluginBase) {
		p.logger = logger
	}
}

// WithMetrics 设置指标收集器
func WithMetrics(metrics MetricsCollector) Option {
	return func(p *SmartPluginBase) {
		p.metrics = metrics
	}
}

// WithStorage 设置存储接口
func WithStorage(storage Storage) Option {
	return func(p *SmartPluginBase) {
		p.storage = storage
	}
}

// WithEventEmitter 设置事件发射器
func WithEventEmitter(emitter EventEmitter) Option {
	return func(p *SmartPluginBase) {
		p.eventEmitter = emitter
	}
}

// AddValidator 添加配置验证器
func (p *SmartPluginBase) AddValidator(validator ConfigValidator) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validators = append(p.validators, validator)
}

// AddMiddleware 添加中间件
func (p *SmartPluginBase) AddMiddleware(middleware PluginMiddleware) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.middleware = append(p.middleware, middleware)
}

// Name 获取插件名称
func (p *SmartPluginBase) Name() string {
	return p.name
}

// GetMetadata 获取插件元数据
func (p *SmartPluginBase) GetMetadata() *PluginMetadata {
	return p.metadata
}

// Load 加载插件
func (p *SmartPluginBase) Load(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证状态
	if p.stateManager.GetState() != StateUninitialized {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("plugin %s is already loaded", p.name),
			nil)
	}

	// 验证配置
	for _, validator := range p.validators {
		if err := validator.Validate(config); err != nil {
			p.stateManager.SetError(err)
			return NewPluginError(ErrConfigInvalid,
				"config validation failed",
				err)
		}
	}

	// 保存配置
	p.config = config

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnLoad(config); err != nil {
			p.stateManager.SetError(err)
			return NewPluginError(ErrExecuteFailed,
				"hook OnLoad failed",
				err)
		}
	}

	// 更新状态
	if err := p.stateManager.SetState(StateInitialized); err != nil {
		return err
	}

	return nil
}

// Unload 卸载插件
func (p *SmartPluginBase) Unload() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证状态
	if p.stateManager.GetState() == StateUninitialized {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("plugin %s is not loaded", p.name),
			nil)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnUnload(); err != nil {
			p.stateManager.SetError(err)
			return NewPluginError(ErrExecuteFailed,
				"hook OnUnload failed",
				err)
		}
	}

	// 清理资源
	p.config = nil
	if err := p.stateManager.SetState(StateUninitialized); err != nil {
		return err
	}

	return nil
}

// Execute 执行插件逻辑
func (p *SmartPluginBase) Execute(ctx context.Context, params map[string]interface{}) error {
	// 验证状态
	if p.stateManager.GetState() != StateRunning {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("plugin %s is not running", p.name),
			nil)
	}

	// 创建执行上下文
	execCtx := map[string]interface{}{
		"context": ctx,
		"params":  params,
		"plugin":  p,
	}

	// 按顺序执行中间件的 Before 方法
	for i, middleware := range p.middleware {
		if err := middleware.Before(execCtx); err != nil {
			// 发生错误时,按反序执行已执行中间件的 OnError 方法
			for j := i; j >= 0; j-- {
				if onErr := p.middleware[j].OnError(execCtx, err); onErr != nil {
					p.HandleError(NewPluginError(ErrExecuteFailed,
						"middleware OnError failed",
						onErr))
				}
			}
			return NewPluginError(ErrExecuteFailed,
				"middleware Before failed",
				err)
		}
	}

	// 执行插件逻辑
	var execErr error
	if p.hooks != nil {
		execErr = p.hooks.OnExecute(ctx, params)
	}

	// 调用执行完成钩子
	if p.hooks != nil {
		if err := p.hooks.OnExecuteComplete(ctx, execErr); err != nil {
			p.HandleError(NewPluginError(ErrExecuteFailed,
				"hook OnExecuteComplete failed",
				err))
		}
	}

	// 按反序执行中间件的 After 方法
	for i := len(p.middleware) - 1; i >= 0; i-- {
		if err := p.middleware[i].After(execCtx); err != nil {
			// After 方法的错误只记录不返回
			p.HandleError(NewPluginError(ErrExecuteFailed,
				"middleware After failed",
				err))
		}
	}

	// 如果执行过程中有错误，通过错误处理器处理
	if execErr != nil {
		if p.errorHandler != nil {
			if err := p.errorHandler.HandleError(execErr); err != nil {
				return NewPluginError(ErrExecuteFailed,
					"error handler failed",
					err)
			}
		}
		return NewPluginError(ErrExecuteFailed,
			"plugin execution failed",
			execErr)
	}

	return nil
}

// Start 启动插件
func (p *SmartPluginBase) Start() error {
	// 验证状态
	if p.stateManager.GetState() != StateInitialized {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("plugin %s is not initialized", p.name),
			nil)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnStart(); err != nil {
			p.stateManager.SetError(err)
			return NewPluginError(ErrExecuteFailed,
				"hook OnStart failed",
				err)
		}
	}

	// 更新状态
	if err := p.stateManager.SetState(StateRunning); err != nil {
		return err
	}

	return nil
}

// Stop 停止插件
func (p *SmartPluginBase) Stop() error {
	// 验证状态
	if p.stateManager.GetState() != StateRunning {
		return NewPluginError(ErrInvalidState,
			fmt.Sprintf("plugin %s is not running", p.name),
			nil)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnStop(); err != nil {
			p.stateManager.SetError(err)
			return NewPluginError(ErrExecuteFailed,
				"hook OnStop failed",
				err)
		}
	}

	// 更新状态
	if err := p.stateManager.SetState(StateStopped); err != nil {
		return err
	}

	return nil
}

// GetState 获取插件状态
func (p *SmartPluginBase) GetState() PluginState {
	return p.stateManager.GetState()
}

// GetConfig 获取插件配置
func (p *SmartPluginBase) GetConfig() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if cfg, ok := p.config.(map[string]interface{}); ok {
		return cfg
	}
	return nil
}

// UpdateConfig 更新插件配置
func (p *SmartPluginBase) UpdateConfig(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证配置
	for _, validator := range p.validators {
		if err := validator.Validate(config); err != nil {
			return NewPluginError(ErrConfigInvalid,
				"config validation failed",
				err)
		}
	}

	// 更新配置
	p.config = config
	return nil
}

// ValidateConfig 验证配置
func (p *SmartPluginBase) ValidateConfig(config map[string]interface{}) error {
	for _, validator := range p.validators {
		if err := validator.Validate(config); err != nil {
			return NewPluginError(ErrConfigInvalid,
				"config validation failed",
				err)
		}
	}
	return nil
}

// EmitEvent 发送事件
func (p *SmartPluginBase) EmitEvent(event string, data interface{}) {
	if p.eventEmitter != nil {
		p.eventEmitter.Emit(event, data)
	}
}

// HandleError 处理错误
func (p *SmartPluginBase) HandleError(err error) {
	if p.errorHandler != nil {
		p.errorHandler.HandleError(err)
	}
}

// GetLogger 获取日志组件
func (p *SmartPluginBase) GetLogger() Logger {
	return p.logger
}

// GetHooks 获取插件钩子
func (p *SmartPluginBase) GetHooks() PluginHooks {
	return p.hooks
}

// RegisterRoutes 注册插件路由的默认实现
func (p *SmartPluginBase) RegisterRoutes(group *gin.RouterGroup) {
	// 默认实现为空,子类可以重写此方法
}

// OnInstall 插件安装时的默认实现
func (p *SmartPluginBase) OnInstall(appID string) error {
	// 默认实现为空,子类可以重写此方法
	return nil
}

// OnUninstall 插件卸载时的默认实现
func (p *SmartPluginBase) OnUninstall(appID string) error {
	// 默认实现为空,子类可以重写此方法
	return nil
}

// GetAPIInfo 获取插件API信息的默认实现
func (p *SmartPluginBase) GetAPIInfo() []APIInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.apiInfo
}

// AddAPIInfo 添加API信息
func (p *SmartPluginBase) AddAPIInfo(info APIInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.apiInfo = append(p.apiInfo, info)
}

// GetDependencies 获取插件依赖的服务
func (p *SmartPluginBase) GetDependencies() []string {
	// 默认实现为空,子类可以重写此方法
	return nil
}

// Configure 配置插件（注入依赖）
func (p *SmartPluginBase) Configure(container container.PluginContainer) error {
	// 默认实现为空,子类可以重写此方法
	return nil
}

// NeedsVerification 判断是否需要验证
func (p *SmartPluginBase) NeedsVerification(ctx context.Context, userID string, action string, context map[string]interface{}) (bool, error) {
	// 默认实现为不需要验证
	return false, nil
}

// ValidateVerification 验证当前验证是否有效
func (p *SmartPluginBase) ValidateVerification(ctx context.Context, userID string, action string, verificationID string) (bool, error) {
	// 默认实现为验证无效
	return false, nil
}

// OnVerificationSuccess 验证成功时的回调
func (p *SmartPluginBase) OnVerificationSuccess(ctx context.Context, userID string, action string, context map[string]interface{}) error {
	// 默认实现为空
	return nil
}

// GetLastVerification 获取上次验证信息
func (p *SmartPluginBase) GetLastVerification(ctx context.Context, userID string, action string) (*model.PluginStatus, error) {
	// 默认实现为无验证信息
	return nil, nil
}
