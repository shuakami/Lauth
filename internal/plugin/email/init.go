package email

import (
	"lauth/internal/plugin/types"
)

// 注册插件工厂函数
func init() {
	// 注册Email插件
	types.RegisterPlugin("email_verify", func() types.Plugin {
		return NewEmailPlugin()
	})
}
