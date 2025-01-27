package service

import "errors"

var (
	// ErrInvalidClient 无效的客户端
	ErrInvalidClient = errors.New("invalid client")
	// ErrInvalidRedirectURI 无效的重定向URI
	ErrInvalidRedirectURI = errors.New("invalid redirect uri")
	// ErrInvalidScope 无效的权限范围
	ErrInvalidScope = errors.New("invalid scope")
	// ErrUnsupportedGrantType 不支持的授权类型
	ErrUnsupportedGrantType = errors.New("unsupported grant type")
	// ErrInvalidGrant 无效的授权码
	ErrInvalidGrant = errors.New("invalid grant")
)
