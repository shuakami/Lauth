package types

// BasePlugin 基础插件结构
type BasePlugin struct {
	locationService LocationService
}

// SetLocationService 设置位置服务
func (p *BasePlugin) SetLocationService(svc LocationService) {
	p.locationService = svc
}

// GetLocationService 获取位置服务
func (p *BasePlugin) GetLocationService() LocationService {
	return p.locationService
}
