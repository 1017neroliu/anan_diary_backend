package repository

import (
	"anan-diary/ad-backend/internal/models"
	"gorm.io/gorm"
)

type InvitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create 创建邀请码
func (r *InvitationRepository) Create(invitation *models.Invitation) error {
	return r.db.Create(invitation).Error
}

// GetByCode 根据邀请码获取邀请信息
func (r *InvitationRepository) GetByCode(code string) (*models.Invitation, error) {
	var invitation models.Invitation
	err := r.db.Where("code = ?", code).First(&invitation).Error
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// GetByBabyID 获取宝宝的所有邀请码
func (r *InvitationRepository) GetByBabyID(babyID uint) ([]models.Invitation, error) {
	var invitations []models.Invitation
	err := r.db.Where("baby_id = ?", babyID).Order("created_at DESC").Find(&invitations).Error
	return invitations, err
}

// GetActiveCode 获取宝宝的一个活跃邀请码
func (r *InvitationRepository) GetActiveCode(babyID uint) (*models.Invitation, error) {
	var invitation models.Invitation
	err := r.db.Where("baby_id = ? AND status = ?", babyID, models.InvitationStatusActive).
		First(&invitation).Error
	return &invitation, err
}

// Update 更新邀请码
func (r *InvitationRepository) Update(invitation *models.Invitation) error {
	return r.db.Save(invitation).Error
}

// UpdateStatus 更新邀请码状态
func (r *InvitationRepository) UpdateStatus(babyID uint, status string) error {
	return r.db.Model(&models.Invitation{}).
		Where("baby_id = ? AND status = ?", babyID, models.InvitationStatusActive).
		Update("status", status).Error
}

// DecrementUsedCount 减少已使用次数
func (r *InvitationRepository) DecrementUsedCount(invitationID uint) error {
	return r.db.Model(&models.Invitation{}).
		Where("id = ?", invitationID).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error
}

// IncrementUsedCount 增加已使用次数
func (r *InvitationRepository) IncrementUsedCount(invitationID uint) error {
	return r.db.Model(&models.Invitation{}).
		Where("id = ?", invitationID).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// CheckInvitationUsed 检查邀请码是否已达到最大使用次数
func (r *InvitationRepository) CheckInvitationUsed(invitationID uint) (bool, error) {
	var invitation models.Invitation
	err := r.db.Select("max_uses", "used_count").
		Where("id = ?", invitationID).
		First(&invitation).Error
	if err != nil {
		return false, err
	}
	return invitation.UsedCount >= invitation.MaxUses, nil
}

// RecordInviteUse 记录邀请码使用
func (r *InvitationRepository) RecordInviteUse(invitationID, userID uint, ipAddress string) error {
	use := &models.InviteUse{
		InvitationID: invitationID,
		UserID:       userID,
		IPAddress:    ipAddress,
	}
	return r.db.Create(use).Error
}

// GetInvitationUsageCount 获取邀请码使用次数
func (r *InvitationRepository) GetInvitationUsageCount(invitationID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.InviteUse{}).Where("invitation_id = ?", invitationID).Count(&count).Error
	return count, err
}

// RecordInvitationUsage 记录使用并更新邀请码状态
func (r *InvitationRepository) RecordInvitationUsage(invitationID, userID uint, ipAddress string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否达到最大使用次数
		maxUses, err := r.GetInvitationUsageCount(invitationID)
		if err != nil {
			return err
		}

		// 获取邀请码信息
		var invitation models.Invitation
		if err := tx.Where("id = ?", invitationID).First(&invitation).Error; err != nil {
			return err
		}

		// 检查是否已达到最大使用次数
		if int(maxUses) >= invitation.MaxUses {
			// 更新状态为已过期
			return tx.Model(&invitation).Update("status", models.InvitationStatusExpired).Error
		}

		// 增加使用次数
		if err := tx.Model(&invitation).UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}

		// 记录使用
		use := &models.InviteUse{
			InvitationID: invitationID,
			UserID:       userID,
			IPAddress:    ipAddress,
		}
		return tx.Create(use).Error
	})
}
