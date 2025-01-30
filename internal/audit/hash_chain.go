package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

// HashChain 哈希链结构
type HashChain struct {
	mu       sync.RWMutex
	lastHash string
}

// NewHashChain 创建新的哈希链
func NewHashChain() *HashChain {
	return &HashChain{
		lastHash: "", // 初始哈希为空
	}
}

// calculateHash 计算日志的哈希值
func (hc *HashChain) calculateHash(log *AuditLog) string {
	// 创建日志的副本，清除哈希字段
	logCopy := *log
	logCopy.Hash = ""

	// 将日志序列化为JSON
	data, err := json.Marshal(logCopy)
	if err != nil {
		return ""
	}

	// 计算SHA256哈希
	hasher := sha256.New()
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))

	return hash
}

// AddLog 添加日志到哈希链
func (hc *HashChain) AddLog(log *AuditLog) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// 设置前一个哈希
	log.PrevHash = hc.lastHash

	// 计算当前日志的哈希
	hash := hc.calculateHash(log)
	if hash == "" {
		return fmt.Errorf("failed to calculate hash")
	}

	// 设置日志哈希
	log.Hash = hash

	// 更新最后一个哈希
	hc.lastHash = hash

	return nil
}

// VerifyLog 验证单条日志的完整性
func (hc *HashChain) VerifyLog(log *AuditLog) bool {
	// 计算日志的哈希
	calculatedHash := hc.calculateHash(log)
	if calculatedHash == "" {
		return false
	}

	// 验证哈希是否匹配
	return calculatedHash == log.Hash
}

// VerifyChain 验证日志链的完整性
func (hc *HashChain) VerifyChain(logs []*AuditLog) bool {
	if len(logs) == 0 {
		return true
	}

	// 验证第一条日志
	if !hc.VerifyLog(logs[0]) {
		return false
	}

	// 验证后续日志的哈希链接
	for i := 1; i < len(logs); i++ {
		if !hc.VerifyLog(logs[i]) {
			return false
		}
		if logs[i].PrevHash != logs[i-1].Hash {
			return false
		}
	}

	return true
}

// GetLastHash 获取最后一个哈希值
func (hc *HashChain) GetLastHash() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastHash
}
