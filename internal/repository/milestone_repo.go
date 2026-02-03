package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

// MilestoneRepository 里程碑数据访问层
type MilestoneRepository struct {
	db *gorm.DB
}

// NewMilestoneRepository 创建里程碑仓库
func NewMilestoneRepository(db *gorm.DB) *MilestoneRepository {
	return &MilestoneRepository{db: db}
}

// FindByName 根据名称查找里程碑
func (r *MilestoneRepository) FindByName(name string) (*models.MilestoneTag, error) {
	var tag models.MilestoneTag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetAll 获取所有里程碑
func (r *MilestoneRepository) GetAll() ([]models.MilestoneTag, error) {
	var tags []models.MilestoneTag
	err := r.db.Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}
