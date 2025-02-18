package email

import (
	"net/mail"
)

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"lauth/pkg/config"
)

// EmailSender 邮件发送器接口
type EmailSender interface {
	// SendHTML 发送HTML邮件
	SendHTML(to []string, subject string, templateName string, data interface{}) error
	// SendText 发送纯文本邮件
	SendText(to []string, subject string, content string) error
}

// SMTPSender SMTP邮件发送器
type SMTPSender struct {
	config    *config.SMTPConfig
	templates *TemplateManager
	auth      smtp.Auth
}

// NewSMTPSender 创建SMTP邮件发送器
func NewSMTPSender(config *config.SMTPConfig) (*SMTPSender, error) {
	// 创建模板管理器
	templates := NewTemplateManager(config.TemplatePath)

	// 根据不同邮件服务商选择合适的认证方式
	var auth smtp.Auth
	switch {
	case strings.Contains(config.Host, "gmail.com"):
		// Gmail需要使用OAuth2或应用专用密码
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	case strings.Contains(config.Host, "qq.com"):
		// QQ邮箱使用授权码作为密码
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	case strings.Contains(config.Host, "163.com"):
		// 163邮箱使用授权码作为密码
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	default:
		// 默认使用PlainAuth
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	return &SMTPSender{
		config:    config,
		templates: templates,
		auth:      auth,
	}, nil
}

// SendHTML 发送HTML邮件
func (s *SMTPSender) SendHTML(to []string, subject string, templateName string, data interface{}) error {
	// Sanitize the 'to' field
	sanitizedTo := sanitizeEmails(to)

	// 加载模板
	if err := s.templates.LoadTemplate(templateName); err != nil {
		return fmt.Errorf("failed to load template: %v", err)
	}

	// 执行模板
	body, err := s.templates.Execute(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// 构建邮件内容
	message := s.buildMessage(sanitizedTo, subject, body, true)

	// 发送邮件
	return s.send(sanitizedTo, message)
}

// SendText 发送纯文本邮件
func (s *SMTPSender) SendText(to []string, subject string, content string) error {
	// 构建邮件内容
	message := s.buildMessage(to, subject, content, false)

	// 发送邮件
	return s.send(to, message)
}

// AddTemplate 添加HTML模板
func (s *SMTPSender) AddTemplate(name string, content string) error {
	return s.templates.AddTemplate(name, content)
}

// GetTemplateNames 获取所有已注册的模板名称
func (s *SMTPSender) GetTemplateNames() []string {
	return s.templates.GetTemplateNames()
}

// buildMessage 构建邮件内容
func (s *SMTPSender) buildMessage(to []string, subject string, body string, isHTML bool) []byte {
	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.FromEmail)
	headers["To"] = strings.Join(to, ",")
	headers["Subject"] = subject

	if isHTML {
		headers["Content-Type"] = "text/html; charset=UTF-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	// 拼接邮件内容
	message := ""
	for key, value := range headers {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	message += "\r\n" + body

	return []byte(message)
}

// sanitizeEmails sanitizes a list of email addresses
func sanitizeEmails(emails []string) []string {
	var sanitized []string
	for _, email := range emails {
		if _, err := mail.ParseAddress(email); err == nil {
			sanitized = append(sanitized, email)
		}
	}
	return sanitized
}

// createSMTPConnection 创建SMTP连接
func (s *SMTPSender) createSMTPConnection(dialer *net.Dialer, addr string) (net.Conn, *smtp.Client, error) {
	var conn net.Conn
	var err error

	switch s.config.Port {
	case 465:
		// 直接TLS连接
		tlsConfig := &tls.Config{
			ServerName:         s.config.Host,
			InsecureSkipVerify: s.config.InsecureSkipVerify,
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	case 587:
		// 先普通连接，然后升级到TLS（STARTTLS）
		conn, err = dialer.Dial("tcp", addr)
		if err == nil {
			client, err := smtp.NewClient(conn, s.config.Host)
			if err != nil {
				conn.Close()
				return nil, nil, fmt.Errorf("failed to create SMTP client: %v", err)
			}

			// 尝试STARTTLS
			if err = client.StartTLS(&tls.Config{
				ServerName:         s.config.Host,
				InsecureSkipVerify: s.config.InsecureSkipVerify,
			}); err != nil {
				client.Close()
				return nil, nil, fmt.Errorf("STARTTLS failed: %v", err)
			}

			return conn, client, nil
		}
	default:
		// 普通连接（不推荐）
		conn, err = dialer.Dial("tcp", addr)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to SMTP server: %v", err)
	}

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to create SMTP client: %v", err)
	}

	return conn, client, nil
}

// sendMailContent 发送邮件内容
func (s *SMTPSender) sendMailContent(client *smtp.Client, to []string, message []byte) error {
	// 认证
	if err := client.Auth(s.auth); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}

	// 设置发件人
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// 设置收件人
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", addr, err)
		}
	}

	// 发送邮件内容
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create message writer: %v", err)
	}
	defer w.Close()

	if _, err = w.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}

// send 发送邮件
func (s *SMTPSender) send(to []string, message []byte) error {
	// 设置连接超时
	dialer := &net.Dialer{
		Timeout:   time.Duration(s.config.ConnectTimeout) * time.Second,
		KeepAlive: time.Duration(s.config.ConnectTimeout) * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 创建SMTP连接
	conn, client, err := s.createSMTPConnection(dialer, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	// 发送邮件内容
	return s.sendMailContent(client, to, message)
}

// LogSender 日志发送器（用于测试）
type LogSender struct {
	templates *TemplateManager
}

// newLogSender 创建日志发送器
func newLogSender() *LogSender {
	return &LogSender{
		templates: NewTemplateManager(""),
	}
}

// SendText 发送纯文本邮件（仅打印日志）
func (s *LogSender) SendText(to []string, subject string, content string) error {
	fmt.Printf("[TEXT] Send email to %v\nSubject: %s\nContent: %s\n", to, subject, content)
	return nil
}

// SendHTML 发送HTML邮件（仅打印日志）
func (s *LogSender) SendHTML(to []string, subject string, templateName string, data interface{}) error {
	fmt.Printf("[HTML] Send email to %v\nSubject: %s\nTemplate: %s\nData: %+v\n", to, subject, templateName, data)
	return nil
}

// AddTemplate 添加HTML模板（仅打印日志）
func (s *LogSender) AddTemplate(name string, content string) error {
	fmt.Printf("[TEMPLATE] Add template %s: %s\n", name, content)
	return s.templates.AddTemplate(name, content)
}

// GetTemplateNames 获取所有已注册的模板名称
func (s *LogSender) GetTemplateNames() []string {
	return s.templates.GetTemplateNames()
}
