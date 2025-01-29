package config

import (
	"fmt"

	"lauth/pkg/database"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Database database.Config `yaml:"database"`
	MongoDB  MongoDBConfig   `yaml:"mongodb"`
	Redis    RedisConfig     `yaml:"redis"`
	JWT      JWTConfig       `yaml:"jwt"`
	OIDC     OIDCConfig      `yaml:"oidc"`
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

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
