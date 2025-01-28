package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Profile 用户档案
type Profile struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID     string                 `bson:"user_id" json:"user_id"`         // 关联的用户ID
	AppID      string                 `bson:"app_id" json:"app_id"`           // 关联的应用ID
	Avatar     string                 `bson:"avatar" json:"avatar"`           // 头像URL
	Nickname   string                 `bson:"nickname" json:"nickname"`       // 昵称
	RealName   string                 `bson:"real_name" json:"real_name"`     // 真实姓名
	Gender     string                 `bson:"gender" json:"gender"`           // 性别
	Birthday   *time.Time             `bson:"birthday" json:"birthday"`       // 生日
	Email      string                 `bson:"email" json:"email"`             // 邮箱
	Phone      string                 `bson:"phone" json:"phone"`             // 手机号
	Address    *Address               `bson:"address" json:"address"`         // 地址
	Social     *Social                `bson:"social" json:"social"`           // 社交信息
	CustomData map[string]interface{} `bson:"custom_data" json:"custom_data"` // 自定义数据
	Files      []File                 `bson:"files" json:"files"`             // 关联的文件
	CreatedAt  time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time              `bson:"updated_at" json:"updated_at"`
}

// Address 地址信息
type Address struct {
	Country  string `bson:"country" json:"country"`     // 国家
	Province string `bson:"province" json:"province"`   // 省份
	City     string `bson:"city" json:"city"`           // 城市
	District string `bson:"district" json:"district"`   // 区县
	Street   string `bson:"street" json:"street"`       // 街道
	PostCode string `bson:"post_code" json:"post_code"` // 邮编
}

// Social 社交信息
type Social struct {
	Website  string `bson:"website" json:"website"`   // 个人网站
	Github   string `bson:"github" json:"github"`     // Github
	Twitter  string `bson:"twitter" json:"twitter"`   // Twitter
	Facebook string `bson:"facebook" json:"facebook"` // Facebook
	LinkedIn string `bson:"linkedin" json:"linkedin"` // LinkedIn
	WeChat   string `bson:"wechat" json:"wechat"`     // 微信
	QQ       string `bson:"qq" json:"qq"`             // QQ
	Weibo    string `bson:"weibo" json:"weibo"`       // 微博
}

// File 文件信息
type File struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID     string                 `bson:"user_id" json:"user_id"`         // 关联的用户ID
	AppID      string                 `bson:"app_id" json:"app_id"`           // 关联的应用ID
	Name       string                 `bson:"name" json:"name"`               // 文件名
	Type       string                 `bson:"type" json:"type"`               // 文件类型(MIME类型)
	Size       int64                  `bson:"size" json:"size"`               // 文件大小(字节)
	Path       string                 `bson:"path" json:"path"`               // 存储路径
	URL        string                 `bson:"url" json:"url"`                 // 访问URL
	Hash       string                 `bson:"hash" json:"hash"`               // 文件哈希值
	Tags       []string               `bson:"tags" json:"tags"`               // 标签
	CustomData map[string]interface{} `bson:"custom_data" json:"custom_data"` // 自定义数据
	CreatedAt  time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time              `bson:"updated_at" json:"updated_at"`
}

// CreateProfileRequest 创建用户档案请求
type CreateProfileRequest struct {
	Avatar     string                 `json:"avatar"`
	Nickname   string                 `json:"nickname"`
	RealName   string                 `json:"real_name"`
	Gender     string                 `json:"gender"`
	Birthday   *time.Time             `json:"birthday"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	Address    *Address               `json:"address"`
	Social     *Social                `json:"social"`
	CustomData map[string]interface{} `json:"custom_data"`
}

// UpdateProfileRequest 更新用户档案请求
type UpdateProfileRequest struct {
	Avatar     *string                `json:"avatar"`
	Nickname   *string                `json:"nickname"`
	RealName   *string                `json:"real_name"`
	Gender     *string                `json:"gender"`
	Birthday   *time.Time             `json:"birthday"`
	Email      *string                `json:"email"`
	Phone      *string                `json:"phone"`
	Address    *Address               `json:"address"`
	Social     *Social                `json:"social"`
	CustomData map[string]interface{} `json:"custom_data"`
}

// FileUploadRequest 文件上传请求
type FileUploadRequest struct {
	Tags       []string               `json:"tags"`
	CustomData map[string]interface{} `json:"custom_data"`
}

// FileUpdateRequest 文件更新请求
type FileUpdateRequest struct {
	Name       *string                `json:"name"`
	Tags       []string               `json:"tags"`
	CustomData map[string]interface{} `json:"custom_data"`
}
