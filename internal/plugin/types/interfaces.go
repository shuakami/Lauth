package types

import (
	"context"
	"fmt"
	"lauth/internal/model"
	"lauth/pkg/container"

	"github.com/gin-gonic/gin"
)

// PluginError 插件错误
type PluginError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ErrorCode 错误码
type ErrorCode string

const (
	// ErrInvalidState 状态错误
	ErrInvalidState ErrorCode = "INVALID_STATE"
	// ErrConfigInvalid 配置无效
	ErrConfigInvalid ErrorCode = "CONFIG_INVALID"
	// ErrDependencyMissing 依赖缺失
	ErrDependencyMissing ErrorCode = "DEPENDENCY_MISSING"
	// ErrExecuteFailed 执行失败
	ErrExecuteFailed ErrorCode = "EXECUTE_FAILED"
	// ErrVerificationFailed 验证失败
	ErrVerificationFailed ErrorCode = "VERIFICATION_FAILED"
)

// NewPluginError 创建插件错误
func NewPluginError(code ErrorCode, message string, cause error) error {
	return &PluginError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// OperationMetadata 操作元数据
type OperationMetadata struct {
	Name        string            `json:"name"`        // 操作名称
	Description string            `json:"description"` // 操作描述
	Parameters  map[string]string `json:"parameters"`  // 参数说明
	Returns     map[string]string `json:"returns"`     // 返回值说明
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name        string              `json:"name"`        // 插件名称
	Description string              `json:"description"` // 插件描述
	Version     string              `json:"version"`     // 插件版本
	Author      string              `json:"author"`      // 插件作者
	Required    bool                `json:"required"`    // 是否默认必需
	Stage       string              `json:"stage"`       // 默认执行阶段
	Actions     []string            `json:"actions"`     // 支持的业务场景
	Operations  []OperationMetadata `json:"operations"`  // 支持的操作类型
}

// Plugin 定义了插件的基本接口
// 每个插件都必须实现这个接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// GetMetadata 返回插件元数据
	GetMetadata() *PluginMetadata

	// Load 加载插件
	// config: 插件配置
	Load(config map[string]interface{}) error

	// Unload 卸载插件
	Unload() error

	// Start 启动插件
	Start() error

	// Stop 停止插件
	Stop() error

	// NeedsVerificationSession 判断指定操作是否需要验证会话
	// operation: 操作类型
	// 返回 true 表示需要验证会话，false 表示不需要
	NeedsVerificationSession(operation string) bool
}

// Executable 定义了插件执行接口
// 如果插件需要执行业务逻辑,可以实现这个接口
type Executable interface {
	// Execute 执行插件逻辑
	Execute(ctx context.Context, params map[string]interface{}) error
}

// Configurable 定义了插件配置管理接口
// 如果插件需要管理配置,可以实现这个接口
type Configurable interface {
	// GetConfig 获取插件配置
	GetConfig() map[string]interface{}

	// UpdateConfig 更新插件配置
	UpdateConfig(config map[string]interface{}) error

	// ValidateConfig 验证配置是否有效
	ValidateConfig(config map[string]interface{}) error
}

// Verifiable 定义了插件验证逻辑接口
// 如果插件需要进行验证操作,可以实现这个接口
type Verifiable interface {
	// NeedsVerification 判断是否需要验证
	// userID: 用户ID
	// action: 动作类型（如：login, register）
	// context: 上下文信息（如：IP, 设备信息等）
	NeedsVerification(ctx context.Context, userID string, action string, context map[string]interface{}) (bool, error)

	// ValidateVerification 验证当前验证是否有效
	// userID: 用户ID
	// action: 动作类型
	// verificationID: 验证ID
	ValidateVerification(ctx context.Context, userID string, action string, verificationID string) (bool, error)

	// OnVerificationSuccess 验证成功时的回调
	// 用于更新用户配置（如添加可信IP、设备等）
	OnVerificationSuccess(ctx context.Context, userID string, action string, context map[string]interface{}) error

	// GetLastVerification 获取上次验证信息
	// userID: 用户ID
	// action: 动作类型
	GetLastVerification(ctx context.Context, userID string, action string) (*model.PluginStatus, error)
}

// Routable 定义了插件路由注册接口
// 如果插件需要提供HTTP接口,可以实现这个接口
type Routable interface {
	// RegisterRoutes 注册插件路由
	// group: 插件路由组(前缀为/api/v1/plugins/{appID}/{pluginName})
	RegisterRoutes(group *gin.RouterGroup)

	// GetAPIInfo 获取插件API信息
	// 返回插件注册的所有API接口信息
	GetAPIInfo() []APIInfo

	// GetRoutesRequireAuth 获取需要认证的路由列表
	// 返回插件中需要认证的路由路径(相对于插件路由组的路径)
	// 例如: []string{"/setup", "/disable"} 表示 /setup 和 /disable 路由需要认证
	// 如果返回nil或空切片，表示所有路由都不需要认证
	// 如果返回包含"*"的切片，表示所有路由都需要认证
	GetRoutesRequireAuth() []string
}

// Injectable 定义了插件依赖注入接口
// 如果插件需要依赖其他服务,可以实现这个接口
type Injectable interface {
	// GetDependencies 获取插件依赖的服务
	// 返回服务名称列表
	GetDependencies() []string

	// Configure 配置插件（注入依赖）
	// container: 依赖注入容器
	Configure(container container.PluginContainer) error
}

// Installable 定义了插件安装/卸载接口
// 如果插件需要在安装/卸载时进行特殊处理,可以实现这个接口
type Installable interface {
	// OnInstall 插件安装时的回调
	// 用于进行必要的初始化工作(如创建数据表等)
	OnInstall(appID string) error

	// OnUninstall 插件卸载时的回调
	// 用于进行清理工作(如删除数据表等)
	OnUninstall(appID string) error
}

// SmartPlugin 定义了智能插件接口
// 智能插件是一个可选功能的集合
// 插件可以根据需要实现相应的接口
type SmartPlugin interface {
	Plugin       // 基础插件能力(必需)
	Executable   // 执行能力(可选)
	Configurable // 配置管理能力(可选)
	Verifiable   // 验证能力(可选)
	Routable     // 路由能力(可选)
	Injectable   // 依赖注入能力(可选)
	Installable  // 安装能力(可选)
}

// Manager 插件管理器接口
type Manager interface {
	// LoadPlugin 加载插件
	LoadPlugin(appID string, p Plugin, config map[string]interface{}) error

	// UnloadPlugin 卸载插件
	UnloadPlugin(appID string, name string) error

	// GetPlugin 获取插件
	// 如果插件实现了SmartPlugin接口,返回SmartPlugin实例
	GetPlugin(appID string, name string) (Plugin, bool)

	// GetSmartPlugin 获取SmartPlugin实例
	// 如果插件没有实现SmartPlugin接口,返回false
	GetSmartPlugin(appID string, name string) (SmartPlugin, bool)

	// ExecutePlugin 执行指定插件
	ExecutePlugin(ctx context.Context, appID string, name string, params map[string]interface{}) error

	// ListPlugins 列出App的所有插件
	ListPlugins(appID string) []string

	// InitPlugins 初始化插件（从数据库加载插件配置）
	InitPlugins(ctx context.Context) error

	// GetPluginConfigs 获取应用的所有插件配置
	GetPluginConfigs(ctx context.Context, appID string) ([]*model.PluginConfig, error)

	// SavePluginConfig 保存插件配置
	SavePluginConfig(ctx context.Context, config *model.PluginConfig) error

	// RegisterPluginRoutes 注册插件路由
	// routerGroup: 路由组(前缀为/api/v1/plugins/{appID})
	RegisterPluginRoutes(appID string, routerGroup *gin.RouterGroup) error

	// InstallPlugin 安装插件
	// 会调用插件的OnInstall方法
	InstallPlugin(ctx context.Context, appID string, name string, config map[string]interface{}) error

	// UninstallPlugin 卸载插件
	// 会调用插件的OnUninstall方法
	UninstallPlugin(ctx context.Context, appID string, name string) error
}

// AuthMiddleware 认证中间件接口
type AuthMiddleware interface {
	// HandleAuth 处理认证
	HandleAuth() gin.HandlerFunc
}
