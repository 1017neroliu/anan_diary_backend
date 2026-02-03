package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

// BabyRepository 宝宝数据访问层
type BabyRepository struct {
	db *gorm.DB
}

// NewBabyRepository 创建宝宝仓库
func NewBabyRepository(db *gorm.DB) *BabyRepository {
	return &BabyRepository{db: db}
}

// Create 创建宝宝档案
func (r *BabyRepository) Create(baby *models.Baby) error {
	return r.db.Create(baby).Error
}

// GetByID 根据ID获取宝宝
func (r *BabyRepository) GetByID(id uint) (*models.Baby, error) {
	var baby models.Baby
	err := r.db.Preload("FamilyMembers").First(&baby, id).Error
	if err != nil {
		return nil, err
	}
	return &baby, nil
}

// GetByUserID 获取用户的宝宝列表
func (r *BabyRepository) GetByUserID(userID uint) ([]models.Baby, error) {
	var babies []models.Baby
	err := r.db.Where("user_id = ?", userID).Find(&babies).Error
	if err != nil {
		return nil, err
	}
	return babies, nil
}

// GetByBabyID 根据出生证编号获取宝宝
func (r *BabyRepository) GetByBabyID(babyID string) (*models.Baby, error) {
	var baby models.Baby
	err := r.db.Where("baby_id = ?", babyID).First(&baby).Error
	if err != nil {
		return nil, err
	}
	return &baby, nil
}

// Delete 删除宝宝
func (r *BabyRepository) Delete(id uint) error {
	return r.db.Delete(&models.Baby{}, id).Error
}

// Update 更新宝宝信息
func (r *BabyRepository) Update(baby *models.Baby) error {
	return r.db.Save(baby).Error
}
