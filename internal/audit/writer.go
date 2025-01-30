package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Writer 审计日志写入器
type Writer struct {
	mu           sync.Mutex
	baseDir      string               // 基础目录
	currentFiles map[string]*os.File  // 当前打开的文件，按应用ID索引
	hashChain    *HashChain           // 哈希链
	rotateSize   int64                // 日志文件轮转大小（字节）
	currentSizes map[string]int64     // 当前文件大小，按应用ID索引
	indexes      map[string]*LogIndex // 应用索引，按应用ID索引
}

// WriterConfig 写入器配置
type WriterConfig struct {
	BaseDir    string     // 基础目录
	RotateSize int64      // 日志文件轮转大小（字节），默认100MB
	HashChain  *HashChain // 哈希链
}

// NewWriter 创建新的日志写入器
func NewWriter(config WriterConfig) (*Writer, error) {
	if config.RotateSize <= 0 {
		config.RotateSize = 100 * 1024 * 1024 // 默认100MB
	}

	fmt.Printf("Attempting to create audit log directory: %s\n", config.BaseDir)

	// 创建基础目录
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %v", config.BaseDir, err)
	}

	writer := &Writer{
		baseDir:      config.BaseDir,
		currentFiles: make(map[string]*os.File),
		hashChain:    config.HashChain,
		rotateSize:   config.RotateSize,
		currentSizes: make(map[string]int64),
		indexes:      make(map[string]*LogIndex),
	}

	return writer, nil
}

// Write 写入审计日志
func (w *Writer) Write(log *AuditLog) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 生成日志ID
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// 设置时间戳
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	// 如果没有指定应用ID，使用default
	if log.AppID == "" {
		log.AppID = "default"
	}

	// 添加到哈希链
	if err := w.hashChain.AddLog(log); err != nil {
		return fmt.Errorf("failed to add log to hash chain: %v", err)
	}

	// 获取或创建当前文件
	file, err := w.getCurrentFile(log.AppID, log.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to get current file: %v", err)
	}

	// 序列化日志
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %v", err)
	}
	data = append(data, '\n')

	// 写入日志文件
	n, err := file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write log: %v", err)
	}

	// 更新文件大小
	w.currentSizes[log.AppID] += int64(n)

	// 更新索引
	if err := w.updateIndex(log); err != nil {
		return fmt.Errorf("failed to update index: %v", err)
	}

	// 检查是否需要轮转
	if w.currentSizes[log.AppID] >= w.rotateSize {
		if err := w.rotate(log.AppID); err != nil {
			return fmt.Errorf("failed to rotate log file: %v", err)
		}
	}

	return nil
}

// getCurrentFile 获取当前日志文件
func (w *Writer) getCurrentFile(appID string, t time.Time) (*os.File, error) {
	// 检查是否已有打开的文件
	if file, ok := w.currentFiles[appID]; ok {
		return file, nil
	}

	// 创建新文件
	return w.createNewFile(appID, t)
}

// createNewFile 创建新的日志文件
func (w *Writer) createNewFile(appID string, t time.Time) (*os.File, error) {
	// 生成日志路径
	logPath := NewLogPath(appID, t)
	fullPath := filepath.Join(w.baseDir, logPath.String())

	// 创建目录
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// 创建文件
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file %s: %v", fullPath, err)
	}

	// 更新当前文件和大小
	w.currentFiles[appID] = file
	w.currentSizes[appID] = 0

	return file, nil
}

// rotate 轮转日志文件
func (w *Writer) rotate(appID string) error {
	// 关闭当前文件
	if file, ok := w.currentFiles[appID]; ok {
		file.Close()
		delete(w.currentFiles, appID)
		delete(w.currentSizes, appID)
	}

	return nil
}

// updateIndex 更新索引
func (w *Writer) updateIndex(log *AuditLog) error {
	// 获取或创建应用索引
	index, ok := w.indexes[log.AppID]
	if !ok {
		index = &LogIndex{
			StartTime: log.Timestamp,
			EndTime:   log.Timestamp,
			Files:     make([]LogFileInfo, 0),
			Stats:     make(map[EventType]int),
			AppStats:  make(map[string]AppLogStats),
		}
		w.indexes[log.AppID] = index
	}

	// 更新时间范围
	if log.Timestamp.Before(index.StartTime) {
		index.StartTime = log.Timestamp
	}
	if log.Timestamp.After(index.EndTime) {
		index.EndTime = log.Timestamp
	}

	// 更新统计信息
	index.Stats[log.EventType]++
	appStats := index.AppStats[log.AppID]
	appStats.TotalLogs++
	if appStats.EventStats == nil {
		appStats.EventStats = make(map[EventType]int)
	}
	appStats.EventStats[log.EventType]++
	appStats.Size = w.currentSizes[log.AppID]
	index.AppStats[log.AppID] = appStats

	// 保存索引文件
	return w.saveIndex(log.AppID)
}

// saveIndex 保存索引到文件
func (w *Writer) saveIndex(appID string) error {
	index := w.indexes[appID]
	if index == nil {
		return nil
	}

	// 生成索引文件路径
	indexPath := filepath.Join(w.baseDir, appID, "index.json")

	// 创建目录
	dir := filepath.Dir(indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %v", err)
	}

	// 序列化索引
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %v", err)
	}

	return nil
}

// Close 关闭写入器
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var lastErr error
	for appID, file := range w.currentFiles {
		if err := file.Close(); err != nil {
			lastErr = err
		}
		delete(w.currentFiles, appID)
		delete(w.currentSizes, appID)
	}

	return lastErr
}

// GetCurrentFile 获取应用当前日志文件路径
func (w *Writer) GetCurrentFile(appID string) string {
	w.mu.Lock()
	defer w.mu.Unlock()

	if file, ok := w.currentFiles[appID]; ok {
		return file.Name()
	}
	return ""
}

// GetHashChain 获取哈希链
func (w *Writer) GetHashChain() *HashChain {
	return w.hashChain
}
