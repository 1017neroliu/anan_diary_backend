package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// Create 创建点赞
func (r *LikeRepository) Create(like *models.Like) error {
	return r.db.Create(like).Error
}

// Delete 取消点赞
func (r *LikeRepository) Delete(growthRecordID, userID uint) error {
	return r.db.Where("growth_record_id = ? AND user_id = ?", growthRecordID, userID).
		Delete(&models.Like{}).Error
}

// GetByUser 获取用户的所有点赞记录
func (r *LikeRepository) GetByUser(userID uint) ([]models.Like, error) {
	var likes []models.Like
	err := r.db.Where("user_id = ?", userID).
		Preload("GrowthRecord").
		Order("created_at DESC").
		Find(&likes).Error
	return likes, err
}

// GetByGrowthRecord 获取记录的所有点赞者
func (r *LikeRepository) GetByGrowthRecord(growthRecordID uint) ([]models.Like, error) {
	var likes []models.Like
	err := r.db.Where("growth_record_id = ?", growthRecordID).
		Preload("User").
		Order("created_at ASC").
		Find(&likes).Error
	return likes, err
}

// GetLikeCount 获取点赞数量
func (r *LikeRepository) GetLikeCount(growthRecordID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Like{}).Where("growth_record_id = ?", growthRecordID).Count(&count).Error
	return count, err
}

// HasLiked 检查用户是否点赞了某条记录
func (r *LikeRepository) HasLiked(growthRecordID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Like{}).
		Where("growth_record_id = ? AND user_id = ?", growthRecordID, userID).
		Count(&count).Error
	return count > 0, err
}

// IncrementLikeCount 增加点赞计数
func (r *LikeRepository) IncrementLikeCount(growthRecordID uint) error {
	return r.db.Model(&models.GrowthRecord{}).
		Where("id = ?", growthRecordID).
		UpdateColumn("likes_count", gorm.Expr("COALESCE(likes_count, 0) + 1")).Error
}

// DecrementLikeCount 减少点赞计数
func (r *LikeRepository) DecrementLikeCount(growthRecordID uint) error {
	return r.db.Model(&models.GrowthRecord{}).
		Where("id = ? AND likes_count > 0", growthRecordID).
		UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error
}

// ResetLikesCount 重置点赞计数
func (r *LikeRepository) ResetLikesCount(growthRecordID uint) error {
	return r.db.Model(&models.GrowthRecord{}).
		Where("id = ?", growthRecordID).
		UpdateColumn("likes_count", 0).Error
}

// GetTopLikedRecords 获取获赞最多的记录
func (r *LikeRepository) GetTopLikedRecords(babyID uint, limit int) ([]models.GrowthRecord, error) {
	var records []models.GrowthRecord
	err := r.db.Table("growth_records").
		Select("*, (SELECT COUNT(*) FROM likes WHERE likes.growth_record_id = growth_records.id) as likes_count").
		Where("growth_records.baby_id = ?", babyID).
		Order("likes_count DESC").
		Limit(limit).
		Scan(&records).Error

	// 预加载关联数据
	if err == nil && len(records) > 0 {
		for i := range records {
			growthRecordID := records[i].ID
			r.db.Where("growth_record_id = ?", growthRecordID).First(&records[i])
		}
	}

	return records, err
}
