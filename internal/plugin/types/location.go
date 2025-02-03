package types

import (
	"context"
	"time"
)

// LocationResponse 位置信息响应
type LocationResponse struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	ISP       string    `json:"isp"`
	LoginTime time.Time `json:"login_time"`
}

// LocationService 位置服务接口
type LocationService interface {
	// 记录登录位置
	RecordLoginLocation(ctx context.Context, appID, userID, ip string) error
	// 获取用户最近的登录位置
	GetLatestLocations(ctx context.Context, userID string, limit int) ([]*LocationResponse, error)
	// 获取用户指定时间范围内的登录位置
	GetLocationsByTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*LocationResponse, error)
}
