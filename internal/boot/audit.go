package boot

import (
	"time"

	"lauth/internal/audit"
	"lauth/internal/service"
	"lauth/pkg/config"
)

// AuditComponents 包含审计相关组件
type AuditComponents struct {
	Writer                    *audit.Writer
	Reader                    *audit.Reader
	WebSocketServer           *audit.WebSocketServer
	AuditPermissionMiddleware *audit.AuditPermissionMiddleware
}

// InitAudit 初始化审计相关组件
func InitAudit(cfg *config.Config, roleService service.RoleService) (*AuditComponents, error) {
	// 初始化哈希链
	hashChain := audit.NewHashChain()

	// 初始化审计日志写入器
	writer, err := audit.NewWriter(audit.WriterConfig{
		BaseDir:    cfg.Audit.LogDir,
		RotateSize: cfg.Audit.RotationSize,
		HashChain:  hashChain,
	})
	if err != nil {
		return nil, err
	}

	// 初始化WebSocket服务器
	wsServer := audit.NewWebSocketServer(&audit.WebSocketConfig{
		PingInterval:   time.Duration(cfg.Audit.WebSocket.PingInterval) * time.Second,
		WriteWait:      time.Duration(cfg.Audit.WebSocket.WriteWait) * time.Second,
		ReadWait:       time.Duration(cfg.Audit.WebSocket.ReadWait) * time.Second,
		MaxMessageSize: int64(cfg.Audit.WebSocket.MaxMessageSize),
	})

	// 初始化审计日志读取器
	reader := audit.NewReader(cfg.Audit.LogDir)

	// 初始化审计日志权限中间件
	permissionMiddleware := audit.NewAuditPermissionMiddleware(roleService)

	return &AuditComponents{
		Writer:                    writer,
		Reader:                    reader,
		WebSocketServer:           wsServer,
		AuditPermissionMiddleware: permissionMiddleware,
	}, nil
}
