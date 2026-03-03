package models

import (
	"time"
)

// Baby 宝宝档案模型
type Baby struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	BirthDate string    `gorm:"size:10;not null" json:"birth_date"`
	BabyID    string    `gorm:"uniqueIndex;size:50;not null" json:"baby_id"`
	AvatarURL string    `gorm:"size:500" json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	FamilyMembers []FamilyMember `gorm:"foreignKey:BabyID" json:"family_members,omitempty"`
}

// TableName 指定表名
func (Baby) TableName() string {
	return "babies"
}

// GetAgeInDays 计算宝宝当前年龄（天数）
func (b *Baby) GetAgeInDays() int {
	birth, err := time.Parse("2006-01-02", b.BirthDate)
	if err != nil {
		return 0
	}
	return int(time.Since(birth).Hours() / 24)
}

// GetAgeInMonths 计算宝宝当前年龄（月数）
func (b *Baby) GetAgeInMonths() float64 {
	birth, err := time.Parse("2006-01-02", b.BirthDate)
	if err != nil {
		return 0
	}
	return float64(time.Since(birth).Hours()/24) / 30
}
