package boot

import (
	"lauth/internal/repository"
	"lauth/pkg/database"

	"gorm.io/gorm"
)

// Repositories 包含所有仓储实例
type Repositories struct {
	AppRepo                      repository.AppRepository
	UserRepo                     repository.UserRepository
	RoleRepo                     repository.RoleRepository
	PermissionRepo               repository.PermissionRepository
	RuleRepo                     repository.RuleRepository
	OAuthClientRepo              repository.OAuthClientRepository
	OAuthClientSecretRepo        repository.OAuthClientSecretRepository
	AuthCodeRepo                 repository.AuthorizationCodeRepository
	PluginStatusRepo             repository.PluginStatusRepository
	PluginConfigRepo             repository.PluginConfigRepository
	VerificationSessionRepo      repository.VerificationSessionRepository
	PluginUserConfigRepo         repository.PluginUserConfigRepository
	PluginVerificationRecordRepo repository.PluginVerificationRecordRepository
	LoginLocationRepo            repository.LoginLocationRepository
	ProfileRepo                  repository.ProfileRepository
	FileRepo                     repository.FileRepository
	SuperAdminRepo               repository.SuperAdminRepository
}

// InitRepositories 初始化所有仓储实例
func InitRepositories(db *gorm.DB, mongodb *database.MongoClient) *Repositories {
	return &Repositories{
		AppRepo:                      repository.NewAppRepository(db),
		UserRepo:                     repository.NewUserRepository(db),
		RoleRepo:                     repository.NewRoleRepository(db),
		PermissionRepo:               repository.NewPermissionRepository(db),
		RuleRepo:                     repository.NewRuleRepository(db),
		OAuthClientRepo:              repository.NewOAuthClientRepository(db),
		OAuthClientSecretRepo:        repository.NewOAuthClientSecretRepository(db),
		AuthCodeRepo:                 repository.NewAuthorizationCodeRepository(db),
		PluginStatusRepo:             repository.NewPluginStatusRepository(db),
		PluginConfigRepo:             repository.NewPluginConfigRepository(db),
		VerificationSessionRepo:      repository.NewVerificationSessionRepository(db),
		PluginUserConfigRepo:         repository.NewPluginUserConfigRepository(db),
		PluginVerificationRecordRepo: repository.NewPluginVerificationRecordRepository(db),
		LoginLocationRepo:            repository.NewLoginLocationRepository(db),
		ProfileRepo:                  repository.NewProfileRepository(mongodb),
		FileRepo:                     repository.NewFileRepository(mongodb),
		SuperAdminRepo:               repository.NewSuperAdminRepository(db),
	}
}
