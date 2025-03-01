package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Reader 审计日志读取器
type Reader struct {
	baseDir string               // 基础目录
	indexes map[string]*LogIndex // 应用索引缓存
}

// NewReader 创建新的日志读取器
func NewReader(baseDir string) *Reader {
	return &Reader{
		baseDir: baseDir,
		indexes: make(map[string]*LogIndex),
	}
}

// ReadLogs 读取指定时间范围内的日志
func (r *Reader) ReadLogs(params QueryParams) ([]*AuditLog, error) {
	// 如果指定了应用ID，只读取该应用的日志
	if params.AppID != "" {
		return r.readAppLogs(params.AppID, params)
	}

	// 否则读取所有应用的日志
	var allLogs []*AuditLog
	apps, err := r.listApps()
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %v", err)
	}

	for _, appID := range apps {
		logs, err := r.readAppLogs(appID, params)
		if err != nil {
			return nil, fmt.Errorf("failed to read logs for app %s: %v", appID, err)
		}
		allLogs = append(allLogs, logs...)
	}

	// 按时间戳排序
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp.Before(allLogs[j].Timestamp)
	})

	// 应用分页
	if params.Offset >= len(allLogs) {
		return []*AuditLog{}, nil
	}
	end := params.Offset + params.Limit
	if end > len(allLogs) {
		end = len(allLogs)
	}
	return allLogs[params.Offset:end], nil
}

// validateAppID 验证应用ID
func (r *Reader) validateAppID(appID string) error {
	if strings.Contains(appID, "/") || strings.Contains(appID, "\\") || strings.Contains(appID, "..") {
		return fmt.Errorf("invalid appID: %s", appID)
	}
	return nil
}

// readHistoricalLogs 读取历史日志
func (r *Reader) readHistoricalLogs(index *LogIndex, params QueryParams) ([]*AuditLog, error) {
	var logs []*AuditLog
	for _, fileInfo := range index.Files {
		if r.isFileRelevant(fileInfo, params) {
			fileLogs, err := r.readFile(filepath.Join(r.baseDir, fileInfo.Path), params)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %v", fileInfo.Path, err)
			}
			logs = append(logs, fileLogs...)
		}
	}
	return logs, nil
}

// scanCurrentDayLogs 扫描当前日期的日志
func (r *Reader) scanCurrentDayLogs(appID string, params QueryParams) ([]*AuditLog, error) {
	now := time.Now()
	currentDayPath := filepath.Join(
		r.baseDir,
		appID,
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	if _, err := os.Stat(currentDayPath); err != nil {
		if os.IsNotExist(err) {
			return []*AuditLog{}, nil
		}
		return nil, fmt.Errorf("failed to access current day directory: %v", err)
	}

	files, err := os.ReadDir(currentDayPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read current day directory: %v", err)
	}

	var logs []*AuditLog
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "audit-") {
			filePath := filepath.Join(currentDayPath, file.Name())
			fileLogs, err := r.readFile(filePath, params)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %v", file.Name(), err)
			}
			logs = append(logs, fileLogs...)
		}
	}
	return logs, nil
}

// mergeSortLogs 合并并排序日志
func (r *Reader) mergeSortLogs(logs ...[]*AuditLog) []*AuditLog {
	var merged []*AuditLog
	for _, logSet := range logs {
		merged = append(merged, logSet...)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp.Before(merged[j].Timestamp)
	})
	return merged
}

// readAppLogs 读取指定应用的日志
func (r *Reader) readAppLogs(appID string, params QueryParams) ([]*AuditLog, error) {
	// 验证应用ID
	if err := r.validateAppID(appID); err != nil {
		return nil, err
	}

	// 加载应用索引
	index, err := r.loadIndex(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to load index for app %s: %v", appID, err)
	}

	// 如果没有索引，说明没有日志
	if index == nil {
		return []*AuditLog{}, nil
	}

	// 读取历史日志
	historicalLogs, err := r.readHistoricalLogs(index, params)
	if err != nil {
		return nil, err
	}

	// 扫描当前日期的日志
	currentLogs, err := r.scanCurrentDayLogs(appID, params)
	if err != nil {
		return nil, err
	}

	// 合并并排序所有日志
	return r.mergeSortLogs(historicalLogs, currentLogs), nil
}

// loadIndex 加载应用索引
func (r *Reader) loadIndex(appID string) (*LogIndex, error) {
	// 检查缓存
	if index, ok := r.indexes[appID]; ok {
		return index, nil
	}

	// 读取索引文件
	indexPath := filepath.Join(r.baseDir, appID, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var index LogIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	// 更新缓存
	r.indexes[appID] = &index
	return &index, nil
}

// isFileRelevant 检查文件是否与查询参数相关
func (r *Reader) isFileRelevant(fileInfo LogFileInfo, params QueryParams) bool {
	// 检查时间范围
	if params.StartTime != nil && fileInfo.EndTime.Before(*params.StartTime) {
		return false
	}
	if params.EndTime != nil && fileInfo.StartTime.After(*params.EndTime) {
		return false
	}

	// 检查事件类型
	if len(params.EventTypes) > 0 {
		hasEvent := false
		for _, et := range params.EventTypes {
			if fileInfo.Events[et] > 0 {
				hasEvent = true
				break
			}
		}
		if !hasEvent {
			return false
		}
	}

	return true
}

// listApps 列出所有应用
func (r *Reader) listApps() ([]string, error) {
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var apps []string
	for _, entry := range entries {
		if entry.IsDir() {
			apps = append(apps, entry.Name())
		}
	}
	return apps, nil
}

// ListApps 获取所有应用ID列表
func (r *Reader) ListApps() ([]string, error) {
	return r.listApps()
}

// readFile 读取单个日志文件
func (r *Reader) readFile(filename string, params QueryParams) ([]*AuditLog, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []*AuditLog
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		var log AuditLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			fmt.Printf("Warning: failed to parse line %d: %v\n", lineNum, err)
			continue // 跳过无效的日志行
		}

		// 应用过滤条件
		if !r.matchFilters(&log, params) {
			continue
		}

		logs = append(logs, &log)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// checkTimeRange 检查时间范围
func (r *Reader) checkTimeRange(log *AuditLog, params QueryParams) bool {
	if params.StartTime != nil && log.Timestamp.Before(*params.StartTime) {
		return false
	}
	if params.EndTime != nil && log.Timestamp.After(*params.EndTime) {
		return false
	}
	return true
}

// checkEventType 检查事件类型
func (r *Reader) checkEventType(log *AuditLog, params QueryParams) bool {
	if len(params.EventTypes) == 0 {
		return true
	}
	for _, et := range params.EventTypes {
		if log.EventType == et {
			return true
		}
	}
	return false
}

// checkBasicFields 检查基本字段
func (r *Reader) checkBasicFields(log *AuditLog, params QueryParams) bool {
	// 检查用户ID
	if params.UserID != "" && log.UserID != params.UserID {
		return false
	}

	// 检查应用ID
	if params.AppID != "" && log.AppID != params.AppID {
		return false
	}

	// 检查客户端IP
	if params.ClientIP != "" && log.ClientIP != params.ClientIP {
		return false
	}

	// 检查状态码
	if params.StatusCode != 0 && log.StatusCode != params.StatusCode {
		return false
	}

	return true
}

// matchFilters 检查日志是否匹配过滤条件
func (r *Reader) matchFilters(log *AuditLog, params QueryParams) bool {
	// 按最可能过滤的条件顺序检查
	return r.checkTimeRange(log, params) &&
		r.checkEventType(log, params) &&
		r.checkBasicFields(log, params)
}

// VerifyLogFile 验证日志文件的完整性
func (r *Reader) VerifyLogFile(filename string) (bool, error) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		return false, fmt.Errorf("invalid file name")
	}
	file, err := os.Open(filepath.Join(r.baseDir, filename))
	if err != nil {
		return false, err
	}
	defer file.Close()

	var logs []*AuditLog
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var log AuditLog
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			return false, fmt.Errorf("invalid log format: %v", err)
		}
		logs = append(logs, &log)
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	// 验证哈希链
	hashChain := NewHashChain()
	return hashChain.VerifyChain(logs), nil
}

// GetStats 获取应用的统计信息
func (r *Reader) GetStats(appID string, startTime, endTime time.Time) (*AppLogStats, error) {
	index, err := r.loadIndex(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %v", err)
	}

	if index == nil {
		return &AppLogStats{
			EventStats: make(map[EventType]int),
		}, nil
	}

	stats, ok := index.AppStats[appID]
	if !ok {
		return &AppLogStats{
			EventStats: make(map[EventType]int),
		}, nil
	}

	// 返回统计信息的副本
	return &AppLogStats{
		TotalLogs:  stats.TotalLogs,
		EventStats: stats.EventStats,
		Size:       stats.Size,
	}, nil
}

// GetLogCount 获取应用的日志总数
func (r *Reader) GetLogCount(appID string) (int64, error) {
	// 加载应用索引
	index, err := r.loadIndex(appID)
	if err != nil {
		return 0, fmt.Errorf("failed to load index for app %s: %v", appID, err)
	}

	if index == nil {
		return 0, nil
	}

	stats, ok := index.AppStats[appID]
	if !ok {
		return 0, nil
	}

	return int64(stats.TotalLogs), nil
}
