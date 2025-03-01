package plugin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// pluginRouter 插件路由注册器
// 用于根据路径决定使用哪个路由组（需要认证或不需要认证）
type pluginRouter struct {
	pluginGroup *gin.RouterGroup // 不需要认证的路由组
	authGroup   *gin.RouterGroup // 需要认证的路由组
	authRoutes  []string         // 需要认证的路由列表
}

// AsRouterGroup 将pluginRouter转换为*gin.RouterGroup
// 这个方法用于兼容Routable接口的RegisterRoutes方法
func (r *pluginRouter) AsRouterGroup() *gin.RouterGroup {
	return r.pluginGroup
}

// Group 实现gin.RouterGroup的Group方法
func (r *pluginRouter) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	// 检查路径是否需要认证
	needsAuth := isAuthNeeded(relativePath, r.authRoutes)
	if needsAuth {
		return r.authGroup.Group(relativePath, handlers...)
	}
	return r.pluginGroup.Group(relativePath, handlers...)
}

// Handle 实现gin.RouterGroup的Handle方法
func (r *pluginRouter) Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	needsAuth := isAuthNeeded(relativePath, r.authRoutes)
	if needsAuth {
		return r.authGroup.Handle(httpMethod, relativePath, handlers...)
	}
	return r.pluginGroup.Handle(httpMethod, relativePath, handlers...)
}

// POST 是router.Handle("POST", path, handle)的快捷方式
func (r *pluginRouter) POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodPost, relativePath, handlers...)
}

// GET 是router.Handle("GET", path, handle)的快捷方式
func (r *pluginRouter) GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodGet, relativePath, handlers...)
}

// DELETE 是router.Handle("DELETE", path, handle)的快捷方式
func (r *pluginRouter) DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodDelete, relativePath, handlers...)
}

// PATCH 是router.Handle("PATCH", path, handle)的快捷方式
func (r *pluginRouter) PATCH(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodPatch, relativePath, handlers...)
}

// PUT 是router.Handle("PUT", path, handle)的快捷方式
func (r *pluginRouter) PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodPut, relativePath, handlers...)
}

// OPTIONS 是router.Handle("OPTIONS", path, handle)的快捷方式
func (r *pluginRouter) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodOptions, relativePath, handlers...)
}

// HEAD 是router.Handle("HEAD", path, handle)的快捷方式
func (r *pluginRouter) HEAD(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return r.Handle(http.MethodHead, relativePath, handlers...)
}

// Any 注册一个匹配所有HTTP方法的路由
func (r *pluginRouter) Any(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	needsAuth := isAuthNeeded(relativePath, r.authRoutes)
	if needsAuth {
		return r.authGroup.Any(relativePath, handlers...)
	}
	return r.pluginGroup.Any(relativePath, handlers...)
}

// isAuthNeeded 用于判断给定的路由 path 是否需要认证
func isAuthNeeded(relativePath string, authRoutes []string) bool {
	for _, route := range authRoutes {
		if route == "*" { // 特殊情况：所有路由都需要认证
			return true
		}
		// 比较时要考虑相对路径前缀
		// 如果 authRoutes 中定义的 "xxx" 等于 relativePath 或者二者只差一个斜杠
		if route == relativePath || (len(relativePath) > 0 && relativePath[0] == '/' && route == relativePath[1:]) {
			return true
		}
	}
	return false
}
