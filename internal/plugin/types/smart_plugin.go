package types

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// PluginState 插件状态
type PluginState int32

const (
	// StateUninitialized 未初始化
	StateUninitialized PluginState = iota
	// StateInitialized 已初始化
	StateInitialized
	// StateRunning 运行中
	StateRunning
	// StateStopped 已停止
	StateStopped
	// StateError 错误状态
	StateError
)

// SmartPlugin 智能插件接口
// 扩展自基础Plugin接口,提供更灵活的扩展机制
type SmartPlugin interface {
	// 继承基础Plugin接口
	Plugin

	// RegisterRoutes 注册插件路由
	// group: 插件路由组(前缀为/api/v1/plugins/{appID}/{pluginName})
	RegisterRoutes(group *gin.RouterGroup)

	// OnInstall 插件安装时的回调
	// 用于进行必要的初始化工作(如创建数据表等)
	OnInstall(appID string) error

	// OnUninstall 插件卸载时的回调
	// 用于进行清理工作(如删除数据表等)
	OnUninstall(appID string) error

	// GetAPIInfo 获取插件API信息
	// 返回插件注册的所有API接口信息
	GetAPIInfo() []APIInfo
}

// APIInfo API接口信息
type APIInfo struct {
	Method      string            `json:"method"`      // HTTP方法
	Path        string            `json:"path"`        // 路径
	Description string            `json:"description"` // 接口描述
	Parameters  map[string]string `json:"parameters"`  // 参数说明
	Returns     map[string]string `json:"returns"`     // 返回值说明
}

// SmartPluginBase 智能插件基类
type SmartPluginBase struct {
	mu sync.RWMutex

	// 基础属性
	name         string             // 插件名称
	version      string             // 插件版本
	metadata     *PluginMetadata    // 插件元数据
	state        atomic.Int32       // 插件状态
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
		validators: make([]ConfigValidator, 0),
		middleware: make([]PluginMiddleware, 0),
	}
	p.state.Store(int32(StateUninitialized))

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
	if p.state.Load() != int32(StateUninitialized) {
		return fmt.Errorf("plugin %s is already loaded", p.name)
	}

	// 验证配置
	for _, validator := range p.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("config validation failed: %v", err)
		}
	}

	// 保存配置
	p.config = config

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnLoad(config); err != nil {
			return fmt.Errorf("hook OnLoad failed: %v", err)
		}
	}

	// 更新状态
	p.state.Store(int32(StateInitialized))
	return nil
}

// Unload 卸载插件
func (p *SmartPluginBase) Unload() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证状态
	if p.state.Load() == int32(StateUninitialized) {
		return fmt.Errorf("plugin %s is not loaded", p.name)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnUnload(); err != nil {
			return fmt.Errorf("hook OnUnload failed: %v", err)
		}
	}

	// 清理资源
	p.config = nil
	p.state.Store(int32(StateUninitialized))
	return nil
}

// Execute 执行插件逻辑
func (p *SmartPluginBase) Execute(ctx context.Context, params map[string]interface{}) error {
	// 验证状态
	if p.state.Load() != int32(StateRunning) {
		return fmt.Errorf("plugin %s is not running", p.name)
	}

	// 构建执行链
	handler := p.hooks.OnExecute
	for i := len(p.middleware) - 1; i >= 0; i-- {
		handler = p.middleware[i](handler)
	}

	// 执行
	return handler(ctx, params)
}

// Start 启动插件
func (p *SmartPluginBase) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证状态
	if p.state.Load() != int32(StateInitialized) {
		return fmt.Errorf("plugin %s is not initialized", p.name)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnStart(); err != nil {
			return fmt.Errorf("hook OnStart failed: %v", err)
		}
	}

	// 更新状态
	p.state.Store(int32(StateRunning))
	return nil
}

// Stop 停止插件
func (p *SmartPluginBase) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证状态
	if p.state.Load() != int32(StateRunning) {
		return fmt.Errorf("plugin %s is not running", p.name)
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnStop(); err != nil {
			return fmt.Errorf("hook OnStop failed: %v", err)
		}
	}

	// 更新状态
	p.state.Store(int32(StateStopped))
	return nil
}

// GetState 获取插件状态
func (p *SmartPluginBase) GetState() PluginState {
	return PluginState(p.state.Load())
}

// GetConfig 获取插件配置
func (p *SmartPluginBase) GetConfig() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// UpdateConfig 更新插件配置
func (p *SmartPluginBase) UpdateConfig(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 验证配置
	for _, validator := range p.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("config validation failed: %v", err)
		}
	}

	// 调用钩子
	if p.hooks != nil {
		if err := p.hooks.OnConfigUpdate(config); err != nil {
			return fmt.Errorf("hook OnConfigUpdate failed: %v", err)
		}
	}

	// 更新配置
	p.config = config
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
		p.errorHandler.Handle(err)
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
