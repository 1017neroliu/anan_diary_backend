package models

import "time"

// BabyPhoto 宝宝照片模型（家庭共享的相册，hero 横幅轮播数据源）
type BabyPhoto struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	BabyID     uint      `gorm:"index;not null" json:"baby_id"`       // 归属宝宝
	UploaderID uint      `gorm:"not null" json:"uploader_id"`         // 上传者
	ImagePath  string    `gorm:"size:500;not null" json:"image_path"` // 相对 URL，如 /uploads/photos/photo_3_xxx.jpg
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (BabyPhoto) TableName() string {
	return "baby_photos"
}

// PhotoLike 照片点赞模型
type PhotoLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PhotoID   uint      `gorm:"not null" json:"photo_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (PhotoLike) TableName() string {
	return "photo_likes"
}
