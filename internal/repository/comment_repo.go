package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 创建评论
func (r *CommentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

// GetByID 获取评论详情
func (r *CommentRepository) GetByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	err := r.db.Preload("User").
		Preload("Parent").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByGrowthRecord 获取记录的所有评论
func (r *CommentRepository) GetByGrowthRecord(growthRecordID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("growth_record_id = ?", growthRecordID).
		Preload("User").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

// GetByUser 获取用户的所有评论
func (r *CommentRepository) GetByUser(userID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.Where("user_id = ?", userID).
		Preload("GrowthRecord").
		Preload("User").
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

// Update 更新评论
func (r *CommentRepository) Update(comment *models.Comment) error {
	comment.UpdatedAt = models.GetTimeNow()
	return r.db.Save(comment).Error
}

// Delete 删除评论
func (r *CommentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

// DeleteByUser 删除用户的所有评论
func (r *CommentRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.Comment{}).Error
}

// DeleteByGrowthRecord 删除记录的所有评论
func (r *CommentRepository) DeleteByGrowthRecord(growthRecordID uint) error {
	return r.db.Where("growth_record_id = ?", growthRecordID).Delete(&models.Comment{}).Error
}

// GetCommentCount 获取评论数量
func (r *CommentRepository) GetCommentCount(growthRecordID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Comment{}).Where("growth_record_id = ?", growthRecordID).Count(&count).Error
	return count, err
}

// HasLiked 检查用户是否点赞了某条记录
func (r *CommentRepository) HasLiked(growthRecordID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Like{}).
		Where("growth_record_id = ? AND user_id = ?", growthRecordID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetLikeCount 获取点赞数量
func (r *CommentRepository) GetLikeCount(growthRecordID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Like{}).Where("growth_record_id = ?", growthRecordID).Count(&count).Error
	return count, err
}
