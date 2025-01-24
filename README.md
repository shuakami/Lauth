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
</p>

LAuth is an enterprise-grade unified authentication platform that provides centralized authentication services for multiple applications. Built with performance, security, and ease of use in mind.

## Features

- **Multi-Application Support**: Manage authentication for multiple applications from a single platform
- **High Performance**: Built with Go, optimized for speed and resource efficiency
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
  - Role-based access control (coming soon)
  - Audit logging
  - Configurable authentication flows

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

### Application Management

- `POST /api/v1/apps` - Create application
- `GET /api/v1/apps/:id` - Get application details
- `PUT /api/v1/apps/:id` - Update application
- `DELETE /api/v1/apps/:id` - Delete application
- `GET /api/v1/apps` - List applications

### User Management

- `POST /api/v1/apps/:app_id/users` - Create user
- `GET /api/v1/apps/:app_id/users/:id` - Get user details
- `PUT /api/v1/apps/:app_id/users/:id` - Update user
- `DELETE /api/v1/apps/:app_id/users/:id` - Delete user
- `GET /api/v1/apps/:app_id/users` - List users

## Configuration

LAuth can be configured via environment variables or configuration file. The configuration file is located at `config/config.yaml`.

Key configuration options:
- Server port and mode
- Database connection
- Redis connection
- JWT settings
- Authentication options

## Roadmap

- [ ] Role-based access control (RBAC)
- [ ] OAuth2.0 support
- [ ] OpenID Connect support
- [ ] Multi-factor authentication
- [ ] Audit logging
- [ ] SDK development
- [ ] Docker support
- [ ] Kubernetes deployment guides

## License

This project is licensed under the AGPL-3.0 License. 