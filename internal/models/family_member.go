package models

import (
	"time"
)

// FamilyMember 家庭成员模型
type FamilyMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BabyID    uint      `gorm:"not null;index" json:"baby_id"`
	UserID    *uint     `gorm:"index" json:"user_id,omitempty"` // 关联的注册用户ID（可为空，表示未登录创建）
	Name      string    `gorm:"size:100;not null" json:"name"`
	Role      string    `gorm:"size:20;not null;index" json:"role"` // 爸爸/妈妈/爷爷/奶奶/姥姥/姥爷/哥哥/姐姐
	IsPaid    bool      `gorm:"default:false" json:"is_paid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	Baby    Baby       `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
	User    *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (FamilyMember) TableName() string {
	return "family_members"
}

// GetFreeLimit 获取免费名额限制
func (FamilyMember) GetFreeLimit() int {
	return 2
}
