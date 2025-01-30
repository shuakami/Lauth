package config

import (
	"fmt"

	"lauth/pkg/database"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig    `mapstructure:"server"`
	Database database.Config `mapstructure:"database"`
	MongoDB  MongoDBConfig   `mapstructure:"mongodb"`
	Redis    RedisConfig     `mapstructure:"redis"`
	JWT      JWTConfig       `mapstructure:"jwt"`
	OIDC     OIDCConfig      `mapstructure:"oidc"`
	Audit    AuditConfig     `mapstructure:"audit"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        int
	Mode        string
	AuthEnabled bool `mapstructure:"auth_enabled"` // 是否启用认证
}

// MongoDBConfig MongoDB配置
type MongoDBConfig struct {
	URI         string `mapstructure:"uri"`
	Database    string `mapstructure:"database"`
	MaxPoolSize uint64 `mapstructure:"max_pool_size"`
	MinPoolSize uint64 `mapstructure:"min_pool_size"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret             string
	AccessTokenExpire  int `mapstructure:"access_token_expire"`
	RefreshTokenExpire int `mapstructure:"refresh_token_expire"`
}

// OIDCConfig OIDC配置
type OIDCConfig struct {
	Issuer         string `mapstructure:"issuer"`           // OIDC颁发者标识符
	PrivateKeyPath string `mapstructure:"private_key_path"` // RSA私钥路径
	PublicKeyPath  string `mapstructure:"public_key_path"`  // RSA公钥路径
}

// AuditConfig 审计配置
type AuditConfig struct {
	LogDir        string          `mapstructure:"log_dir"`        // 日志目录
	RotationSize  int64           `mapstructure:"rotation_size"`  // 日志文件轮转大小
	RetentionDays int             `mapstructure:"retention_days"` // 日志保留天数
	WebSocket     WebSocketConfig `mapstructure:"websocket"`      // WebSocket配置
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	PingInterval   int `mapstructure:"ping_interval"`    // 心跳间隔(秒)
	WriteWait      int `mapstructure:"write_wait"`       // 写超时(秒)
	ReadWait       int `mapstructure:"read_wait"`        // 读超时(秒)
	MaxMessageSize int `mapstructure:"max_message_size"` // 最大消息大小(字节)
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml") // 设置配置文件类型
	viper.AutomaticEnv()        // 读取环境变量

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
