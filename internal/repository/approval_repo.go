package repository

import (
	"anan-diary/ad-backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type ApprovalRepository struct {
	db *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

// Create 创建成员申请
func (r *ApprovalRepository) Create(approval *models.MembershipApproval) error {
	return r.db.Create(approval).Error
}

// GetPendingByBabyID 获取宝宝的待审批申请
func (r *ApprovalRepository) GetPendingByBabyID(babyID uint) ([]models.MembershipApproval, error) {
	var approvals []models.MembershipApproval
	err := r.db.Where("baby_id = ? AND status = ?", babyID, models.ApprovalStatusPending).
		Order("created_at ASC").
		Preload("User").
		Find(&approvals).Error
	return approvals, err
}

// GetByID 获取审批记录详情
func (r *ApprovalRepository) GetByID(id uint) (*models.MembershipApproval, error) {
	var approval models.MembershipApproval
	err := r.db.Preload("User").Preload("Reviewer").
		First(&approval, id).Error
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

// GetByUserAndBaby 获取用户对宝宝的特定状态申请
func (r *ApprovalRepository) GetByUserAndBaby(userID, babyID uint, status string) (*models.MembershipApproval, error) {
	var approval models.MembershipApproval
	err := r.db.Where("user_id = ? AND baby_id = ? AND status = ?", userID, babyID, status).
		First(&approval).Error
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

// GetByUserAndBabyRole 获取用户对宝宝的特定角色申请
func (r *ApprovalRepository) GetByUserAndBabyRole(userID, babyID uint, role string) (*models.MembershipApproval, error) {
	var approval models.MembershipApproval
	err := r.db.Where("user_id = ? AND baby_id = ? AND role = ?", userID, babyID, role).
		First(&approval).Error
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

// GetByUser 获取用户的所有申请
func (r *ApprovalRepository) GetByUser(userID uint) ([]models.MembershipApproval, error) {
	var approvals []models.MembershipApproval
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Preload("User").
		Find(&approvals).Error
	return approvals, err
}

// Update 审批申请
func (r *ApprovalRepository) Update(approval *models.MembershipApproval) error {
	return r.db.Save(approval).Error
}

// Approve 批准申请
func (r *ApprovalRepository) Approve(approvalID, reviewerID uint, notes string) error {
	now := time.Now()
	approval := &models.MembershipApproval{
		ID:          approvalID,
		Status:      models.ApprovalStatusApproved,
		ReviewedBy:  &reviewerID,
		ReviewedAt:  &now,
		Notes:       notes,
		UpdatedAt:   time.Now(),
	}
	return r.db.Model(&models.MembershipApproval{}).
		Where("id = ?", approvalID).
		Updates(approval).Error
}

// Reject 拒绝申请
func (r *ApprovalRepository) Reject(approvalID, reviewerID uint, notes string) error {
	now := time.Now()
	approval := &models.MembershipApproval{
		ID:          approvalID,
		Status:      models.ApprovalStatusRejected,
		ReviewedBy:  &reviewerID,
		ReviewedAt:  &now,
		Notes:       notes,
		UpdatedAt:   time.Now(),
	}
	return r.db.Model(&models.MembershipApproval{}).
		Where("id = ?", approvalID).
		Updates(approval).Error
}

// Cancel 取消申请
func (r *ApprovalRepository) Cancel(approvalID, userID uint) error {
	return r.db.Model(&models.MembershipApproval{}).
		Where("id = ? AND user_id = ? AND status = ?", approvalID, userID, models.ApprovalStatusPending).
		Update("status", models.ApprovalStatusCancelled).Error
}

// GetApprovedMembers 获取宝宝已批准的成员列表
func (r *ApprovalRepository) GetApprovedMembers(babyID uint) ([]models.MembershipApproval, error) {
	var approvals []models.MembershipApproval
	err := r.db.Where("baby_id = ? AND status = ?", babyID, models.ApprovalStatusApproved).
		Preload("User").
		Find(&approvals).Error
	return approvals, err
}

// GetMemberCount 获取宝宝已批准的成员数量
func (r *ApprovalRepository) GetMemberCount(babyID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.MembershipApproval{}).
		Where("baby_id = ? AND status = ?", babyID, models.ApprovalStatusApproved).
		Count(&count).Error
	return count, err
}

// HasApprovedRole 检查用户是否已被批准该角色的成员
func (r *ApprovalRepository) HasApprovedRole(userID, babyID uint, role string) (bool, error) {
	var count int64
	err := r.db.Model(&models.MembershipApproval{}).
		Where("user_id = ? AND baby_id = ? AND role = ? AND status = ?",
			userID, babyID, role, models.ApprovalStatusApproved).
		Count(&count).Error
	return count > 0, err
}

// HasAnyApprovedRole 检查用户是否已被批准该宝宝下的任何角色
func (r *ApprovalRepository) HasAnyApprovedRole(userID, babyID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.MembershipApproval{}).
		Where("user_id = ? AND baby_id = ? AND status = ?",
			userID, babyID, models.ApprovalStatusApproved).
		Count(&count).Error
	return count > 0, err
}
