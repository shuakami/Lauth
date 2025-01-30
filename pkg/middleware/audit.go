package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"lauth/internal/audit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// responseWriter 自定义响应写入器，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditMiddleware 审计中间件
type AuditMiddleware struct {
	writer   *audit.Writer
	wsServer *audit.WebSocketServer
}

// NewAuditMiddleware 创建新的审计中间件
func NewAuditMiddleware(writer *audit.Writer, wsServer *audit.WebSocketServer) *AuditMiddleware {
	return &AuditMiddleware{
		writer:   writer,
		wsServer: wsServer,
	}
}

// Handle 处理请求
func (m *AuditMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 捕获请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 包装响应写入器
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = w

		// 处理请求
		c.Next()

		// 获取用户信息
		var userID, appID string
		if claims := GetUserFromContext(c); claims != nil {
			userID = claims.UserID
			appID = claims.AppID
		}

		// 创建审计日志
		log := &audit.AuditLog{
			ID:            uuid.New().String(),
			Timestamp:     startTime,
			EventType:     determineEventType(c),
			UserID:        userID,
			AppID:         appID,
			ClientIP:      c.ClientIP(),
			RequestMethod: c.Request.Method,
			RequestPath:   c.Request.URL.Path,
			RequestQuery:  c.Request.URL.RawQuery,
			UserAgent:     c.Request.UserAgent(),
			StatusCode:    c.Writer.Status(),
			Details: map[string]interface{}{
				"duration_ms": time.Since(startTime).Milliseconds(),
				"headers":     getHeaders(c.Request),
			},
		}

		// 写入审计日志
		if err := m.writer.Write(log); err != nil {
			c.Error(err)
		}

		// 通过WebSocket广播日志
		if m.wsServer != nil {
			m.wsServer.Broadcast(log)
		}
	}
}

// determineEventType 根据请求确定事件类型
func determineEventType(c *gin.Context) audit.EventType {
	path := c.Request.URL.Path
	method := c.Request.Method

	switch {
	case path == "/api/v1/auth/login":
		return audit.EventLogin
	case path == "/api/v1/auth/logout":
		return audit.EventLogout
	case path == "/api/v1/auth/refresh":
		return audit.EventTokenRefresh
	case path == "/api/v1/oauth/token":
		return audit.EventTokenIssue
	case path == "/api/v1/oauth/authorize":
		return audit.EventAuthorize
	case path == "/api/v1/oauth/revoke":
		return audit.EventTokenRevoke
	case path == "/api/v1/users" && method == http.MethodPost:
		return audit.EventUserCreate
	case path == "/api/v1/users" && method == http.MethodPut:
		return audit.EventUserUpdate
	case path == "/api/v1/users" && method == http.MethodDelete:
		return audit.EventUserDelete
	case path == "/api/v1/apps" && method == http.MethodPost:
		return audit.EventAppCreate
	case path == "/api/v1/apps" && method == http.MethodPut:
		return audit.EventAppUpdate
	case path == "/api/v1/apps" && method == http.MethodDelete:
		return audit.EventAppDelete
	case path == "/api/v1/oauth/clients" && method == http.MethodPost:
		return audit.EventClientCreate
	case path == "/api/v1/oauth/clients" && method == http.MethodPut:
		return audit.EventClientUpdate
	case path == "/api/v1/oauth/clients" && method == http.MethodDelete:
		return audit.EventClientDelete
	default:
		return ""
	}
}

// getHeaders 获取请求头信息（排除敏感信息）
func getHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for k, v := range r.Header {
		// 排除敏感头部
		switch k {
		case "Authorization", "Cookie", "Set-Cookie":
			continue
		default:
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
	}
	return headers
}
