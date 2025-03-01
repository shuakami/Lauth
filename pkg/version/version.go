package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

var (
	// 这些变量会在编译时通过ldflags注入
	Version   = "0.1.0"   // 版本号
	BuildTime = "unknown" // 构建时间
)

const (
	versionFile    = "VERSION"
	githubRepoAPI  = "https://api.github.com/repos/shuakami/lauth/releases/latest"
	defaultVersion = "0.1.0"
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version   string `json:"version"`    // 版本号
	BuildTime string `json:"build_time"` // 构建时间
	GoVersion string `json:"go_version"` // Go版本
	OS        string `json:"os"`         // 操作系统
	Arch      string `json:"arch"`       // 系统架构
}

// GetVersion 获取版本号
func GetVersion() string {
	return Version
}

// GetVersionInfo 获取详细版本信息
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}

// CheckLatestVersion 检查最新版本
func CheckLatestVersion() (*VersionInfo, error) {
	resp, err := http.Get(githubRepoAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to check latest version: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// 移除版本号前缀的'v'
	version := strings.TrimPrefix(release.TagName, "v")

	return &VersionInfo{
		Version: version,
	}, nil
}

// IsOutdated 检查当前版本是否过期
func IsOutdated() (bool, string, error) {
	latest, err := CheckLatestVersion()
	if err != nil {
		return false, "", err
	}

	if Version != latest.Version {
		return true, latest.Version, nil
	}

	return false, "", nil
}
