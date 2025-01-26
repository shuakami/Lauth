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
- **Secure by Design**: 
  - JWT-based authentication
  - Token revocation
  - Password encryption
  - Configurable security policies
- **Easy Integration**: 
  - RESTful API
  - Comprehensive documentation
  - Simple SDK (coming soon)
- **Enterprise Ready**:
  - Multi-tenant architecture
  - Audit logging
  - Configurable authentication flows
  - High-performance caching

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
- **Database**: PostgreSQL
- **Cache**: Redis
- **Authentication**: JWT
- **API**: RESTful with Gin framework
- **Documentation**: Swagger/OpenAPI

## Quick Start

### Prerequisites

- Go 1.19 or higher
- PostgreSQL 12 or higher
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

### Application Management

- `POST /api/v1/apps` - Create application
- `GET /api/v1/apps/:id` - Get application details
- `PUT /api/v1/apps/:id` - Update application
- `DELETE /api/v1/apps/:id` - Delete application
- `GET /api/v1/apps` - List applications

### User Management

- `POST /api/v1/apps/:id/users` - Create user
- `GET /api/v1/apps/:id/users/:user_id` - Get user details
- `PUT /api/v1/apps/:id/users/:user_id` - Update user
- `DELETE /api/v1/apps/:id/users/:user_id` - Delete user
- `GET /api/v1/apps/:id/users` - List users
- `PUT /api/v1/apps/:id/users/:user_id/password` - Update password

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

### OAuth 2.0 Management

- `POST /api/v1/apps/:id/oauth/clients` - Create OAuth client
- `GET /api/v1/apps/:id/oauth/clients/:client_id` - Get OAuth client details
- `PUT /api/v1/apps/:id/oauth/clients/:client_id` - Update OAuth client
- `DELETE /api/v1/apps/:id/oauth/clients/:client_id` - Delete OAuth client
- `GET /api/v1/apps/:id/oauth/clients` - List OAuth clients
- `POST /api/v1/apps/:id/oauth/clients/:client_id/secrets` - Create client secret
- `GET /api/v1/apps/:id/oauth/clients/:client_id/secrets` - List client secrets
- `DELETE /api/v1/apps/:id/oauth/clients/:client_id/secrets/:secret_id` - Delete client secret

### OAuth 2.0 Authorization

- `GET /api/v1/oauth/authorize` - OAuth authorization endpoint
- `POST /api/v1/oauth/token` - Token endpoint (coming soon)

## Configuration

LAuth can be configured via environment variables or configuration file. The configuration file is located at `config/config.yaml`.

Key configuration options:
- Server port and mode
- Database connection
- Redis connection
- JWT settings
- Authentication options
- Permission system settings
- Rules engine configuration

## Roadmap

- [x] Role-based access control (RBAC)
- [x] Attribute-based access control (ABAC)
- [x] Rules engine
- [x] OAuth2.0 support (Authorization Code Grant)
- [ ] OAuth2.0 additional grant types
- [ ] OpenID Connect support
- [ ] Multi-factor authentication
- [ ] SDK development
- [ ] Docker support
- [ ] Kubernetes deployment guides

## License

This project is licensed under the AGPL-3.0 License. 