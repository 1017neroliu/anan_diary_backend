package repository

import (
	"anan-diary/ad-backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create 创建订阅
func (r *SubscriptionRepository) Create(subscription *models.Subscription) error {
	return r.db.Create(subscription).Error
}

// GetByID 获取订阅详情
func (r *SubscriptionRepository) GetByID(id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetActiveByUserID 获取用户的活跃订阅
func (r *SubscriptionRepository) GetActiveByUserID(userID uint) (*models.Subscription, error) {
	var subscription models.Subscription
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetByUserID 获取用户的所有订阅
func (r *SubscriptionRepository) GetByUserID(userID uint) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&subscriptions).Error
	return subscriptions, err
}

// GetByPlan 获取特定计划的活跃订阅数
func (r *SubscriptionRepository) GetActiveCountByPlan(plan string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Subscription{}).
		Where("plan = ? AND is_active = ?", plan, true).
		Count(&count).Error
	return count, err
}

// ActivateSubscription 激活订阅
func (r *SubscriptionRepository) ActivateSubscription(subscriptionID uint) error {
	// 先禁用该用户的其他活跃订阅
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 获取订阅详情
		var subscription models.Subscription
		if err := tx.Where("id = ?", subscriptionID).First(&subscription).Error; err != nil {
			return err
		}

		// 禁用其他活跃订阅
		if err := tx.Model(&models.Subscription{}).
			Where("user_id = ? AND id != ?", subscription.UserID, subscriptionID).
			Update("is_active", false).Error; err != nil {
			return err
		}

		// 激活当前订阅
		now := time.Now()
		return tx.Model(&subscription).
			Updates(map[string]interface{}{
				"is_active":    true,
				"start_date":   now,
				"end_date":     now.AddDate(0, 1, 0), // 默认1个月
				"updated_at":   now,
			}).Error
	})
}

// RenewSubscription 续期订阅
func (r *SubscriptionRepository) RenewSubscription(subscriptionID uint, months int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 获取订阅详情
		var subscription models.Subscription
		if err := tx.Where("id = ?", subscriptionID).First(&subscription).Error; err != nil {
			return err
		}

		// 延长到期时间
		newEndDate := subscription.EndDate.AddDate(0, months, 0)
		return tx.Model(&subscription).
			Updates(map[string]interface{}{
				"is_active":    true,
				"end_date":     newEndDate,
				"updated_at":   time.Now(),
			}).Error
	})
}

// ExpireSubscription 到期订阅
func (r *SubscriptionRepository) ExpireSubscription(subscriptionID uint) error {
	return r.db.Model(&models.Subscription{}).
		Where("id = ?", subscriptionID).
		Update("is_active", false).Error
}

// GetStorageQuota 获取存储配额
func (r *SubscriptionRepository) GetStorageQuota(plan string) (int64, error) {
	var quota struct {
		MaxMembers      int64
		StorageLimitGB  int64
		SupportsMultiBaby bool
		HasPDFExport     bool
	}

	err := r.db.Model(&models.StorageQuota{}).
		Where("plan = ?", plan).
		First(&quota).Error
	if err != nil {
		return 0, err
	}

	// 返回存储限制（字节）
	return quota.StorageLimitGB * 1024 * 1024 * 1024, nil
}

// GetMaxMembers 获取最大成员数
func (r *SubscriptionRepository) GetMaxMembers(plan string) (int64, error) {
	var quota struct {
		MaxMembers int64
	}

	err := r.db.Model(&models.StorageQuota{}).
		Where("plan = ?", plan).
		First(&quota).Error
	if err != nil {
		return 0, err
	}

	return quota.MaxMembers, nil
}

// SupportsMultiBaby 检查是否支持多宝宝
func (r *SubscriptionRepository) SupportsMultiBaby(plan string) (bool, error) {
	var quota struct {
		SupportsMultiBaby bool
	}

	err := r.db.Model(&models.StorageQuota{}).
		Where("plan = ?", plan).
		First(&quota).Error
	if err != nil {
		return false, err
	}

	return quota.SupportsMultiBaby, nil
}

// HasPDFExport 检查是否有PDF导出功能
func (r *SubscriptionRepository) HasPDFExport(plan string) (bool, error) {
	var quota struct {
		HasPDFExport bool
	}

	err := r.db.Model(&models.StorageQuota{}).
		Where("plan = ?", plan).
		First(&quota).Error
	if err != nil {
		return false, err
	}

	return quota.HasPDFExport, nil
}

// CheckSubscriptionStatus 检查订阅状态
func (r *SubscriptionRepository) CheckSubscriptionStatus(userID uint) (bool, bool, error) {
	// 检查是否有活跃订阅
	activeSub, err := r.GetActiveByUserID(userID)
	if err != nil {
		return false, false, err
	}

	// 检查是否过期
	now := time.Now()
	isExpired := activeSub != nil && activeSub.EndDate.Before(now)

	// 如果订阅已过期，自动禁用
	if isExpired && activeSub.IsActive {
		_ = r.ExpireSubscription(activeSub.ID)
		return false, false, nil
	}

	return activeSub != nil && !isExpired, activeSub != nil, nil
}

// RecordPayment 记录支付
func (r *SubscriptionRepository) RecordPayment(userID uint, plan string, amount float64) error {
	subscription := &models.Subscription{
		UserID:      userID,
		Plan:        plan,
		IsActive:    true,
		StartDate:   time.Now(),
		EndDate:     time.Now().AddDate(0, 1, 0),
	}
	return r.Create(subscription)
}

// GetActiveSubscriptionCount 获取特定计划的活跃订阅数
func (r *SubscriptionRepository) GetActiveSubscriptionCount() (map[string]int64, error) {
	type PlanCount struct {
		Plan string
		Count int64
	}

	var results []PlanCount
	err := r.db.Model(&models.Subscription{}).
		Select("plan, COUNT(*) as count").
		Where("is_active = ?", true).
		Group("plan").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Plan] = result.Count
	}

	return counts, nil
}

// GetSubscriptionStats 获取订阅统计数据
func (r *SubscriptionRepository) GetSubscriptionStats() (map[string]interface{}, error) {
	// 获取各计划人数
	counts, err := r.GetActiveSubscriptionCount()
	if err != nil {
		return nil, err
	}

	// 获取总订阅数
	var totalSubs int64
	err = r.db.Model(&models.Subscription{}).Count(&totalSubs).Error
	if err != nil {
		return nil, err
	}

	// 获取总活跃订阅数
	var activeSubs int64
	err = r.db.Model(&models.Subscription{}).Where("is_active = ?", true).Count(&activeSubs).Error
	if err != nil {
		return nil, err
	}

	// 获取总用户数
	var totalUsers int64
	err = r.db.Model(&models.User{}).Count(&totalUsers).Error
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_subscriptions": totalSubs,
		"active_subscriptions": activeSubs,
		"total_users":          totalUsers,
		"active_users":         totalUsers,
		"by_plan":              counts,
	}, nil
}

// RenewMultipleMonths 续期多个月份
func (r *SubscriptionRepository) RenewMultipleMonths(subscriptionID uint, months int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var subscription models.Subscription
		if err := tx.Where("id = ?", subscriptionID).First(&subscription).Error; err != nil {
			return err
		}

		newEndDate := subscription.EndDate.AddDate(0, months, 0)
		return tx.Model(&subscription).
			Update("end_date", newEndDate).
			Update("updated_at", time.Now()).Error
	})
}

// GetSubscriptionDetail 获取订阅详情（包含配额信息）
func (r *SubscriptionRepository) GetSubscriptionDetail(userID uint) (map[string]interface{}, error) {
	// 获取活跃订阅
	subscription, err := r.GetActiveByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 获取配额信息
	maxMembers, _ := r.GetMaxMembers(subscription.Plan)
	supportsMultiBaby, _ := r.SupportsMultiBaby(subscription.Plan)
	hasPDFExport, _ := r.HasPDFExport(subscription.Plan)

	return map[string]interface{}{
		"is_active":      subscription.IsActive,
		"plan":           subscription.Plan,
		"start_date":     subscription.StartDate,
		"end_date":       subscription.EndDate,
		"max_members":    maxMembers,
		"supports_multi_baby": supportsMultiBaby,
		"has_pdf_export": hasPDFExport,
	}, nil
}
