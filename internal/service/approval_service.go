package service

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository/interfaces"
)

// ApprovalService 审批业务逻辑
type ApprovalService struct {
	approvalRepo   interfaces.ApprovalRepository
	memberRepo     interfaces.FamilyMemberRepository
	permissionRepo interfaces.PermissionRepository
	authz          *AuthzService
}

// NewApprovalService 创建审批服务
func NewApprovalService(
	approvalRepo interfaces.ApprovalRepository,
	memberRepo interfaces.FamilyMemberRepository,
	permissionRepo interfaces.PermissionRepository,
	authz *AuthzService,
) *ApprovalService {
	return &ApprovalService{
		approvalRepo:   approvalRepo,
		memberRepo:     memberRepo,
		permissionRepo: permissionRepo,
		authz:          authz,
	}
}

// ListPending 列出宝宝的待审批申请（仅管理员）
func (s *ApprovalService) ListPending(userID, babyID uint) ([]models.MembershipApproval, error) {
	if err := s.authz.RequireAdmin(userID, babyID); err != nil {
		return nil, err
	}
	return s.approvalRepo.GetPendingByBabyID(babyID)
}

// GetDetail 获取审批详情（管理员可查）
func (s *ApprovalService) GetDetail(userID, approvalID uint) (*models.MembershipApproval, error) {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := s.authz.RequireAdmin(userID, approval.BabyID); err != nil {
		return nil, err
	}
	return approval, nil
}

// Approve 批准成员申请
func (s *ApprovalService) Approve(userID, approvalID uint, notes string) error {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return ErrNotFound
	}

	// 权限校验：管理员
	if err := s.authz.RequireAdmin(userID, approval.BabyID); err != nil {
		return err
	}

	// 状态校验
	if approval.Status != models.ApprovalStatusPending {
		return ErrValidation("该申请已被处理")
	}

	// 批准申请
	if err := s.approvalRepo.Approve(approvalID, userID, notes); err != nil {
		return ErrInternal("审批失败")
	}

	// 创建家庭成员记录
	member := &models.FamilyMember{
		BabyID: approval.BabyID,
		UserID: &approval.UserID,
		Name:   approval.Name,
		Role:   approval.Role,
	}
	if err := s.memberRepo.Create(member); err != nil {
		return ErrInternal("创建家庭成员失败")
	}

	// 授予权限（干净的单组权限，避免重复）
	permissions := []*models.UserPermission{
		{
			UserID:          approval.UserID,
			BabyID:          approval.BabyID,
			PermissionType:  models.PermissionViewRecords,
			PermissionValue: true,
		},
		{
			UserID:          approval.UserID,
			BabyID:          approval.BabyID,
			PermissionType:  models.PermissionPostComment,
			PermissionValue: true,
		},
		{
			UserID:          approval.UserID,
			BabyID:          approval.BabyID,
			PermissionType:  models.PermissionLikeRecord,
			PermissionValue: true,
		},
	}
	if err := s.permissionRepo.BatchSetPermissions(permissions); err != nil {
		return ErrInternal("设置权限失败")
	}

	return nil
}

// Reject 拒绝成员申请
func (s *ApprovalService) Reject(userID, approvalID uint, notes string) error {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return ErrNotFound
	}

	if err := s.authz.RequireAdmin(userID, approval.BabyID); err != nil {
		return err
	}

	if approval.Status != models.ApprovalStatusPending {
		return ErrValidation("该申请已被处理")
	}

	if err := s.approvalRepo.Reject(approvalID, userID, notes); err != nil {
		return ErrInternal("拒绝失败")
	}
	return nil
}

// Cancel 取消自己的申请（仅申请人本人）
func (s *ApprovalService) Cancel(userID, approvalID uint) error {
	approval, err := s.approvalRepo.GetByID(approvalID)
	if err != nil {
		return ErrNotFound
	}

	// 只能取消自己的申请
	if approval.UserID != userID {
		return ErrForbidden
	}

	if approval.Status != models.ApprovalStatusPending {
		return ErrValidation("只能取消待审批的申请")
	}

	if err := s.approvalRepo.Cancel(approvalID, userID); err != nil {
		return ErrInternal("取消申请失败")
	}
	return nil
}

// ListMine 列出我的所有申请
func (s *ApprovalService) ListMine(userID uint) ([]models.MembershipApproval, error) {
	return s.approvalRepo.GetByUser(userID)
}
