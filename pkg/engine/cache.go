package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/pkg/redis"
)

const (
	// ruleKeyPrefix Redis键前缀
	ruleKeyPrefix = "rules:"
	// defaultExpiration 默认过期时间
	defaultExpiration = 24 * time.Hour
)

// cache 规则缓存实现
type cache struct {
	client *redis.Client
}

// NewCache 创建规则缓存实例
func NewCache(client *redis.Client) Cache {
	return &cache{client: client}
}

// Get 从缓存获取规则
func (c *cache) Get(ctx context.Context, appID string) ([]*model.Rule, error) {
	key := c.buildKey(appID)
	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var rules []*model.Rule
	if err := json.Unmarshal([]byte(data), &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %v", err)
	}

	return rules, nil
}

// Set 将规则存入缓存
func (c *cache) Set(ctx context.Context, appID string, rules []*model.Rule, expiration time.Duration) error {
	key := c.buildKey(appID)
	data, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %v", err)
	}

	if expiration == 0 {
		expiration = defaultExpiration
	}

	return c.client.Set(ctx, key, string(data), expiration)
}

// Delete 从缓存删除规则
func (c *cache) Delete(ctx context.Context, appID string) error {
	key := c.buildKey(appID)
	return c.client.Del(ctx, key)
}

// buildKey 构建缓存键
func (c *cache) buildKey(appID string) string {
	return fmt.Sprintf("%s%s", ruleKeyPrefix, appID)
}
