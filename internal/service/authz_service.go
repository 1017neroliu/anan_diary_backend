package service

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository/interfaces"
)

// AuthzService 授权服务 — 所有权限判断的唯一来源
type AuthzService struct {
	permissionRepo interfaces.PermissionRepository
	memberRepo     interfaces.FamilyMemberRepository
	babyRepo       interfaces.BabyRepository
}

// NewAuthzService 创建授权服务
func NewAuthzService(
	permissionRepo interfaces.PermissionRepository,
	memberRepo interfaces.FamilyMemberRepository,
	babyRepo interfaces.BabyRepository,
) *AuthzService {
	return &AuthzService{
		permissionRepo: permissionRepo,
		memberRepo:     memberRepo,
		babyRepo:       babyRepo,
	}
}

// getCreatorUserID 获取宝宝的创建者 user_id（私有方法）
func (s *AuthzService) getCreatorUserID(babyID uint) (uint, error) {
	baby, err := s.babyRepo.GetByID(babyID)
	if err != nil {
		return 0, err
	}
	return baby.UserID, nil
}

// IsCreator 判断用户是否是宝宝的创建者
func (s *AuthzService) IsCreator(userID, babyID uint) (bool, error) {
	creatorID, err := s.getCreatorUserID(babyID)
	if err != nil {
		return false, nil // 宝宝不存在时安全返回 false
	}
	return creatorID == userID, nil
}

// IsMember 判断用户是否是宝宝的家庭成员
func (s *AuthzService) IsMember(userID, babyID uint) (bool, error) {
	// 1. 创建者视为成员
	isCreator, err := s.IsCreator(userID, babyID)
	if err != nil {
		return false, nil
	}
	if isCreator {
		return true, nil
	}

	// 2. 拥有任意权限视为成员
	perms, err := s.permissionRepo.GetByUserAndBaby(userID, babyID)
	if err == nil && len(perms) > 0 {
		return true, nil
	}

	// 3. 在 family_members 表中
	members, err := s.memberRepo.GetByBabyID(babyID)
	if err != nil {
		return false, nil
	}
	for _, m := range members {
		if m.UserID != nil && *m.UserID == userID {
			return true, nil
		}
	}

	return false, nil
}

// IsAdmin 判断用户是否是宝宝的管理员
func (s *AuthzService) IsAdmin(userID, babyID uint) (bool, error) {
	// 1. 权限表显式授权
	hasManage, err := s.permissionRepo.HasAdminPermission(userID, babyID)
	if err == nil && hasManage {
		return true, nil
	}

	// 2. 创建者兜底（兼容旧数据）
	isCreator, err := s.IsCreator(userID, babyID)
	if err != nil {
		return false, nil
	}
	return isCreator, nil
}

// CanCreateRecord 判断用户能否为宝宝创建成长记录
func (s *AuthzService) CanCreateRecord(userID, babyID uint) (bool, error) {
	return s.IsMember(userID, babyID)
}

// CanDeleteRecord 判断用户能否删除指定成长记录
func (s *AuthzService) CanDeleteRecord(userID, babyID, recordOwnerUserID uint) (bool, error) {
	// 1. 管理员可删
	isAdmin, err := s.IsAdmin(userID, babyID)
	if err == nil && isAdmin {
		return true, nil
	}

	// 2. 拥有 delete_records 权限可删
	canDelete, err := s.permissionRepo.HasRecordDeletePermission(userID, babyID)
	if err == nil && canDelete {
		return true, nil
	}

	// 3. 记录作者可删自己的记录
	return recordOwnerUserID == userID, nil
}

// CanManageFamily 判断用户能否管理家庭
func (s *AuthzService) CanManageFamily(userID, babyID uint) (bool, error) {
	return s.IsAdmin(userID, babyID)
}

// CanExportData 判断用户能否导出数据
func (s *AuthzService) CanExportData(userID, babyID uint) (bool, error) {
	isAdmin, err := s.IsAdmin(userID, babyID)
	if err == nil && isAdmin {
		return true, nil
	}
	return s.permissionRepo.HasRecordExportPermission(userID, babyID)
}

// ========== 断言方法 ==========

// RequireMember 断言用户是成员
func (s *AuthzService) RequireMember(userID, babyID uint) error {
	ok, _ := s.IsMember(userID, babyID)
	if !ok {
		return ErrNotMember
	}
	return nil
}

// RequireAdmin 断言用户是管理员
func (s *AuthzService) RequireAdmin(userID, babyID uint) error {
	ok, _ := s.IsAdmin(userID, babyID)
	if !ok {
		return ErrNotAdmin
	}
	return nil
}

// RequireManageFamily 断言用户可管理家庭
func (s *AuthzService) RequireManageFamily(userID, babyID uint) error {
	ok, _ := s.CanManageFamily(userID, babyID)
	if !ok {
		return ErrForbidden
	}
	return nil
}

// 确保编译时不会因未使用 models 报错
var _ = models.PermissionManageFamily
