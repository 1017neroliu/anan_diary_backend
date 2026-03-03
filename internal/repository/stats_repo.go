package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

type StatsRepository struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// GetOrCreate 获取或创建用户统计
func (r *StatsRepository) GetOrCreate(userID uint) (*models.UserStats, error) {
	var stats models.UserStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		stats = models.UserStats{
			UserID:         userID,
			TotalRecords:   0,
			TotalLikes:     0,
			TotalComments:  0,
			StorageUsedBytes: 0,
			BabyCount:      0,
		}
		err = r.db.Create(&stats).Error
	}
	return &stats, err
}

// IncrementRecordCount 增加记录计数
func (r *StatsRepository) IncrementRecordCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_records", gorm.Expr("total_records + 1")).Error
}

// DecrementRecordCount 减少记录计数
func (r *StatsRepository) DecrementRecordCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ? AND total_records > 0", userID).
		UpdateColumn("total_records", gorm.Expr("total_records - 1")).Error
}

// IncrementLikeCount 增加点赞计数
func (r *StatsRepository) IncrementLikeCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_likes", gorm.Expr("total_likes + 1")).Error
}

// DecrementLikeCount 减少点赞计数
func (r *StatsRepository) DecrementLikeCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ? AND total_likes > 0", userID).
		UpdateColumn("total_likes", gorm.Expr("total_likes - 1")).Error
}

// IncrementCommentCount 增加评论计数
func (r *StatsRepository) IncrementCommentCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_comments", gorm.Expr("total_comments + 1")).Error
}

// DecrementCommentCount 减少评论计数
func (r *StatsRepository) DecrementCommentCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ? AND total_comments > 0", userID).
		UpdateColumn("total_comments", gorm.Expr("total_comments - 1")).Error
}

// UpdateStorageUsed 更新存储使用量
func (r *StatsRepository) UpdateStorageUsed(userID uint, sizeBytes int64) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ?", userID).
		UpdateColumn("storage_used_bytes", sizeBytes).Error
}

// IncrementBabyCount 增加宝宝计数
func (r *StatsRepository) IncrementBabyCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ?", userID).
		UpdateColumn("baby_count", gorm.Expr("baby_count + 1")).Error
}

// DecrementBabyCount 减少宝宝计数
func (r *StatsRepository) DecrementBabyCount(userID uint) error {
	return r.db.Model(&models.UserStats{}).
		Where("user_id = ? AND baby_count > 0", userID).
		UpdateColumn("baby_count", gorm.Expr("baby_count - 1")).Error
}

// GetStats 获取用户统计
func (r *StatsRepository) GetStats(userID uint) (*models.UserStats, error) {
	var stats models.UserStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// GetStatsByBabyID 获取宝宝的成员贡献统计
func (r *StatsRepository) GetStatsByBabyID(babyID uint) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 优先从 family_members 表获取成员列表
	var members []models.FamilyMember
	err := r.db.Where("baby_id = ?", babyID).
		Preload("User").
		Find(&members).Error
	if err != nil {
		return nil, err
	}

	// 统计每个成员的记录数
	for _, member := range members {
		var recordCount int64
		if member.UserID != nil {
			r.db.Model(&models.GrowthRecord{}).
				Where("baby_id = ? AND user_id = ?", babyID, *member.UserID).
				Count(&recordCount)

			name := member.Name
			if member.User != nil && member.User.Name != "" {
				name = member.User.Name
			}

			results = append(results, map[string]interface{}{
				"user_id":      *member.UserID,
				"name":         name,
				"role":         member.Role,
				"record_count": recordCount,
			})
		}
	}

	return results, nil
}

// GetTodayRecordCount 获取今日记录数
func (r *StatsRepository) GetTodayRecordCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.GrowthRecord{}).
		Where("user_id = ? AND DATE(created_at) = CURRENT_DATE", userID).
		Count(&count).Error
	return count, err
}

// GetWeeklyRecordCount 获取本周记录数
func (r *StatsRepository) GetWeeklyRecordCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.GrowthRecord{}).
		Where("user_id = ? AND created_at >= CURRENT_DATE - INTERVAL '7 days'", userID).
		Count(&count).Error
	return count, err
}

// GetMonthlyRecordCount 获取本月记录数
func (r *StatsRepository) GetMonthlyRecordCount(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.GrowthRecord{}).
		Where("user_id = ? AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)", userID).
		Count(&count).Error
	return count, err
}
