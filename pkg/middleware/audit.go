package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"lauth/internal/audit"
	"lauth/internal/service"

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

// EventTypeStrategy 事件类型策略接口
type EventTypeStrategy interface {
	DetermineEventType(path string, method string) audit.EventType
}

// PathMethodStrategy 路径方法策略
type PathMethodStrategy struct {
	eventTypeMap map[string]audit.EventType
}

// NewPathMethodStrategy 创建路径方法策略实例
func NewPathMethodStrategy() *PathMethodStrategy {
	strategy := &PathMethodStrategy{
		eventTypeMap: make(map[string]audit.EventType),
	}

	// 注册默认的事件类型映射
	strategy.RegisterEventType("/api/v1/auth/login", http.MethodPost, audit.EventLogin)
	strategy.RegisterEventType("/api/v1/auth/logout", http.MethodPost, audit.EventLogout)
	strategy.RegisterEventType("/api/v1/auth/refresh", http.MethodPost, audit.EventTokenRefresh)
	strategy.RegisterEventType("/api/v1/oauth/token", http.MethodPost, audit.EventTokenIssue)
	strategy.RegisterEventType("/api/v1/oauth/authorize", http.MethodPost, audit.EventAuthorize)
	strategy.RegisterEventType("/api/v1/oauth/revoke", http.MethodPost, audit.EventTokenRevoke)
	strategy.RegisterEventType("/api/v1/users", http.MethodPost, audit.EventUserCreate)
	strategy.RegisterEventType("/api/v1/users", http.MethodPut, audit.EventUserUpdate)
	strategy.RegisterEventType("/api/v1/users", http.MethodDelete, audit.EventUserDelete)
	strategy.RegisterEventType("/api/v1/apps", http.MethodPost, audit.EventAppCreate)
	strategy.RegisterEventType("/api/v1/apps", http.MethodPut, audit.EventAppUpdate)
	strategy.RegisterEventType("/api/v1/apps", http.MethodDelete, audit.EventAppDelete)
	strategy.RegisterEventType("/api/v1/oauth/clients", http.MethodPost, audit.EventClientCreate)
	strategy.RegisterEventType("/api/v1/oauth/clients", http.MethodPut, audit.EventClientUpdate)
	strategy.RegisterEventType("/api/v1/oauth/clients", http.MethodDelete, audit.EventClientDelete)

	return strategy
}

// RegisterEventType 注册事件类型
func (s *PathMethodStrategy) RegisterEventType(path string, method string, eventType audit.EventType) {
	key := s.generateKey(path, method)
	s.eventTypeMap[key] = eventType
}

// DetermineEventType 确定事件类型
func (s *PathMethodStrategy) DetermineEventType(path string, method string) audit.EventType {
	key := s.generateKey(path, method)
	if eventType, ok := s.eventTypeMap[key]; ok {
		return eventType
	}
	return ""
}

// generateKey 生成映射键
func (s *PathMethodStrategy) generateKey(path string, method string) string {
	return path + ":" + method
}

// AuditMiddleware 审计中间件
type AuditMiddleware struct {
	writer        *audit.Writer
	wsServer      *audit.WebSocketServer
	eventStrategy EventTypeStrategy
	ipService     service.IPLocationService
}

// NewAuditMiddleware 创建新的审计中间件
func NewAuditMiddleware(writer *audit.Writer, wsServer *audit.WebSocketServer, ipService service.IPLocationService) *AuditMiddleware {
	return &AuditMiddleware{
		writer:        writer,
		wsServer:      wsServer,
		eventStrategy: NewPathMethodStrategy(),
		ipService:     ipService,
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
			EventType:     m.eventStrategy.DetermineEventType(c.Request.URL.Path, c.Request.Method),
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

		// 如果是登录事件，异步查询位置信息
		if log.EventType == audit.EventLogin {
			go func(logID string, clientIP string) {
				// 查询IP位置信息
				location, err := m.ipService.SearchIP(c.Request.Context(), clientIP)
				if err != nil {
					return
				}

				// 更新审计日志
				locationInfo := map[string]interface{}{
					"country":  location.Country,
					"province": location.Province,
					"city":     location.City,
					"isp":      location.ISP,
				}

				// 创建位置更新日志
				locationLog := &audit.AuditLog{
					ID:            logID,
					Timestamp:     startTime,
					EventType:     log.EventType,
					UserID:        userID,
					AppID:         appID,
					ClientIP:      clientIP,
					RequestMethod: log.RequestMethod,
					RequestPath:   log.RequestPath,
					RequestQuery:  log.RequestQuery,
					UserAgent:     log.UserAgent,
					StatusCode:    log.StatusCode,
					Details: map[string]interface{}{
						"duration_ms": log.Details["duration_ms"],
						"headers":     log.Details["headers"],
						"location":    locationInfo,
					},
				}

				// 更新审计日志
				if err := m.writer.Write(locationLog); err != nil {
					return
				}

				// 通过WebSocket广播更新后的日志
				if m.wsServer != nil {
					m.wsServer.Broadcast(locationLog)
				}
			}(log.ID, log.ClientIP)
		}
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
