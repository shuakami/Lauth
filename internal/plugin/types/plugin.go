package types

import (
	"context"
	"lauth/internal/model"
	"lauth/pkg/container"

	"github.com/gin-gonic/gin"
)

// Plugin 验证插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// GetMetadata 返回插件元数据
	GetMetadata() *PluginMetadata

	// Load 加载插件
	Load(config map[string]interface{}) error

	// Unload 卸载插件
	Unload() error

	// Execute 执行插件逻辑
	Execute(ctx context.Context, params map[string]interface{}) error

	// NeedsVerification 判断是否需要验证
	// userID: 用户ID
	// action: 动作类型（如：login, register）
	// context: 上下文信息（如：IP, 设备信息等）
	NeedsVerification(ctx context.Context, userID string, action string, context map[string]interface{}) (bool, error)

	// GetLastVerification 获取上次验证信息
	// userID: 用户ID
	// action: 动作类型
	GetLastVerification(ctx context.Context, userID string, action string) (*model.PluginStatus, error)

	// ValidateVerification 验证当前验证是否有效
	// userID: 用户ID
	// action: 动作类型
	// verificationID: 验证ID
	ValidateVerification(ctx context.Context, userID string, action string, verificationID string) (bool, error)

	// GetUserConfig 获取用户配置
	GetUserConfig(ctx context.Context, userID string) (map[string]interface{}, error)

	// UpdateUserConfig 更新用户配置
	UpdateUserConfig(ctx context.Context, userID string, config map[string]interface{}) error

	// OnVerificationSuccess 验证成功时的回调
	// 用于更新用户配置（如添加可信IP、设备等）
	OnVerificationSuccess(ctx context.Context, userID string, action string, context map[string]interface{}) error

	// Start 启动插件
	Start() error

	// Stop 停止插件
	Stop() error

	// GetDependencies 获取插件依赖的服务
	// 返回服务名称列表
	GetDependencies() []string

	// Configure 配置插件（注入依赖）
	// container: 依赖注入容器
	Configure(container container.PluginContainer) error
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
