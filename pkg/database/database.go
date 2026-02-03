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

	// 创建索引
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_babies_user_id ON babies(user_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_babies_baby_id ON babies(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_family_members_baby_id ON family_members(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_family_members_role ON family_members(role);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_growth_records_baby_id ON growth_records(baby_id);`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_growth_records_created_at ON growth_records(created_at);`)

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
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate test database: %w", err)
	}

	return nil
}
