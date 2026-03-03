package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:20;default:'parent'" json:"role"` // parent 或 grandparent
	Name         string    `gorm:"size:100" json:"name"`                  // 用户姓名
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
