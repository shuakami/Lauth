package v1

import (
	"net/http"
	"sort"
	"time"

	"lauth/internal/audit"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 在生产环境中应该根据实际需求限制来源
	},
}

// AuditHandler 审计处理器
type AuditHandler struct {
	reader   *audit.Reader
	wsServer *audit.WebSocketServer
}

// NewAuditHandler 创建审计处理器实例
func NewAuditHandler(reader *audit.Reader, wsServer *audit.WebSocketServer) *AuditHandler {
	return &AuditHandler{
		reader:   reader,
		wsServer: wsServer,
	}
}

// Register 注册路由
func (h *AuditHandler) Register(r *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	auditGroup := r.Group("/audit", authMiddleware.HandleAuth())
	{
		auditGroup.GET("/logs", h.GetLogs)         // 查询日志
		auditGroup.GET("/ws", h.HandleWebSocket)   // WebSocket连接
		auditGroup.GET("/verify", h.VerifyLogFile) // 验证日志文件
		auditGroup.GET("/stats", h.GetStats)       // 获取统计信息
	}
}

// LogQueryRequest 日志查询请求
type LogQueryRequest struct {
	StartTime  *time.Time        `form:"start_time"`
	EndTime    *time.Time        `form:"end_time"`
	EventTypes []audit.EventType `form:"event_types"`
	UserID     string            `form:"user_id"`
	AppID      string            `form:"app_id"`
	ClientIP   string            `form:"client_ip"`
	StatusCode int               `form:"status_code"`
	Limit      int               `form:"limit"`
	Offset     int               `form:"offset"`
}

// GetLogs 查询审计日志
func (h *AuditHandler) GetLogs(c *gin.Context) {
	var req LogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	// 转换为查询参数
	params := audit.QueryParams{
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		EventTypes: req.EventTypes,
		UserID:     req.UserID,
		AppID:      req.AppID,
		ClientIP:   req.ClientIP,
		StatusCode: req.StatusCode,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	// 查询日志
	logs, err := h.reader.ReadLogs(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果没有找到日志，返回空数组而不是null
	if logs == nil {
		logs = []*audit.AuditLog{}
	}

	// 按时间戳降序排序
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})

	c.JSON(http.StatusOK, gin.H{
		"total": len(logs),
		"items": logs,
	})
}

// HandleWebSocket 处理WebSocket连接
func (h *AuditHandler) HandleWebSocket(c *gin.Context) {
	// 从上下文获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// 添加到WebSocket服务器,传入appID
	h.wsServer.AddClient(conn, claims.AppID)
}

// VerifyLogFile 验证日志文件完整性
func (h *AuditHandler) VerifyLogFile(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename is required"})
		return
	}

	valid, err := h.reader.VerifyLogFile(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"valid":    valid,
	})
}

// GetStats 获取审计统计信息
func (h *AuditHandler) GetStats(c *gin.Context) {
	// 获取时间范围
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	// 获取appID
	appID := c.Query("app_id")

	// 查询日志
	logs, err := h.reader.ReadLogs(audit.QueryParams{
		StartTime: &startTime,
		EndTime:   &endTime,
		AppID:     appID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计信息
	stats := map[string]interface{}{
		"total_logs":        len(logs),
		"connected_clients": h.wsServer.GetClientCount(),
		"event_types":       make(map[audit.EventType]int),
		"status_codes":      make(map[int]int),
		"time_range": map[string]string{
			"start": startTime.Format(time.RFC3339),
			"end":   endTime.Format(time.RFC3339),
		},
	}

	// 统计事件类型和状态码
	for _, log := range logs {
		stats["event_types"].(map[audit.EventType]int)[log.EventType]++
		stats["status_codes"].(map[int]int)[log.StatusCode]++
	}

	c.JSON(http.StatusOK, stats)
}
