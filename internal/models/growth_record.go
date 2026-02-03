package models

import (
	"time"
)

// GrowthRecord 成长记录模型
type GrowthRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	BabyID    uint      `gorm:"not null;index" json:"baby_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Tags      string    `gorm:"type:text" json:"tags"` // JSON字符串
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`

	// 关联关系
	Baby Baby `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
}

// TableName 指定表名
func (GrowthRecord) TableName() string {
	return "growth_records"
}

// GetTags 解析标签JSON字符串
func (r *GrowthRecord) GetTags() []string {
	if r.Tags == "[]" || r.Tags == "" {
		return []string{}
	}
	// 简化处理，实际应解析JSON
	return []string{}
}

// ParseTags 从JSON字符串解析标签
func (r *GrowthRecord) ParseTags() []string {
	if r.Tags == "[]" || r.Tags == "" {
		return []string{}
	}
	// TODO: 实现JSON解析
	return []string{}
}

// SetTags 设置标签JSON字符串
func (r *GrowthRecord) SetTags(tags []string) {
	if len(tags) == 0 {
		r.Tags = "[]"
		return
	}
	// 简化处理，实际应序列化为JSON
	// TODO: 实现JSON序列化
	r.Tags = "[]"
}
