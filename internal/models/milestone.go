package models

import "time"

// MilestoneTag 里程碑标签模型
type MilestoneTag struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	DaysOffset  int       `gorm:"default:0" json:"days_offset"` // 相对出生天的偏移量
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (MilestoneTag) TableName() string {
	return "milestone_tags"
}

// DefaultMilestones 默认里程碑列表
var DefaultMilestones = []MilestoneTag{
	{Name: "第一周", DaysOffset: 7},
	{Name: "第二周", DaysOffset: 14},
	{Name: "第三周", DaysOffset: 21},
	{Name: "满月", DaysOffset: 30},
	{Name: "六周", DaysOffset: 42},
	{Name: "八周", DaysOffset: 56},
	{Name: "百日", DaysOffset: 100},
	{Name: "百日快乐", DaysOffset: 100},
	{Name: "三个月", DaysOffset: 90},
	{Name: "半岁", DaysOffset: 180},
	{Name: "九个月", DaysOffset: 270},
	{Name: "一周岁", DaysOffset: 365},
}

// CheckMilestones 检查并返回匹配的里程碑标签
func CheckMilestones(birthDate time.Time, recordDate time.Time) []string {
	var tags []string

	daysDiff := int(recordDate.Sub(birthDate).Hours() / 24)

	// 检查自定义里程碑
	for _, tag := range DefaultMilestones {
		if daysDiff == tag.DaysOffset {
			tags = append(tags, tag.Name)
		}
	}

	// 检查周标记（每周）
	if daysDiff > 7 && (daysDiff-7)%7 == 0 {
		weekNum := (daysDiff - 7) / 7
		tags = append(tags, string(rune('第'+weekNum)+'周'))
	}

	return tags
}
