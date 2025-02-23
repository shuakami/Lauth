package boot

import (
	"lauth/internal/model"
	"lauth/pkg/config"
	"lauth/pkg/database"

	"gorm.io/gorm"
)

// InitDB 初始化 PostgreSQL 数据库连接
func InitDB(cfg *database.Config) (*gorm.DB, error) {
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		return nil, err
	}

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&model.App{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
		&model.Rule{},
		&model.RuleCondition{},
		&model.OAuthClient{},
		&model.OAuthClientSecret{},
		&model.AuthorizationCode{},
		&model.PluginStatus{},
		&model.PluginConfig{},
		&model.VerificationSession{},
		&model.PluginUserConfig{},
		&model.PluginVerificationRecord{},
		&model.LoginLocation{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

// InitMongo 初始化 MongoDB 连接
func InitMongo(cfg *config.MongoDBConfig) (*database.MongoClient, error) {
	mongoConfig := &database.MongoDBConfig{
		URI:         cfg.URI,
		Database:    cfg.Database,
		MaxPoolSize: cfg.MaxPoolSize,
		MinPoolSize: cfg.MinPoolSize,
	}
	return database.NewMongoClient(mongoConfig)
}
