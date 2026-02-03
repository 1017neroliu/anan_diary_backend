package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

// FamilyMemberRepository 家庭成员数据访问层
type FamilyMemberRepository struct {
	db *gorm.DB
}

// NewFamilyMemberRepository 创建家庭成员仓库
func NewFamilyMemberRepository(db *gorm.DB) *FamilyMemberRepository {
	return &FamilyMemberRepository{db: db}
}

// Create 创建家庭成员
func (r *FamilyMemberRepository) Create(member *models.FamilyMember) error {
	return r.db.Create(member).Error
}

// GetByBabyID 获取宝宝的成员列表
func (r *FamilyMemberRepository) GetByBabyID(babyID uint) ([]models.FamilyMember, error) {
	var members []models.FamilyMember
	err := r.db.Where("baby_id = ?", babyID).Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// CountByBabyID 统计宝宝成员数量
func (r *FamilyMemberRepository) CountByBabyID(babyID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.FamilyMember{}).Where("baby_id = ?", babyID).Count(&count).Error
	return count, err
}

// GetBabyByID 获取宝宝信息（用于权限检查）
func (r *FamilyMemberRepository) GetBabyByID(babyID uint) (*models.Baby, error) {
	var baby models.Baby
	err := r.db.First(&baby, babyID).Error
	if err != nil {
		return nil, err
	}
	return &baby, nil
}
