package model

import (
	"time"

	"encoding/base64"
	"math/rand"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// OAuthClientType 客户端类型
type OAuthClientType string

const (
	// Confidential 机密客户端 - 能够安全存储密钥的客户端(如服务器端应用)
	Confidential OAuthClientType = "confidential"
	// Public 公开客户端 - 不能安全存储密钥的客户端(如浏览器端或移动应用)
	Public OAuthClientType = "public"
)

// OAuthGrantType 授权类型
type OAuthGrantType string

const (
	AuthorizationCodeGrant OAuthGrantType = "authorization_code"
	ClientCredentials      OAuthGrantType = "client_credentials"
	Password               OAuthGrantType = "password"
	Implicit               OAuthGrantType = "implicit"
	RefreshTokenGrant      OAuthGrantType = "refresh_token"
)

// ResponseType 响应类型
type ResponseType string

const (
	CodeResponse        ResponseType = "code"
	IDTokenResponse     ResponseType = "id_token"
	CodeIDTokenResponse ResponseType = "code id_token"
)

// OAuthClient OAuth客户端
type OAuthClient struct {
	ID           string          `json:"id" gorm:"primaryKey;type:uuid"`
	AppID        string          `json:"app_id" gorm:"index;type:uuid"`
	Name         string          `json:"name" gorm:"type:varchar(100)"`
	ClientID     string          `json:"client_id" gorm:"type:varchar(100);uniqueIndex"`
	ClientSecret string          `json:"-" gorm:"type:varchar(100)"`
	Type         OAuthClientType `json:"type" gorm:"type:varchar(20)"`
	GrantTypes   pq.StringArray  `json:"grant_types" gorm:"type:text[]"`
	RedirectURIs pq.StringArray  `json:"redirect_uris" gorm:"type:text[]"`
	Scopes       pq.StringArray  `json:"scopes" gorm:"type:text[]"`
	Status       bool            `json:"status" gorm:"default:true"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID和客户端凭证
func (c *OAuthClient) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.ClientID == "" {
		c.ClientID = uuid.New().String()
	}
	if c.ClientSecret == "" {
		c.ClientSecret = uuid.NewString()
	}
	return nil
}

// TableName 指定表名
func (OAuthClient) TableName() string {
	return "oauth_clients"
}

// CreateOAuthClientRequest 创建OAuth客户端请求
type CreateOAuthClientRequest struct {
	Name         string          `json:"name" binding:"required"`
	Type         OAuthClientType `json:"type" binding:"required,oneof=confidential public"`
	GrantTypes   []string        `json:"grant_types" binding:"required,dive,oneof=authorization_code client_credentials password implicit refresh_token"`
	RedirectURIs []string        `json:"redirect_uris" binding:"omitempty,required_unless=Type public,dive,url"`
	Scopes       []string        `json:"scopes" binding:"required"`
}

// UpdateOAuthClientRequest 更新OAuth客户端请求
type UpdateOAuthClientRequest struct {
	Name         string   `json:"name"`
	GrantTypes   []string `json:"grant_types" binding:"omitempty,dive,oneof=authorization_code client_credentials password implicit refresh_token"`
	RedirectURIs []string `json:"redirect_uris" binding:"omitempty,dive,url"`
	Scopes       []string `json:"scopes"`
	Status       *bool    `json:"status"`
}

// OAuthClientResponse OAuth客户端响应
type OAuthClientResponse struct {
	ID           string          `json:"id"`
	AppID        string          `json:"app_id"`
	Name         string          `json:"name"`
	ClientID     string          `json:"client_id"`
	Type         OAuthClientType `json:"type"`
	GrantTypes   []string        `json:"grant_types"`
	RedirectURIs []string        `json:"redirect_uris"`
	Scopes       []string        `json:"scopes"`
	Status       bool            `json:"status"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// OAuthClientSecret OAuth客户端秘钥
type OAuthClientSecret struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	ClientID    string    `json:"client_id" gorm:"index;type:varchar(100)"`
	Secret      string    `json:"secret" gorm:"type:varchar(100)"`
	Description string    `json:"description" gorm:"type:varchar(200)"`
	LastUsedAt  time.Time `json:"last_used_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名
func (OAuthClientSecret) TableName() string {
	return "oauth_client_secrets"
}

// CreateClientSecretRequest 创建客户端秘钥请求
type CreateClientSecretRequest struct {
	Description string `json:"description" binding:"required"`
	ExpiresIn   int64  `json:"expires_in" binding:"required,min=3600"` // 最小1小时
}

// ClientSecretResponse 客户端秘钥响应
type ClientSecretResponse struct {
	ID          string `json:"id"`
	Secret      string `json:"secret,omitempty"` // 仅在创建时返回
	Description string `json:"description"`
	LastUsedAt  string `json:"last_used_at"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

// AuthorizationRequest OAuth授权请求
type AuthorizationRequest struct {
	ResponseType ResponseType `json:"response_type" form:"response_type" binding:"required,oneof=code 'id_token' 'code id_token'"` // 响应类型
	ClientID     string       `json:"client_id" form:"client_id" binding:"required"`                                               // 客户端ID
	RedirectURI  string       `json:"redirect_uri" form:"redirect_uri" binding:"required,url"`                                     // 重定向URI
	Scope        string       `json:"scope" form:"scope" binding:"required"`                                                       // 申请的权限范围
	State        string       `json:"state" form:"state" binding:"required"`                                                       // 状态参数

	// OIDC特定参数
	Nonce       string `json:"nonce" form:"nonce"`                 // OIDC nonce参数
	Display     string `json:"display" form:"display"`             // 显示类型(page, popup, touch, wap)
	Prompt      string `json:"prompt" form:"prompt"`               // 提示类型(none, login, consent, select_account)
	MaxAge      int    `json:"max_age" form:"max_age"`             // 最大认证时间(秒)
	UILocales   string `json:"ui_locales" form:"ui_locales"`       // UI语言偏好
	IDTokenHint string `json:"id_token_hint" form:"id_token_hint"` // 之前颁发的ID Token
	LoginHint   string `json:"login_hint" form:"login_hint"`       // 登录提示
	ACRValues   string `json:"acr_values" form:"acr_values"`       // 请求的认证上下文类型
}

// AuthorizationCode OAuth授权码
type AuthorizationCode struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	Code        string    `json:"code" gorm:"type:varchar(100);uniqueIndex"` // 授权码
	ClientID    string    `json:"client_id" gorm:"type:varchar(100);index"`  // 客户端ID
	UserID      string    `json:"user_id" gorm:"type:varchar(100);index"`    // 用户ID
	RedirectURI string    `json:"redirect_uri" gorm:"type:varchar(500)"`     // 重定向URI
	Scope       string    `json:"scope" gorm:"type:varchar(500)"`            // 授权范围
	ExpiresAt   time.Time `json:"expires_at"`                                // 过期时间
	CreatedAt   time.Time `json:"created_at"`                                // 创建时间
}

// TableName 指定表名
func (AuthorizationCode) TableName() string {
	return "oauth_authorization_codes"
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID和授权码
func (ac *AuthorizationCode) BeforeCreate(tx *gorm.DB) error {
	if ac.ID == "" {
		ac.ID = uuid.New().String()
	}
	if ac.Code == "" {
		// 生成32字节的随机授权码
		codeBytes := make([]byte, 32)
		if _, err := rand.Read(codeBytes); err != nil {
			return err
		}
		ac.Code = base64.URLEncoding.EncodeToString(codeBytes)
	}
	return nil
}

// TokenRequest OAuth令牌请求
type TokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code"`
	RedirectURI  string `form:"redirect_uri"`
	ClientID     string `form:"client_id" binding:"required"`
	ClientSecret string `form:"client_secret" binding:"required"`
	RefreshToken string `form:"refresh_token"`

	// OIDC特定参数
	Nonce string `form:"nonce"` // OIDC nonce参数
}

// TokenResponse OAuth令牌响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`            // 访问令牌
	TokenType    string `json:"token_type"`              // 令牌类型
	ExpiresIn    int64  `json:"expires_in"`              // 过期时间(秒)
	RefreshToken string `json:"refresh_token,omitempty"` // 刷新令牌
	Scope        string `json:"scope,omitempty"`         // 权限范围

	// OIDC特定响应
	IDToken string `json:"id_token,omitempty"` // ID令牌(仅在scope包含openid时返回)
}

// TokenError OAuth令牌错误响应
type TokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

const (
	// GrantTypeAuthorizationCode 授权码授权类型
	GrantTypeAuthorizationCode = "authorization_code"
	// GrantTypeRefreshToken 刷新令牌授权类型
	GrantTypeRefreshToken = "refresh_token"

	// 错误类型
	ErrorInvalidRequest       = "invalid_request"
	ErrorInvalidClient        = "invalid_client"
	ErrorInvalidGrant         = "invalid_grant"
	ErrorUnauthorizedClient   = "unauthorized_client"
	ErrorUnsupportedGrantType = "unsupported_grant_type"
	ErrorInvalidScope         = "invalid_scope"
)

// OIDC相关常量
const (
	// OIDC标准scope
	ScopeOpenID  = "openid"
	ScopeProfile = "profile"
	ScopeEmail   = "email"
	ScopePhone   = "phone"
	ScopeAddress = "address"

	// OIDC标准响应类型
	ResponseTypeIDToken     = "id_token"
	ResponseTypeIDTokenCode = "code id_token"
)

// OIDCClaims OpenID Connect标准Claims
type OIDCClaims struct {
	// 必需Claims
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`

	// 认证相关Claims
	AuthTime int64  `json:"auth_time,omitempty"`
	Nonce    string `json:"nonce,omitempty"`
	ACR      string `json:"acr,omitempty"`
	AMR      string `json:"amr,omitempty"`
	AZP      string `json:"azp,omitempty"`

	// 用户信息Claims
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty"`
	Nickname          string `json:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty"`
	Picture           string `json:"picture,omitempty"`
	Website           string `json:"website,omitempty"`
	Gender            string `json:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty"`
	Zoneinfo          string `json:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty"`
	UpdatedAt         int64  `json:"updated_at,omitempty"`

	// 联系信息Claims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	PhoneVerified bool   `json:"phone_verified,omitempty"`

	// 地址Claims
	Address *OIDCAddress `json:"address,omitempty"`
}

// GetExpirationTime 实现jwt.Claims接口
func (c OIDCClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.ExpiresAt == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.ExpiresAt, 0)), nil
}

// GetIssuedAt 实现jwt.Claims接口
func (c OIDCClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	if c.IssuedAt == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}

// GetNotBefore 实现jwt.Claims接口
func (c OIDCClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

// GetAudience 实现jwt.Claims接口
func (c OIDCClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{c.Audience}, nil
}

// GetIssuer 实现jwt.Claims接口
func (c OIDCClaims) GetIssuer() (string, error) {
	return c.Issuer, nil
}

// GetSubject 实现jwt.Claims接口
func (c OIDCClaims) GetSubject() (string, error) {
	return c.Subject, nil
}

// OIDCAddress OpenID Connect地址Claims
type OIDCAddress struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"street_address,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// OIDCConfiguration OpenID Connect服务发现配置
type OIDCConfiguration struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSUri                          string   `json:"jwks_uri"`
	RegistrationEndpoint             string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}
