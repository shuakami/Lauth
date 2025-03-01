package totp

import (
	"lauth/internal/plugin/types"
)

// 注册插件工厂函数
func init() {
	// 注册TOTP插件
	types.RegisterPlugin("totp", func() types.Plugin {
		return NewTOTPPlugin()
	})
}
