package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 主函数：扫描插件目录并生成导入代码
func main() {
	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	// 构建绝对路径 - 注意：从当前工作目录开始，不要追加多余的路径
	pluginDir := filepath.Join(workDir, "internal", "plugin")
	outputFile := filepath.Join(pluginDir, "import_gen.go")

	fmt.Printf("扫描目录: %s\n", pluginDir)

	// 检查目录是否存在
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		fmt.Printf("插件目录不存在: %s\n", pluginDir)
		os.Exit(1)
	}

	// 找到所有可能的插件目录
	pluginPaths, err := findPluginDirs(pluginDir)
	if err != nil {
		fmt.Printf("扫描插件目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("发现插件: %v\n", pluginPaths)

	// 生成导入代码
	if err := generateImportFile(pluginPaths, outputFile); err != nil {
		fmt.Printf("生成导入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("成功生成插件导入文件: %s\n", outputFile)
}

// 查找插件目录
func findPluginDirs(rootDir string) ([]string, error) {
	var pluginPaths []string

	// 读取根目录
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	// 遍历目录项
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过特殊目录和非插件目录
		dirName := entry.Name()
		if dirName == "types" || dirName == "user" {
			continue
		}

		// 检查是否是插件目录
		dirPath := filepath.Join(rootDir, dirName)

		// 调试信息
		fmt.Printf("检查目录: %s\n", dirPath)

		// 使用更严格的判断条件确认是否为插件目录
		isPlugin, err := isValidPluginDir(dirPath)
		if err != nil {
			fmt.Printf("  检查目录出错 %s: %v\n", dirPath, err)
			continue
		}

		if isPlugin {
			// 将路径转换为导入路径格式
			importPath := fmt.Sprintf("lauth/internal/plugin/%s", dirName)
			pluginPaths = append(pluginPaths, importPath)
			fmt.Printf("  确认为插件目录: %s\n", dirPath)
		} else {
			fmt.Printf("  不是有效的插件目录: %s\n", dirPath)
		}
	}

	return pluginPaths, nil
}

// isValidPluginDir 使用更严格的条件判断目录是否为有效的插件目录
func isValidPluginDir(dirPath string) (bool, error) {
	// 1. 检查是否存在init.go文件
	if _, err := os.Stat(filepath.Join(dirPath, "init.go")); err == nil {
		return true, nil
	}

	// 2. 检查目录中的Go文件是否包含插件注册相关代码
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}

		// 读取文件内容
		filePath := filepath.Join(dirPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		// 检查是否包含插件注册相关代码
		if containsPluginRegistration(string(content)) {
			return true, nil
		}
	}

	return false, nil
}

// containsPluginRegistration 检查代码是否包含插件注册相关内容
func containsPluginRegistration(content string) bool {
	patterns := []string{
		`types\.RegisterPlugin\(`,
		`func\s+init\(\)\s*{`,
		`RegisterPlugin\(`,
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(content) {
			return true
		}
	}

	return false
}

// 生成导入文件
func generateImportFile(pluginPaths []string, outputFile string) error {
	// 生成代码内容
	var code strings.Builder
	code.WriteString("// 此文件由plugin_scanner自动生成，请勿手动修改\n")
	code.WriteString("package plugin\n\n")
	code.WriteString("import (\n")

	// 为每个插件路径添加一个导入语句
	for _, path := range pluginPaths {
		code.WriteString(fmt.Sprintf("\t_ \"%s\" // 自动导入插件\n", path))
	}

	code.WriteString(")\n\n")
	code.WriteString("// AutoImportPlugins 此函数仅用于文档目的\n")
	code.WriteString("// 插件会通过上方的导入语句自动注册\n")
	code.WriteString("func AutoImportPlugins() {\n")
	code.WriteString("\t// 自动生成的函数，不需要调用\n")
	code.WriteString("}\n")

	// 写入文件
	return os.WriteFile(outputFile, []byte(code.String()), 0644)
}
