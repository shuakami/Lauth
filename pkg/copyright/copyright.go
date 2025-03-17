package copyright

import (
	"fmt"
	"strings"

	"lauth/pkg/version"

	"github.com/fatih/color"
)

var (
	// 颜色组合
	titleColor   = color.New(color.FgHiCyan, color.Bold)
	versionColor = color.New(color.FgHiGreen)
	successColor = color.New(color.FgGreen)
	warningColor = color.New(color.FgYellow)
	pluginColor  = color.New(color.FgHiMagenta)
	defaultColor = color.New(color.FgWhite)
	numberColor  = color.New(color.FgHiYellow)
)

type PluginEndpoint struct {
	Name string
	APIs []string
	Auth bool
}

type SystemStatus struct {
	Version        string
	RedisStatus    bool
	MongoDBStatus  bool
	PostgresStatus bool
	Plugins        []string
	PluginPaths    []PluginEndpoint
	Apps           []string
	UserCount      int64
	LogCount       int64
	NewAdmin       bool   // 是否新创建的超级管理员
	AdminUser      string // 超级管理员用户名
	AdminPass      string // 超级管理员密码
}

// PrintCopyright 打印版权信息
func PrintCopyright(status SystemStatus) {
	// 清空屏幕
	fmt.Print("\033[H\033[2J")

	// 打印 Logo
	printLogo()

	// 打印系统信息框架
	printFrame(status)
}

func printFrame(status SystemStatus) {
	// 顶部边框
	titleColor.Println("| System Information")
	defaultColor.Println("│")

	// 版本信息
	defaultColor.Print("│ Version    : ")
	versionInfo := version.GetVersionInfo()
	versionColor.Printf("%s", versionInfo["version"])
	if hash, ok := versionInfo["git_commit"]; ok {
		defaultColor.Printf(" (")
		versionColor.Printf("%s", hash[:8])
		if branch, ok := versionInfo["git_branch"]; ok {
			defaultColor.Printf("@")
			versionColor.Printf("%s", branch)
		}
		defaultColor.Printf(")")
	}
	if buildTime, ok := versionInfo["build_time"]; ok {
		defaultColor.Printf(" built at %s", buildTime)
	}
	fmt.Println()

	// 数据库状态
	defaultColor.Println("│")
	defaultColor.Println("│ Database Status")
	defaultColor.Print("│ ⚡ Redis    : ")
	printStatus(status.RedisStatus)
	defaultColor.Print("│ ⚡ MongoDB  : ")
	printStatus(status.MongoDBStatus)
	defaultColor.Print("│ ⚡ Postgres : ")
	printStatus(status.PostgresStatus)

	// 应用信息
	defaultColor.Println("│")
	defaultColor.Println("│ Applications")
	if len(status.Apps) > 0 {
		for _, app := range status.Apps {
			defaultColor.Print("│ ⚡ ")
			successColor.Printf("%s\n", app)
		}
	} else {
		defaultColor.Print("│ ")
		warningColor.Println("No applications found")
	}

	// 插件信息
	defaultColor.Println("│")
	defaultColor.Println("│ Plugins")
	if len(status.PluginPaths) > 0 {
		for _, plugin := range status.PluginPaths {
			// 打印插件名称
			defaultColor.Print("│ ⚡ ")
			pluginColor.Printf("%s", plugin.Name)
			if plugin.Auth {
				warningColor.Print(" 🔒")
			} else {
				successColor.Print(" 🔓")
			}
			defaultColor.Println()

			// 打印插件的路由
			for _, api := range plugin.APIs {
				defaultColor.Print("│   └─ ")
				defaultColor.Printf("%s\n", api)
			}
			defaultColor.Println("│")
		}
	} else {
		defaultColor.Print("│ ")
		warningColor.Println("No plugins installed")
	}

	// 统计信息
	defaultColor.Println("│ Statistics")
	defaultColor.Print("│ ⚡ Users    : ")
	numberColor.Printf("%d\n", status.UserCount)
	defaultColor.Print("│ ⚡ Logs     : ")
	numberColor.Printf("%d\n", status.LogCount)

	// 版权信息
	defaultColor.Println("│")
	defaultColor.Print("│ ")
	titleColor.Print("Lauth")
	defaultColor.Print(" by ")
	defaultColor.Printf("Shuakami/Luoxiaohei\n")

	// 如果有新创建的超级管理员，显示凭据
	if status.NewAdmin && status.AdminUser != "" && status.AdminPass != "" {
		fmt.Println()
		alertColor := color.New(color.FgHiRed, color.Bold)

		alertColor.Println("!!! SUPER ADMIN CREDENTIALS CREATED !!!")
		alertColor.Printf("Username: %s\n", status.AdminUser)
		alertColor.Printf("Password: %s\n", status.AdminPass)
		alertColor.Println("PLEASE CHANGE YOUR PASSWORD AFTER FIRST LOGIN!")
		alertColor.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	}

	fmt.Println()
}

func printStatus(status bool) {
	if status {
		successColor.Print("Connected")
	} else {
		warningColor.Print("Disconnected")
	}
	fmt.Println()
}

func printLogo() {
	logo := `
     ___         __  __  
    /   | __  __/ /_/ /_ 
   / /| |/ / / / __/ __ \
  / ___ / /_/ / /_/ / / /
 /_/  |_\__,_/\__/_/ /_/ 
`
	lines := strings.Split(logo, "\n")
	for _, line := range lines {
		titleColor.Println(line)
	}
}
