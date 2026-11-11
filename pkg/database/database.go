package database

import (
	"anan-diary/ad-backend/internal/models"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	// 初始化默认里程碑标签（需要表已存在）
	initMilestones()

	return nil
}

// InitDBWithMigration 初始化数据库并执行完整迁移
func InitDBWithMigration(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 使用SQL脚本创建表
	if err := runMigrationScript(); err != nil {
		return err
	}

	// 初始化默认里程碑标签
	initMilestones()

	return nil
}

// runMigrationScript 执行SQL迁移脚本
func runMigrationScript() error {
	// 1. 创建 users 表
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) DEFAULT 'parent',
			name VARCHAR(100) DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 2. 创建 babies 表
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS babies (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			birth_date VARCHAR(10) NOT NULL,
			baby_id VARCHAR(50) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, baby_id)
		);
	`).Error; err != nil {
		return err
	}

	// 3. 创建 family_members 表
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS family_members (
			id SERIAL PRIMARY KEY,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			role VARCHAR(20) NOT NULL,
			is_paid BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 4. 创建 growth_records 表
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS growth_records (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			tags TEXT DEFAULT '[]',
			created_at TIMESTAMP NOT NULL,
			UNIQUE(baby_id, created_at, content)
		);
	`).Error; err != nil {
		return err
	}

	// 5. 创建 milestone_tags 表
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS milestone_tags (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL,
			days_offset INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 6. 创建邀请码表（invitations）
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS invitations (
			id SERIAL PRIMARY KEY,
			code VARCHAR(50) UNIQUE NOT NULL,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'revoked')),
			max_uses INTEGER DEFAULT 10,
			used_count INTEGER DEFAULT 0,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 7. 创建邀请码使用记录表（invitation_uses）
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS invitation_uses (
			id SERIAL PRIMARY KEY,
			invitation_id INTEGER NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			ip_address INET,
			used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 8. 创建审批系统表（membership_approvals）
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS membership_approvals (
			id SERIAL PRIMARY KEY,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(20) NOT NULL,
			name VARCHAR(100) NOT NULL,
			reason TEXT,
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
			reviewed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMP,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 9. 创建用户权限表（user_permissions）
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_permissions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			permission_type VARCHAR(50) NOT NULL,
			permission_value BOOLEAN NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, baby_id, permission_type)
		);
	`).Error; err != nil {
		return err
	}

	// 10. 创建背景音乐表（bgms）
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS bgms (
			id SERIAL PRIMARY KEY,
			baby_id INTEGER NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			uploader_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(200) NOT NULL,
			file_path VARCHAR(500) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`).Error; err != nil {
		return err
	}

	// 补齐模型中存在但建表 SQL 遗漏的列（幂等，IF NOT EXISTS）
	DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(100) DEFAULT '';`)
	DB.Exec(`ALTER TABLE babies ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500) DEFAULT '';`)
	DB.Exec(`ALTER TABLE family_members ADD COLUMN IF NOT EXISTS user_id INTEGER;`)
	DB.Exec(`ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS media TEXT;`)
	DB.Exec(`ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS growing_up BOOLEAN DEFAULT false;`)
	DB.Exec(`ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN DEFAULT false;`)

	// 创建索引
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_babies_user_id ON babies(user_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_babies_baby_id ON babies(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_family_members_baby_id ON family_members(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_family_members_role ON family_members(role);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_growth_records_baby_id ON growth_records(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_growth_records_created_at ON growth_records(created_at);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_invitations_code ON invitations(code);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_invitations_baby_id ON invitations(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_invitations_status ON invitations(status);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_invitation_uses_invitation_id ON invitation_uses(invitation_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_invitation_uses_user_id ON invitation_uses(user_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_approvals_baby_id ON membership_approvals(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_approvals_user_id ON membership_approvals(user_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_approvals_status ON membership_approvals(status);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_permissions_user_id ON user_permissions(user_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_permissions_baby_id ON user_permissions(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_bgms_baby_id ON bgms(baby_id);`)

	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// initMilestones 初始化默认里程碑标签
func initMilestones() {
	var count int64
	DB.Model(&models.MilestoneTag{}).Count(&count)
	if count == 0 {
		log.Println("Initializing default milestone tags...")
		for _, tag := range models.DefaultMilestones {
			if err := DB.Create(&tag).Error; err != nil {
				log.Printf("Failed to create milestone tag %s: %v", tag.Name, err)
			}
		}
	}
}

// GetCurrentUser 获取当前登录用户
func GetCurrentUser(c *gin.Context) *models.User {
	var user models.User
	if err := DB.First(&user, c.GetInt("user_id")).Error; err != nil {
		return nil
	}
	return &user
}

// InitTestDB 初始化测试数据库（用于开发测试）
func InitTestDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to test database: %w", err)
	}

	// 自动迁移所有表
	err = DB.AutoMigrate(
		&models.User{},
		&models.Baby{},
		&models.FamilyMember{},
		&models.GrowthRecord{},
		&models.MilestoneTag{},
		&models.Invitation{},
		&models.InviteUse{},
		&models.MembershipApproval{},
		&models.UserPermission{},
		&models.Like{},
		&models.Comment{},
		&models.UserStats{},
		&models.Subscription{},
		&models.StorageQuota{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate test database: %w", err)
	}

	return nil
}
