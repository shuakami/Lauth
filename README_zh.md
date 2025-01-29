# LAuth

<p align="center">
  企业级统一认证平台，支持多应用管理
</p>

<p align="center">
  <a href="https://golang.org/doc/go1.19">
    <img src="https://img.shields.io/badge/go-1.19-blue.svg" alt="Go version"/>
  </a>
  <a href="https://www.gnu.org/licenses/agpl-3.0">
    <img src="https://img.shields.io/badge/License-AGPL%20v3-blue.svg" alt="License"/>
  </a>
  <a href="https://github.com/shuakami/Lauth/blob/master/README.md">
    <img src="https://img.shields.io/badge/English-blue.svg" alt="English"/>
  </a>
</p>

LAuth 是一个企业级统一认证平台，为多个应用提供集中式的认证服务。该平台注重性能、安全性和易用性。

## 功能特性

- **多应用支持**：在单一平台管理多个应用的认证需求
- **高性能**：使用 Go 语言构建，针对速度和资源利用进行优化
- **先进的权限系统**：
  - 基于角色的访问控制（RBAC）
  - 基于属性的访问控制（ABAC）
  - 动态规则引擎
  - 细粒度权限管理
  - 角色层级支持
- **OAuth 2.0 支持**：
  - 授权码模式
  - 客户端管理
  - 安全的令牌处理
  - 可自定义的权限范围
  - 令牌检查
  - 令牌撤销
- **OpenID Connect 支持**：
  - 完整的OAuth 2.0集成
  - ID令牌支持
  - 标准Claims
  - 多种响应类型（code、id_token、code id_token）
  - OIDC发现服务
  - JWKS端点
  - 用户信息端点
  - 标准OIDC参数（nonce、prompt、max_age等）
- **安全性设计**：
  - 基于 JWT 的认证机制
  - 令牌撤销功能
  - 密码加密存储
  - 可配置的安全策略
- **易于集成**：
  - RESTful API 接口
  - 完整的文档支持
  - 简单的 SDK（开发中）
- **企业级特性**：
  - 多租户架构
  - 审计日志
  - 可配置的认证流程
  - 高性能缓存
- **用户档案管理**：
  - 灵活的档案模式
  - 自定义字段支持
  - 基于MongoDB的档案存储
  - 与用户管理无缝集成

## 系统架构

### 权限系统

权限系统结合了 RBAC 和 ABAC 模型，提供灵活且强大的访问控制：

- **RBAC 核心**：
  - 角色管理
  - 权限分配
  - 用户-角色关联
  - 角色继承

- **规则引擎**：
  - 静态和动态规则
  - 丰富的操作符支持
  - 基于优先级的执行
  - 基于 Redis 的缓存
  - 实时验证

- **权限类型**：
  - 基于资源的权限
  - 基于操作的权限
  - 自定义属性规则

## 技术栈

- **开发语言**：Go 1.19+
- **数据库**：
  - PostgreSQL（核心数据）
  - MongoDB（档案数据）
- **缓存**：Redis
- **认证机制**：JWT
- **API框架**：基于 Gin 的 RESTful API
- **文档**：Swagger/OpenAPI

## 快速开始

### 环境要求

- Go 1.19 或更高版本
- PostgreSQL 12 或更高版本
- MongoDB 4.4 或更高版本
- Redis 6 或更高版本

### 安装步骤

1. 克隆仓库
```bash
git clone https://github.com/shuakami/Lauth.git
cd Lauth
```

2. 安装依赖
```bash
go mod download
```

3. 配置应用
```bash
cp config/config.example.yaml config/config.yaml
# 编辑 config.yaml 配置文件
```

4. 运行应用
```bash
go run main.go
```

## API 文档

### 认证接口

- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新访问令牌
- `POST /api/v1/auth/logout` - 用户登出
- `GET /api/v1/auth/validate` - 验证令牌
- `POST /api/v1/auth/validate-rule` - 组合验证令牌和规则并返回用户信息

### 应用管理

- `POST /api/v1/apps` - 创建应用
- `GET /api/v1/apps/:id` - 获取应用详情
- `PUT /api/v1/apps/:id` - 更新应用
- `DELETE /api/v1/apps/:id` - 删除应用
- `GET /api/v1/apps` - 应用列表

### 用户管理

- `POST /api/v1/apps/:id/users` - 创建用户
- `GET /api/v1/apps/:id/users/:user_id` - 获取用户详情（含档案）
- `PUT /api/v1/apps/:id/users/:user_id` - 更新用户
- `DELETE /api/v1/apps/:id/users/:user_id` - 删除用户
- `GET /api/v1/apps/:id/users` - 用户列表（含档案）
- `PUT /api/v1/apps/:id/users/:user_id/password` - 更新密码

### 档案管理

- `GET /api/v1/apps/:id/users/:user_id/profile` - 获取用户档案
- `PUT /api/v1/apps/:id/users/:user_id/profile` - 更新用户档案
- `DELETE /api/v1/apps/:id/users/:user_id/profile` - 删除用户档案
- `POST /api/v1/apps/:id/users/:user_id/profile/files` - 上传档案文件
- `GET /api/v1/apps/:id/users/:user_id/profile/files/:file_id` - 获取档案文件
- `DELETE /api/v1/apps/:id/users/:user_id/profile/files/:file_id` - 删除档案文件

### 角色管理

- `POST /api/v1/apps/:id/roles` - 创建角色
- `GET /api/v1/apps/:id/roles/:role_id` - 获取角色详情
- `PUT /api/v1/apps/:id/roles/:role_id` - 更新角色
- `DELETE /api/v1/apps/:id/roles/:role_id` - 删除角色
- `GET /api/v1/apps/:id/roles` - 角色列表
- `POST /api/v1/apps/:id/roles/:role_id/permissions` - 为角色添加权限
- `DELETE /api/v1/apps/:id/roles/:role_id/permissions` - 移除角色的权限
- `GET /api/v1/apps/:id/roles/:role_id/permissions` - 获取角色的权限列表
- `POST /api/v1/apps/:id/roles/:role_id/users` - 为角色添加用户
- `DELETE /api/v1/apps/:id/roles/:role_id/users` - 移除角色的用户
- `GET /api/v1/apps/:id/roles/:role_id/users` - 获取角色的用户列表

### 权限管理

- `POST /api/v1/apps/:id/permissions` - 创建权限
- `GET /api/v1/apps/:id/permissions/:permission_id` - 获取权限详情
- `PUT /api/v1/apps/:id/permissions/:permission_id` - 更新权限
- `DELETE /api/v1/apps/:id/permissions/:permission_id` - 删除权限
- `GET /api/v1/apps/:id/permissions` - 权限列表
- `GET /api/v1/apps/:id/permissions/resource/:type` - 按资源类型获取权限列表
- `GET /api/v1/apps/:id/users/:user_id/permissions` - 获取用户的权限列表

### 规则管理

- `POST /api/v1/apps/:id/rules` - 创建规则
- `GET /api/v1/apps/:id/rules/:rule_id` - 获取规则详情
- `PUT /api/v1/apps/:id/rules/:rule_id` - 更新规则
- `DELETE /api/v1/apps/:id/rules/:rule_id` - 删除规则
- `GET /api/v1/apps/:id/rules` - 规则列表
- `GET /api/v1/apps/:id/rules/active` - 获取活动规则列表
- `POST /api/v1/apps/:id/rules/validate` - 验证规则
- `POST /api/v1/apps/:id/rules/:rule_id/conditions` - 添加规则条件
- `PUT /api/v1/apps/:id/rules/:rule_id/conditions` - 更新规则条件
- `DELETE /api/v1/apps/:id/rules/:rule_id/conditions` - 删除规则条件
- `GET /api/v1/apps/:id/rules/:rule_id/conditions` - 获取规则条件

### OAuth 2.0 和 OpenID Connect

#### OAuth 2.0 端点
- `POST /api/v1/oauth/clients` - 创建OAuth客户端
- `GET /api/v1/oauth/clients/:client_id` - 获取OAuth客户端详情
- `PUT /api/v1/oauth/clients/:client_id` - 更新OAuth客户端
- `DELETE /api/v1/oauth/clients/:client_id` - 删除OAuth客户端
- `GET /api/v1/oauth/clients` - OAuth客户端列表
- `POST /api/v1/oauth/authorize` - 授权端点
- `POST /api/v1/oauth/token` - 令牌端点
- `POST /api/v1/oauth/revoke` - 令牌撤销端点
- `POST /api/v1/oauth/introspect` - 令牌检查端点

#### OpenID Connect 端点
- `GET /.well-known/openid-configuration` - OIDC发现端点
- `GET /.well-known/jwks.json` - JWKS端点
- `GET /api/v1/userinfo` - 用户信息端点
- `GET /api/v1/users/me` - 获取当前用户信息

## 配置说明

LAuth 支持通过环境变量或配置文件进行配置。配置文件位于 `config/config.yaml`。

主要配置项：
- 服务器端口和模式
- 数据库连接
- Redis 连接
- JWT 设置
- OIDC 设置（颁发者、密钥）
- 认证选项
- 权限系统设置
- 规则引擎配置

## 开发路线

- [x] 基于角色的访问控制（RBAC）
- [x] 基于属性的访问控制（ABAC）
- [x] 规则引擎
- [x] OAuth2.0 支持（授权码模式）
- [x] OAuth2.0 令牌端点
- [x] OpenID Connect 支持
- [ ] OAuth2.0 其他授权模式
- [ ] 多因素认证
- [ ] SDK 开发
- [ ] Docker 支持
- [ ] Kubernetes 部署指南

## 许可证

本项目采用 AGPL-3.0 许可证。 