package email

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sync"
)

// TemplateManager 邮件模板管理器
type TemplateManager struct {
	templatePath string
	templates    map[string]*template.Template
	mu           sync.RWMutex
}

// NewTemplateManager 创建模板管理器
func NewTemplateManager(templatePath string) *TemplateManager {
	return &TemplateManager{
		templatePath: templatePath,
		templates:    make(map[string]*template.Template),
	}
}

// LoadTemplate 加载模板
func (m *TemplateManager) LoadTemplate(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查模板是否已加载
	if _, exists := m.templates[name]; exists {
		return nil
	}

	// 构建模板文件路径
	templateFile := filepath.Join(m.templatePath, name+".html")

	// 解析模板
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %v", name, err)
	}

	// 存储模板
	m.templates[name] = tmpl
	return nil
}

// Execute 执行模板
func (m *TemplateManager) Execute(name string, data interface{}) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取模板
	tmpl, exists := m.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %v", name, err)
	}

	return buf.String(), nil
}

// AddTemplate 添加HTML模板
func (m *TemplateManager) AddTemplate(name string, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 解析模板内容
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template content: %v", err)
	}

	// 存储模板
	m.templates[name] = tmpl
	return nil
}

// GetTemplateNames 获取所有已注册的模板名称
func (m *TemplateManager) GetTemplateNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.templates))
	for name := range m.templates {
		names = append(names, name)
	}
	return names
}
