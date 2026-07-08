// Package authz �供统一的授权判断服务。
//
// 这是后端权限判断的唯一来源。所有业务权限决策（是否是成员、管理员、
// 能否创建/删除记录、能否管理家庭等）都应通过 Service 调用本包完成，
// 而不是在各 handler 中直接查询数据库或判断 babies.user_id。
//
// 权限模型（经用户确认）：
//   - babies.user_id 仅作为只读的"创建者"信息保留，不再用于权限判断。
//   - family_members 表表示家庭成员关系。
//   - user_permissions 表存储细粒度权限位。
//   - "管理员" = 拥有 manage_family 权限 或 是宝宝创建者（兜底兼容旧数据）。
//   - "成员"   = 拥有任何该宝宝相关权限，或属于 family_members。
package authz

import (
	"errors"

	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
)

// 定义授权相关错误，便于 handler 层映射为合适的 HTTP 响应。
var (
	ErrNotMember        = errors.New("not a family member")
	ErrNotAdmin         = errors.New("not a family admin")
	ErrForbidden        = errors.New("forbidden")
	ErrResourceNotFound = errors.New("resource not found")
)

// Service 授权服务
type Service struct {
	permissionRepo *repository.PermissionRepository
	memberRepo     *repository.FamilyMemberRepository
	babyRepo       *repository.BabyRepository
}

// NewService 创建授权服务
func NewService(
	permissionRepo *repository.PermissionRepository,
	memberRepo *repository.FamilyMemberRepository,
	babyRepo *repository.BabyRepository,
) *Service {
	return &Service{
		permissionRepo: permissionRepo,
		memberRepo:     memberRepo,
		babyRepo:       babyRepo,
	}
}

// IsCreator 判断用户是否是宝宝的创建者（babies.user_id）。
// 注意：创建者身份仅作展示与兜底，不作为唯一管理员依据。
func (s *Service) IsCreator(userID, babyID uint) (bool, error) {
	baby, err := s.babyRepo.GetByID(babyID)
	if err != nil {
		return false, err
	}
	return baby.UserID == userID, nil
}

// IsMember 判断用户是否是宝宝的家庭成员。
// 创建者、拥有 manage_family 权限、拥有 view_records 权限，或在 family_members 中，均视为成员。
func (s *Service) IsMember(userID, babyID uint) (bool, error) {
	// 1. 创建者视为成员
	isCreator, err := s.IsCreator(userID, babyID)
	if err != nil {
		// 宝宝不存在时，记为非成员而非报错，避免泄露存在性
		return false, nil
	}
	if isCreator {
		return true, nil
	}

	// 2. 拥有任意该宝宝权限视为成员
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

// IsAdmin 判断用户是否是宝宝的管理员。
// 管理员 = 拥有 manage_family 权限，或是创建者（兜底）。
func (s *Service) IsAdmin(userID, babyID uint) (bool, error) {
	// 1. 权限表显式授权
	hasManage, err := s.permissionRepo.HasAdminPermission(userID, babyID)
	if err == nil && hasManage {
		return true, nil
	}

	// 2. 创建者兜底（兼容未在权限表登记的旧数据）
	return s.IsCreator(userID, babyID)
}

// CanCreateRecord 判断用户能否为宝宝创建成长记录。
// 成员即可创建（成员身份由 IsMember 判定）。
func (s *Service) CanCreateRecord(userID, babyID uint) (bool, error) {
	return s.IsMember(userID, babyID)
}

// CanDeleteRecord 判断用户能否删除指定成长记录。
// 记录作者、拥有 delete_records 权限，或管理员均可删除。
// growthRepo 用于读取记录的作者。
type recordOwnerLookup func(recordID uint) (uint, error) // 返回记录的 user_id

// CanDeleteRecordByOwner 通过传入的记录作者查找函数判断删除权限。
func (s *Service) CanDeleteRecordByOwner(userID, babyID uint, lookup recordOwnerLookup, recordID uint) (bool, error) {
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
	ownerID, err := lookup(recordID)
	if err != nil {
		return false, nil
	}
	return ownerID == userID, nil
}

// CanManageFamily 判断用户能否管理家庭（审批/移除成员、邀请码等）。
// 管理员才可管理家庭。
func (s *Service) CanManageFamily(userID, babyID uint) (bool, error) {
	return s.IsAdmin(userID, babyID)
}

// CanExportData 判断用户能否导出宝宝数据。
func (s *Service) CanExportData(userID, babyID uint) (bool, error) {
	// 管理员可导出
	isAdmin, err := s.IsAdmin(userID, babyID)
	if err == nil && isAdmin {
		return true, nil
	}
	// 拥有 export_data 权限可导出
	return s.permissionRepo.HasRecordExportPermission(userID, babyID)
}

// RequireMember 断言用户是成员，否则返回 ErrNotMember。
func (s *Service) RequireMember(userID, babyID uint) error {
	ok, _ := s.IsMember(userID, babyID)
	if !ok {
		return ErrNotMember
	}
	return nil
}

// RequireAdmin 断言用户是管理员，否则返回 ErrNotAdmin。
func (s *Service) RequireAdmin(userID, babyID uint) error {
	ok, _ := s.IsAdmin(userID, babyID)
	if !ok {
		return ErrNotAdmin
	}
	return nil
}

// RequireManageFamily 断言用户可管理家庭。
func (s *Service) RequireManageFamily(userID, babyID uint) error {
	ok, _ := s.CanManageFamily(userID, babyID)
	if !ok {
		return ErrForbidden
	}
	return nil
}

// 避免未使用导入告警（models 暂未直接使用，保留以便后续扩展）
var _ = models.PermissionManageFamily
