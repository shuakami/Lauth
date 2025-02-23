package boot

import (
	"lauth/pkg/config"
)

// InitConfig 初始化配置
func InitConfig(path string) (*config.Config, error) {
	return config.LoadConfig(path)
}
