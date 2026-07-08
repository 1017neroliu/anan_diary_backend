package repository

import (
	"anan-diary/ad-backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Create 创建权限记录
func (r *PermissionRepository) Create(permission *models.UserPermission) error {
	return r.db.Create(permission).Error
}

// GetByUserAndBaby 获取用户对宝宝的所有权限
func (r *PermissionRepository) GetByUserAndBaby(userID, babyID uint) ([]models.UserPermission, error) {
	var permissions []models.UserPermission
	err := r.db.Where("user_id = ? AND baby_id = ?", userID, babyID).Find(&permissions).Error
	return permissions, err
}

// GetPermission 获取用户对宝宝特定类型的权限
func (r *PermissionRepository) GetPermission(userID, babyID uint, permissionType string) (bool, error) {
	var permission models.UserPermission
	err := r.db.Where("user_id = ? AND baby_id = ? AND permission_type = ?",
		userID, babyID, permissionType).
		First(&permission).Error
	if err != nil {
		return false, err
	}
	return permission.PermissionValue, nil
}

// HasPermission 检查用户是否拥有特定权限
// 权限记录不存在时返回 (false, nil)——"没有该权限记录"是正常状态，不是错误，
// 否则上层会把 ErrRecordNotFound 当成 500 返回。
func (r *PermissionRepository) HasPermission(userID, babyID uint, permissionType string) (bool, error) {
	has, err := r.GetPermission(userID, babyID, permissionType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return has, nil
}

// SetPermission 设置或更新权限
func (r *PermissionRepository) SetPermission(permission *models.UserPermission) error {
	// 检查是否已存在该权限记录
	var existing models.UserPermission
	err := r.db.Where("user_id = ? AND baby_id = ? AND permission_type = ?",
		permission.UserID, permission.BabyID, permission.PermissionType).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在则创建
		return r.Create(permission)
	} else if err != nil {
		return err
	}

	// 存在则更新
	existing.PermissionValue = permission.PermissionValue
	return r.db.Save(&existing).Error
}

// BatchSetPermissions 批量设置权限
func (r *PermissionRepository) BatchSetPermissions(permissions []*models.UserPermission) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, perm := range permissions {
			if err := r.SetPermission(perm); err != nil {
				return err
			}
		}
		return nil
	})
}

// RemovePermission 移除权限
func (r *PermissionRepository) RemovePermission(userID, babyID uint, permissionType string) error {
	return r.db.Where("user_id = ? AND baby_id = ? AND permission_type = ?",
		userID, babyID, permissionType).Delete(&models.UserPermission{}).Error
}

// RemoveUserPermissions 移除用户的所有权限
func (r *PermissionRepository) RemoveUserPermissions(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.UserPermission{}).Error
}

// RemoveBabyPermissions 移除宝宝的所有权限
func (r *PermissionRepository) RemoveBabyPermissions(babyID uint) error {
	return r.db.Where("baby_id = ?", babyID).Delete(&models.UserPermission{}).Error
}

// GrantFamilyAdminPermission 为宝宝的所有创建者授予管理员权限
func (r *PermissionRepository) GrantFamilyAdminPermission(babyID uint) error {
	// 获取宝宝的创建者
	var baby models.Baby
	if err := r.db.First(&baby, babyID).Error; err != nil {
		return err
	}

	permissions := []*models.UserPermission{
		{
			UserID:         baby.UserID,
			BabyID:         babyID,
			PermissionType: models.PermissionManageFamily,
			PermissionValue: true,
		},
		{
			UserID:         baby.UserID,
			BabyID:         babyID,
			PermissionType: models.PermissionViewRecords,
			PermissionValue: true,
		},
		{
			UserID:         baby.UserID,
			BabyID:         babyID,
			PermissionType: models.PermissionDeleteRecords,
			PermissionValue: true,
		},
		{
			UserID:         baby.UserID,
			BabyID:         babyID,
			PermissionType: models.PermissionExportData,
			PermissionValue: true,
		},
	}

	return r.BatchSetPermissions(permissions)
}

// GetFamilyMembersWithPermissions 获取宝宝的所有家庭成员及其权限
func (r *PermissionRepository) GetFamilyMembersWithPermissions(babyID uint) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// 获取所有已批准的成员
	var approvals []models.MembershipApproval
	err := r.db.Where("baby_id = ? AND status = ?", babyID, models.ApprovalStatusApproved).
		Preload("User").
		Find(&approvals).Error
	if err != nil {
		return nil, err
	}

	// 获取权限
	for _, approval := range approvals {
		permissions, err := r.GetByUserAndBaby(approval.UserID, babyID)
		if err != nil {
			continue
		}

		permMap := make(map[string]bool)
		for _, perm := range permissions {
			permMap[perm.PermissionType] = perm.PermissionValue
		}

		results = append(results, map[string]interface{}{
			"member_id":        approval.ID,
			"user_id":          approval.UserID,
			"name":             approval.User.Name,
			"role":             approval.Role,
			"permissions":      permMap,
			"created_at":       approval.CreatedAt,
		})
	}

	return results, nil
}

// HasAdminPermission 检查用户是否是宝宝的管理员
func (r *PermissionRepository) HasAdminPermission(userID, babyID uint) (bool, error) {
	return r.HasPermission(userID, babyID, models.PermissionManageFamily)
}

// HasRecordDeletePermission 检查用户是否可以删除记录
func (r *PermissionRepository) HasRecordDeletePermission(userID, babyID uint) (bool, error) {
	return r.HasPermission(userID, babyID, models.PermissionDeleteRecords)
}

// HasRecordExportPermission 检查用户是否可以导出数据
func (r *PermissionRepository) HasRecordExportPermission(userID, babyID uint) (bool, error) {
	return r.HasPermission(userID, babyID, models.PermissionExportData)
}

// IsMember 检查用户是否是宝宝的家庭成员
func (r *PermissionRepository) IsMember(userID, babyID uint) (bool, error) {
	hasAdmin, _ := r.HasAdminPermission(userID, babyID)
	if hasAdmin {
		return true, nil
	}

	// 检查是否有查看权限
	hasView, _ := r.HasPermission(userID, babyID, models.PermissionViewRecords)
	return hasView, nil
}
