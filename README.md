# LAuth

<p align="center">
  Enterprise-grade unified authentication platform with multi-application support.
</p>

<p align="center">
  <a href="https://golang.org/doc/go1.19">
    <img src="https://img.shields.io/badge/go-1.19-blue.svg" alt="Go version"/>
  </a>
  <a href="https://www.gnu.org/licenses/agpl-3.0">
    <img src="https://img.shields.io/badge/License-AGPL%20v3-blue.svg" alt="License"/>
  </a>
  <a href="https://github.com/shuakami/Lauth/blob/master/README_zh.md">
    <img src="https://img.shields.io/badge/简体中文-blue.svg" alt="简体中文"/>
  </a>
</p>

LAuth is an enterprise-grade unified authentication platform that provides centralized authentication services for multiple applications. Built with performance, security, and ease of use in mind.

## Features

- **Multi-Application Support**: Manage authentication for multiple applications from a single platform
- **High Performance**: Built with Go, optimized for speed and resource efficiency
- **Advanced Permission System**:
  - Role-Based Access Control (RBAC)
  - Attribute-Based Access Control (ABAC)
  - Dynamic Rules Engine
  - Fine-grained Permission Management
  - Role Hierarchy Support
- **OAuth 2.0 Support**:
  - Authorization Code Grant
  - Client Management
  - Secure Token Handling
  - Customizable Scopes
  - Token Introspection
  - Token Revocation
- **OpenID Connect Support**:
  - Full OAuth 2.0 Integration
  - ID Token Support
  - Standard Claims
  - Multiple Response Types (code, id_token, code id_token)
  - OIDC Discovery Service
  - JWKS Endpoint
  - User Info Endpoint
  - Standard OIDC Parameters (nonce, prompt, max_age, etc.)
- **Secure by Design**: 
  - JWT-based authentication
  - Token revocation
  - Password encryption
  - Configurable security policies
  - Device recognition
  - Login location tracking
  - IP-based security rules
- **Easy Integration**: 
  - RESTful API
  - Comprehensive documentation
  - Simple SDK (coming soon)
- **Enterprise Ready**:
  - Multi-tenant architecture
  - Audit logging with integrity verification
  - Real-time audit log streaming via WebSocket
  - Configurable authentication flows
  - High-performance caching
  - IP geolocation service
  - Event type strategy
  - Login location history
- **Plugin System**:
  - Flexible verification plugins
  - Email verification support
    - Verification code mode
    - Verification link mode
    - Dark mode support
    - Responsive email templates
  - TOTP (Time-based One-Time Password) support
    - QR code generation
    - Configurable settings (period, digits, etc)
    - Setup, verification and disable flows
  - Extensible plugin architecture
  - Plugin lifecycle management
  - Real-time plugin status tracking
  - Exemption rules support
  - User configuration management
  - Verification record tracking
  - Plugin route registration
  - Smart plugin interface
  - Plugin dependency injection
  - Middleware support
  - Enhanced error handling
  - Event emission capability
  - Temporary session support
  - Verification status cleanup
  - Plugin status caching
  - Unified verification context
  - Automatic plugin status tracking
  - Smart verification flow
  - Registration-specific rules
  - Dynamic plugin discovery
  - Automatic plugin registration
  - Optional verification sessions
  - Standardized API responses
- **User Profile Management**:
  - Flexible profile schema
  - Custom fields support
  - Profile data storage in MongoDB
  - Seamless integration with user management

## System Architecture

### Permission System

The permission system combines RBAC and ABAC models to provide flexible and powerful access control:

- **RBAC Core**:
  - Role management
  - Permission assignment
  - User-role association
  - Role inheritance

- **Rules Engine**:
  - Static and dynamic rules
  - Rich operator support
  - Priority-based execution
  - Redis-based caching
  - Real-time validation

- **Permission Types**:
  - Resource-based permissions
  - Operation-based permissions
  - Custom attribute rules

## Tech Stack

- **Language**: Go 1.19+
- **Database**: 
  - PostgreSQL (Core data)
  - MongoDB (Profile data)
- **Cache**: Redis
- **Authentication**: JWT
- **API**: RESTful with Gin framework
- **Documentation**: Swagger/OpenAPI

## Quick Start

### Prerequisites

- Go 1.19 or higher
- PostgreSQL 12 or higher
- MongoDB 4.4 or higher
- Redis 6 or higher

### Installation

1. Clone the repository
```bash
git clone https://github.com/shuakami/Lauth.git
cd Lauth
```

2. Install dependencies
```bash
go mod download
```

3. Configure the application
```bash
cp config/config.example.yaml config/config.yaml
# Edit config.yaml with your settings
```

4. Run the application
```bash
go run main.go
```

## API Documentation

### Authentication Endpoints

- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - User logout
- `GET /api/v1/auth/validate` - Validate token
- `POST /api/v1/auth/validate-rule` - Combined validation for token and rules with user info

### Login Location

- `GET /api/v1/apps/:id/users/:user_id/login-locations` - Get user login locations
- `GET /api/v1/apps/:id/users/:user_id/login-locations/:location_id` - Get login location details
- `GET /api/v1/apps/:id/users/:user_id/login-locations/stats` - Get login location statistics

### Application Management

- `POST /api/v1/apps` - Create application
- `GET /api/v1/apps/:id` - Get application details
- `PUT /api/v1/apps/:id` - Update application
- `DELETE /api/v1/apps/:id` - Delete application
- `GET /api/v1/apps` - List applications

### User Management

- `POST /api/v1/apps/:id/users` - Create user
- `GET /api/v1/apps/:id/users/:user_id` - Get user details with profile
- `PUT /api/v1/apps/:id/users/:user_id` - Update user
- `DELETE /api/v1/apps/:id/users/:user_id` - Delete user
- `GET /api/v1/apps/:id/users` - List users with profiles
- `PUT /api/v1/apps/:id/users/:user_id/password` - Update password

### Profile Management

- `GET /api/v1/apps/:id/users/:user_id/profile` - Get user profile
- `PUT /api/v1/apps/:id/users/:user_id/profile` - Update user profile
- `DELETE /api/v1/apps/:id/users/:user_id/profile` - Delete user profile
- `POST /api/v1/apps/:id/users/:user_id/profile/files` - Upload profile files
- `GET /api/v1/apps/:id/users/:user_id/profile/files/:file_id` - Get profile file
- `DELETE /api/v1/apps/:id/users/:user_id/profile/files/:file_id` - Delete profile file

### Role Management

- `POST /api/v1/apps/:id/roles` - Create role
- `GET /api/v1/apps/:id/roles/:role_id` - Get role details
- `PUT /api/v1/apps/:id/roles/:role_id` - Update role
- `DELETE /api/v1/apps/:id/roles/:role_id` - Delete role
- `GET /api/v1/apps/:id/roles` - List roles
- `POST /api/v1/apps/:id/roles/:role_id/permissions` - Add permissions to role
- `DELETE /api/v1/apps/:id/roles/:role_id/permissions` - Remove permissions from role
- `GET /api/v1/apps/:id/roles/:role_id/permissions` - Get role permissions
- `POST /api/v1/apps/:id/roles/:role_id/users` - Add users to role
- `DELETE /api/v1/apps/:id/roles/:role_id/users` - Remove users from role
- `GET /api/v1/apps/:id/roles/:role_id/users` - Get role users

### Permission Management

- `POST /api/v1/apps/:id/permissions` - Create permission
- `GET /api/v1/apps/:id/permissions/:permission_id` - Get permission details
- `PUT /api/v1/apps/:id/permissions/:permission_id` - Update permission
- `DELETE /api/v1/apps/:id/permissions/:permission_id` - Delete permission
- `GET /api/v1/apps/:id/permissions` - List permissions
- `GET /api/v1/apps/:id/permissions/resource/:type` - List permissions by resource type
- `GET /api/v1/apps/:id/users/:user_id/permissions` - List user permissions

### Rules Management

- `POST /api/v1/apps/:id/rules` - Create rule
- `GET /api/v1/apps/:id/rules/:rule_id` - Get rule details
- `PUT /api/v1/apps/:id/rules/:rule_id` - Update rule
- `DELETE /api/v1/apps/:id/rules/:rule_id` - Delete rule
- `GET /api/v1/apps/:id/rules` - List rules
- `GET /api/v1/apps/:id/rules/active` - List active rules
- `POST /api/v1/apps/:id/rules/validate` - Validate rules
- `POST /api/v1/apps/:id/rules/:rule_id/conditions` - Add rule conditions
- `PUT /api/v1/apps/:id/rules/:rule_id/conditions` - Update rule conditions
- `DELETE /api/v1/apps/:id/rules/:rule_id/conditions` - Remove rule conditions
- `GET /api/v1/apps/:id/rules/:rule_id/conditions` - Get rule conditions

### Plugin Management

- `POST /api/v1/apps/:id/plugins/install` - Install plugin
- `POST /api/v1/apps/:id/plugins/uninstall/:name` - Uninstall plugin
- `POST /api/v1/apps/:id/plugins/:name/execute` - Execute plugin
- `GET /api/v1/apps/:id/plugins/list` - List installed plugins
- `GET /api/v1/apps/:id/plugins/all` - List all registered plugins
- `PUT /api/v1/apps/:id/plugins/:name/config` - Update plugin config

### OAuth 2.0 and OpenID Connect

#### OAuth 2.0 Endpoints
- `POST /api/v1/oauth/clients` - Create OAuth client
- `GET /api/v1/oauth/clients/:client_id` - Get OAuth client details
- `PUT /api/v1/oauth/clients/:client_id` - Update OAuth client
- `DELETE /api/v1/oauth/clients/:client_id` - Delete OAuth client
- `GET /api/v1/oauth/clients` - List OAuth clients
- `POST /api/v1/oauth/authorize` - Authorization endpoint
- `POST /api/v1/oauth/token` - Token endpoint
- `POST /api/v1/oauth/revoke` - Token revocation endpoint
- `POST /api/v1/oauth/introspect` - Token introspection endpoint

#### OpenID Connect Endpoints
- `GET /.well-known/openid-configuration` - OIDC discovery endpoint
- `GET /.well-known/jwks.json` - JWKS endpoint
- `GET /api/v1/userinfo` - UserInfo endpoint
- `GET /api/v1/users/me` - Get current user info

### Audit Logging

- `GET /api/v1/audit/logs` - Query audit logs
- `GET /api/v1/audit/logs/verify` - Verify log file integrity
- `GET /api/v1/audit/stats` - Get audit statistics
- `GET /api/v1/audit/ws` - WebSocket connection for real-time logs

## Configuration

LAuth can be configured via environment variables or configuration file. The configuration file is located at `config/config.yaml`.

Key configuration options:
- Server port and mode
- Database connection
- Redis connection
- JWT settings
- OIDC settings (issuer, keys)
- Authentication options
- Permission system settings
- Rules engine configuration
- Plugin system settings (plugins directory, configurations)

## Roadmap

- [x] Role-based access control (RBAC)
- [x] Attribute-based access control (ABAC)
- [x] Rules engine
- [x] OAuth2.0 support (Authorization Code Grant)
- [x] OAuth2.0 Token endpoint
- [x] OpenID Connect support
- [ ] OAuth2.0 additional grant types
- [x] Multi-factor authentication
- [ ] SDK development
- [ ] Docker support
- [ ] Kubernetes deployment guides

## License

This project is licensed under the AGPL-3.0 License. 