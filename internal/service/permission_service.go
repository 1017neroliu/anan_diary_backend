package service

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository/interfaces"
)

// PermissionService 权限管理业务逻辑
type PermissionService struct {
	permissionRepo interfaces.PermissionRepository
	approvalRepo   interfaces.ApprovalRepository
	memberRepo     interfaces.FamilyMemberRepository
	babyRepo       interfaces.BabyRepository
	authz          *AuthzService
}

// NewPermissionService 创建权限服务
func NewPermissionService(
	permissionRepo interfaces.PermissionRepository,
	approvalRepo interfaces.ApprovalRepository,
	memberRepo interfaces.FamilyMemberRepository,
	babyRepo interfaces.BabyRepository,
	authz *AuthzService,
) *PermissionService {
	return &PermissionService{
		permissionRepo: permissionRepo,
		approvalRepo:   approvalRepo,
		memberRepo:     memberRepo,
		babyRepo:       babyRepo,
		authz:          authz,
	}
}

// GetMyPermissions 获取我的权限
func (s *PermissionService) GetMyPermissions(userID, babyID uint) ([]models.UserPermission, error) {
	return s.permissionRepo.GetByUserAndBaby(userID, babyID)
}

// SetUserPermission 管理员设置某用户的权限
func (s *PermissionService) SetUserPermission(adminID, targetUserID, babyID uint, permType string, value bool) error {
	if err := s.authz.RequireAdmin(adminID, babyID); err != nil {
		return err
	}
	perm := &models.UserPermission{
		UserID:          targetUserID,
		BabyID:          babyID,
		PermissionType:  permType,
		PermissionValue: value,
	}
	return s.permissionRepo.SetPermission(perm)
}

// RemoveUserPermission 管理员移除某用户的某项权限
func (s *PermissionService) RemoveUserPermission(adminID, targetUserID, babyID uint, permType string) error {
	if err := s.authz.RequireAdmin(adminID, babyID); err != nil {
		return err
	}
	return s.permissionRepo.RemovePermission(targetUserID, babyID, permType)
}

// GrantAdmin 授予管理员权限（从路径参数获取 targetUserID）
func (s *PermissionService) GrantAdmin(adminID, targetUserID, babyID uint) error {
	if err := s.authz.RequireAdmin(adminID, babyID); err != nil {
		return err
	}

	permissions := []*models.UserPermission{
		{UserID: targetUserID, BabyID: babyID, PermissionType: models.PermissionManageFamily, PermissionValue: true},
		{UserID: targetUserID, BabyID: babyID, PermissionType: models.PermissionViewRecords, PermissionValue: true},
		{UserID: targetUserID, BabyID: babyID, PermissionType: models.PermissionDeleteRecords, PermissionValue: true},
		{UserID: targetUserID, BabyID: babyID, PermissionType: models.PermissionExportData, PermissionValue: true},
	}
	return s.permissionRepo.BatchSetPermissions(permissions)
}

// RemoveMember 移除家庭成员（管理员可操作，不能移除创建者）
func (s *PermissionService) RemoveMember(adminID, targetUserID, babyID uint) error {
	if err := s.authz.RequireAdmin(adminID, babyID); err != nil {
		return err
	}

	// 不能移除创建者
	baby, err := s.babyRepo.GetByID(babyID)
	if err != nil {
		return ErrNotFound
	}
	if baby.UserID == targetUserID {
		return ErrValidation("不能移除宝宝创建者")
	}

	// 移除该用户在指定宝宝下的所有权限
	perms, err := s.permissionRepo.GetByUserAndBaby(targetUserID, babyID)
	if err == nil {
		for _, p := range perms {
			_ = s.permissionRepo.RemovePermission(targetUserID, babyID, p.PermissionType)
		}
	}

	// 移除 family_members 记录
	if err := s.memberRepo.RemoveByUserAndBaby(targetUserID, babyID); err != nil {
		return ErrInternal("移除家庭成员失败")
	}

	return nil
}

// ListFamilyMembers 获取已批准的家庭成员列表（管理员可查看）
func (s *PermissionService) ListFamilyMembers(userID, babyID uint) ([]models.MembershipApproval, error) {
	if err := s.authz.RequireAdmin(userID, babyID); err != nil {
		return nil, err
	}
	return s.approvalRepo.GetApprovedMembers(babyID)
}

// CheckPermission 检查权限
func (s *PermissionService) CheckPermission(userID, babyID uint, permType string) (bool, error) {
	return s.permissionRepo.HasPermission(userID, babyID, permType)
}

// CheckMembership 检查是否是成员
func (s *PermissionService) CheckMembership(userID, babyID uint) (bool, error) {
	return s.authz.IsMember(userID, babyID)
}
