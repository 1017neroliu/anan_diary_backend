package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
	"time"
)

// GrowthRecordRepository 成长记录数据访问层
type GrowthRecordRepository struct {
	db *gorm.DB
}

// NewGrowthRecordRepository 创建成长记录仓库
func NewGrowthRecordRepository(db *gorm.DB) *GrowthRecordRepository {
	return &GrowthRecordRepository{db: db}
}

// Create 创建成长记录
func (r *GrowthRecordRepository) Create(record *models.GrowthRecord) error {
	return r.db.Create(record).Error
}

// GetByID 根据ID获取记录
func (r *GrowthRecordRepository) GetByID(id uint) (*models.GrowthRecord, error) {
	var record models.GrowthRecord
	err := r.db.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByBabyID 获取宝宝的成长记录列表（按时间倒序）
func (r *GrowthRecordRepository) GetByBabyID(babyID uint) ([]models.GrowthRecord, error) {
	var records []models.GrowthRecord
	err := r.db.Where("baby_id = ?", babyID).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetByBabyAndDate 获取宝宝在特定日期的记录
func (r *GrowthRecordRepository) GetByBabyAndDate(babyID uint, date time.Time) (*models.GrowthRecord, error) {
	var record models.GrowthRecord
	dateStr := date.Format("2006-01-02 15:04:05")
	err := r.db.Where("baby_id = ? AND created_at = ?", babyID, dateStr).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Update 更新记录的可编辑字段（content/tags/media）。
// 使用 map 指定列更新：不触碰 GrowingUp/IsPinned/CreatedAt 等字段，
// 也避免 struct Updates 的零值跳过问题。
func (r *GrowthRecordRepository) Update(id uint, content, tags, media string) error {
	result := r.db.Model(&models.GrowthRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"content": content,
			"tags":    tags,
			"media":   media,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // 并发删除场景兜底
	}
	return nil
}

// Delete 删除记录
func (r *GrowthRecordRepository) Delete(id uint) error {
	return r.db.Delete(&models.GrowthRecord{}, id).Error
}
