package models

import "time"

// BGM 背景音乐模型（家庭成员共享的儿歌列表）
type BGM struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	BabyID     uint      `gorm:"index;not null" json:"baby_id"`      // 归属宝宝（家庭维度共享）
	UploaderID uint      `gorm:"not null" json:"uploader_id"`        // 上传者
	Title      string    `gorm:"size:200;not null" json:"title"`     // 显示名（默认取上传文件名去扩展名）
	FilePath   string    `gorm:"size:500;not null" json:"file_path"` // 相对 URL，如 /uploads/bgm/bgm_3_xxx.mp3
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (BGM) TableName() string {
	return "bgms"
}
