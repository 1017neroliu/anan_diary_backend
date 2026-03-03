package models

import (
	"encoding/json"
	"time"
)

// GrowthRecord 成长记录模型
type GrowthRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	BabyID    uint      `gorm:"not null;index" json:"baby_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Tags      string    `gorm:"type:text" json:"tags"`       // JSON 字符串
	Media     string    `gorm:"type:text" json:"media"`      // 媒体文件 JSON 字符串
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`

	// 关联关系
	Baby Baby `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
}

// TableName 指定表名
func (GrowthRecord) TableName() string {
	return "growth_records"
}

// GetTags 解析标签 JSON 字符串
func (r *GrowthRecord) GetTags() []string {
	if r.Tags == "" || r.Tags == "[]" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
		return []string{}
	}
	return tags
}
