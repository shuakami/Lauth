package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EventType 审计事件类型
type EventType string

const (
	// 认证相关事件
	EventLogin        EventType = "login"
	EventLogout       EventType = "logout"
	EventTokenRefresh EventType = "token_refresh"
	EventTokenRevoke  EventType = "token_revoke"

	// 用户管理事件
	EventUserCreate EventType = "user_create"
	EventUserUpdate EventType = "user_update"
	EventUserDelete EventType = "user_delete"

	// 应用管理事件
	EventAppCreate EventType = "app_create"
	EventAppUpdate EventType = "app_update"
	EventAppDelete EventType = "app_delete"

	// OAuth/OIDC事件
	EventAuthorize    EventType = "authorize"
	EventTokenIssue   EventType = "token_issue"
	EventClientCreate EventType = "client_create"
	EventClientUpdate EventType = "client_update"
	EventClientDelete EventType = "client_delete"
)

// AuditLog 审计日志结构
type AuditLog struct {
	ID        string    `json:"id"`         // 日志ID
	Timestamp time.Time `json:"timestamp"`  // 时间戳
	EventType EventType `json:"event_type"` // 事件类型
	UserID    string    `json:"user_id"`    // 用户ID
	AppID     string    `json:"app_id"`     // 应用ID
	ClientIP  string    `json:"client_ip"`  // 客户端IP

	// 请求信息
	RequestMethod string `json:"request_method"` // HTTP方法
	RequestPath   string `json:"request_path"`   // 请求路径
	RequestQuery  string `json:"request_query"`  // 查询参数
	UserAgent     string `json:"user_agent"`     // User-Agent

	// 响应信息
	StatusCode int `json:"status_code"` // HTTP状态码

	// 事件详情
	Details map[string]interface{} `json:"details"` // 事件详细信息

	// 哈希链
	PrevHash string `json:"prev_hash"` // 前一条日志的哈希
	Hash     string `json:"hash"`      // 当前日志的哈希
}

// String 返回日志的JSON字符串表示
func (l *AuditLog) String() string {
	data, _ := json.Marshal(l)
	return string(data)
}

// QueryParams 审计日志查询参数
type QueryParams struct {
	StartTime  *time.Time  `json:"start_time"`  // 开始时间
	EndTime    *time.Time  `json:"end_time"`    // 结束时间
	EventTypes []EventType `json:"event_types"` // 事件类型
	UserID     string      `json:"user_id"`     // 用户ID
	AppID      string      `json:"app_id"`      // 应用ID
	ClientIP   string      `json:"client_ip"`   // 客户端IP
	StatusCode int         `json:"status_code"` // HTTP状态码
	Limit      int         `json:"limit"`       // 返回数量限制
	Offset     int         `json:"offset"`      // 偏移量
}

// LogIndex 日志索引结构
type LogIndex struct {
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Files     []LogFileInfo          `json:"files"`
	Stats     map[EventType]int      `json:"stats"`
	AppStats  map[string]AppLogStats `json:"app_stats"`
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Path      string            `json:"path"`       // 相对路径
	StartTime time.Time         `json:"start_time"` // 文件中最早的日志时间
	EndTime   time.Time         `json:"end_time"`   // 文件中最晚的日志时间
	Size      int64             `json:"size"`       // 文件大小
	Events    map[EventType]int `json:"events"`     // 事件类型统计
	AppID     string            `json:"app_id"`     // 所属应用ID
}

// AppLogStats 应用日志统计
type AppLogStats struct {
	TotalLogs  int               `json:"total_logs"`  // 总日志数
	EventStats map[EventType]int `json:"event_stats"` // 事件统计
	Size       int64             `json:"size"`        // 总大小
}

// LogPath 日志路径结构
type LogPath struct {
	AppID string // 应用ID
	Year  int    // 年
	Month int    // 月
	Day   int    // 日
	Name  string // 文件名
}

// NewLogPath 创建日志路径
func NewLogPath(appID string, t time.Time) LogPath {
	return LogPath{
		AppID: appID,
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
		Name:  fmt.Sprintf("audit-%s.log", t.Format("20060102-150405")),
	}
}

// String 返回完整的相对路径
func (p LogPath) String() string {
	if p.AppID == "" {
		p.AppID = "default"
	}
	return filepath.Join(
		p.AppID,
		fmt.Sprintf("%04d", p.Year),
		fmt.Sprintf("%02d", p.Month),
		fmt.Sprintf("%02d", p.Day),
		p.Name,
	)
}

// ParseLogPath 从路径解析LogPath
func ParseLogPath(path string) (LogPath, error) {
	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) < 5 {
		return LogPath{}, fmt.Errorf("invalid log path format: %s", path)
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return LogPath{}, fmt.Errorf("invalid year format: %s", parts[1])
	}

	month, err := strconv.Atoi(parts[2])
	if err != nil {
		return LogPath{}, fmt.Errorf("invalid month format: %s", parts[2])
	}

	day, err := strconv.Atoi(parts[3])
	if err != nil {
		return LogPath{}, fmt.Errorf("invalid day format: %s", parts[3])
	}

	return LogPath{
		AppID: parts[0],
		Year:  year,
		Month: month,
		Day:   day,
		Name:  parts[4],
	}, nil
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`    // 消息类型
	Payload interface{} `json:"payload"` // 消息内容
}
