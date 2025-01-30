package audit

import (
	"encoding/json"
	stdlog "log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	PingInterval   time.Duration // 心跳间隔
	WriteWait      time.Duration // 写超时
	ReadWait       time.Duration // 读超时
	MaxMessageSize int64         // 最大消息大小
}

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	mu        sync.RWMutex
	clients   map[*websocket.Conn]*ClientInfo
	broadcast chan *AuditLog
	config    *WebSocketConfig
}

// ClientInfo 客户端信息
type ClientInfo struct {
	AppID string
	Conn  *websocket.Conn
}

// NewWebSocketServer 创建新的WebSocket服务器
func NewWebSocketServer(config *WebSocketConfig) *WebSocketServer {
	if config == nil {
		config = &WebSocketConfig{
			PingInterval:   30 * time.Second,
			WriteWait:      10 * time.Second,
			ReadWait:       60 * time.Second,
			MaxMessageSize: 1024,
		}
	}
	return &WebSocketServer{
		clients:   make(map[*websocket.Conn]*ClientInfo),
		broadcast: make(chan *AuditLog, 100), // 缓冲区大小为100
		config:    config,
	}
}

// Start 启动WebSocket服务器
func (s *WebSocketServer) Start() {
	for {
		select {
		case log := <-s.broadcast:
			s.broadcastLog(log)
		}
	}
}

// AddClient 添加新的WebSocket客户端
func (s *WebSocketServer) AddClient(conn *websocket.Conn, appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[conn] = &ClientInfo{
		AppID: appID,
		Conn:  conn,
	}

	// 启动客户端读取协程
	go s.readPump(conn)
}

// RemoveClient 移除WebSocket客户端
func (s *WebSocketServer) RemoveClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[conn]; ok {
		delete(s.clients, conn)
		conn.Close()
	}
}

// Broadcast 广播审计日志
func (s *WebSocketServer) Broadcast(log *AuditLog) {
	s.broadcast <- log
}

// broadcastLog 向所有客户端广播日志
func (s *WebSocketServer) broadcastLog(log *AuditLog) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	message := WebSocketMessage{
		Type:    "audit_log",
		Payload: log,
	}

	data, err := json.Marshal(message)
	if err != nil {
		stdlog.Printf("Failed to marshal websocket message: %v", err)
		return
	}

	for conn, info := range s.clients {
		// 只发送给相同应用的客户端
		if info.AppID == log.AppID {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				stdlog.Printf("Failed to write to websocket: %v", err)
				s.RemoveClient(conn)
				continue
			}
		}
	}
}

// readPump 处理来自客户端的消息
func (s *WebSocketServer) readPump(conn *websocket.Conn) {
	defer func() {
		s.RemoveClient(conn)
	}()

	for {
		// 读取消息（主要用于检测连接状态）
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				stdlog.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// GetClientCount 获取当前连接的客户端数量
func (s *WebSocketServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
