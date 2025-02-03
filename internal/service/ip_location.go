package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// IPLocation IP地理位置信息
type IPLocation struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	ISP      string `json:"isp"`
}

// IPLocationService IP地理位置服务接口
type IPLocationService interface {
	SearchIP(ctx context.Context, ip string) (*IPLocation, error)
}

// ipLocationService IP地理位置服务实现
type ipLocationService struct {
	searcher  *xdb.Searcher
	dbPath    string
	initOnce  sync.Once
	initError error
}

// NewIPLocationService 创建IP地理位置服务实例
func NewIPLocationService(dbPath string) IPLocationService {
	return &ipLocationService{
		dbPath: dbPath,
	}
}

// initSearcher 初始化IP搜索器
func (s *ipLocationService) initSearcher() error {
	s.initOnce.Do(func() {
		// 读取整个数据库文件到内存
		content, err := os.ReadFile(s.dbPath)
		if err != nil {
			s.initError = fmt.Errorf("failed to read database file: %v", err)
			return
		}

		// 创建内存搜索器
		searcher, err := xdb.NewWithBuffer(content)
		if err != nil {
			s.initError = fmt.Errorf("failed to create ip searcher: %v", err)
			return
		}
		s.searcher = searcher
	})
	return s.initError
}

// SearchIP 搜索IP地理位置
func (s *ipLocationService) SearchIP(ctx context.Context, ip string) (*IPLocation, error) {
	// 初始化搜索器
	if err := s.initSearcher(); err != nil {
		return nil, err
	}

	// 搜索IP
	region, err := s.searcher.SearchByStr(ip)
	if err != nil {
		log.Printf("Failed to search ip location for %s: %v", ip, err)
		return nil, err
	}

	// 解析地理位置信息
	// ip2region 返回格式: "中国|0|江苏省|南京市|电信"
	parts := strings.Split(region, "|")
	location := &IPLocation{}

	if len(parts) >= 5 {
		location.Country = parts[0]
		if parts[2] != "0" {
			location.Province = parts[2]
		}
		if parts[3] != "0" {
			location.City = parts[3]
		}
		if parts[4] != "0" {
			location.ISP = parts[4]
		}
	}

	return location, nil
}
