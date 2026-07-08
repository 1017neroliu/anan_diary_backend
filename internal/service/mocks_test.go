package service

import (
	"errors"

	"anan-diary/ad-backend/internal/models"
)

// ===== 手写 mock，仅用于 authz_service_test.go =====
// 这些 mock 实现 internal/repository/interfaces 中的接口，
// 通过字段控制返回值，方便在测试中精确编排场景。

// errMockSentinel 测试用错误哨兵
var errMockSentinel = errors.New("mock repo error")

// --- BabyRepository mock ---

type mockBabyRepo struct {
	baby     *models.Baby
	err      error
	calledID uint
}

func (m *mockBabyRepo) Create(baby *models.Baby) error                { return nil }
func (m *mockBabyRepo) GetByID(id uint) (*models.Baby, error) {
	m.calledID = id
	if m.err != nil {
		return nil, m.err
	}
	return m.baby, nil
}
func (m *mockBabyRepo) GetByUserID(userID uint) ([]models.Baby, error) { return nil, nil }
func (m *mockBabyRepo) GetByBabyID(babyID string) (*models.Baby, error) { return nil, nil }
func (m *mockBabyRepo) Delete(id uint) error                           { return nil }
func (m *mockBabyRepo) Update(baby *models.Baby) error                 { return nil }

// --- FamilyMemberRepository mock ---

type mockMemberRepo struct {
	members []models.FamilyMember
	err     error
}

func (m *mockMemberRepo) Create(member *models.FamilyMember) error          { return nil }
func (m *mockMemberRepo) GetByBabyID(babyID uint) ([]models.FamilyMember, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.members, nil
}
func (m *mockMemberRepo) CountByBabyID(babyID uint) (int64, error) { return 0, nil }
func (m *mockMemberRepo) RemoveByUserAndBaby(userID, babyID uint) error {
	return nil
}

// --- PermissionRepository mock ---

type mockPermissionRepo struct {
	perms              []models.UserPermission
	getErr             error
	hasAdmin           bool
	hasAdminErr        error
	hasDelete          bool
	hasDeleteErr       error
	hasExport          bool
	hasExportErr       error
}

func (m *mockPermissionRepo) Create(permission *models.UserPermission) error { return nil }
func (m *mockPermissionRepo) GetByUserAndBaby(userID, babyID uint) ([]models.UserPermission, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.perms, nil
}
func (m *mockPermissionRepo) GetPermission(userID, babyID uint, permissionType string) (bool, error) {
	return false, nil
}
func (m *mockPermissionRepo) HasPermission(userID, babyID uint, permissionType string) (bool, error) {
	return false, nil
}
func (m *mockPermissionRepo) SetPermission(permission *models.UserPermission) error { return nil }
func (m *mockPermissionRepo) BatchSetPermissions(permissions []*models.UserPermission) error {
	return nil
}
func (m *mockPermissionRepo) RemovePermission(userID, babyID uint, permissionType string) error {
	return nil
}
func (m *mockPermissionRepo) RemoveUserPermissions(userID uint) error { return nil }
func (m *mockPermissionRepo) RemoveBabyPermissions(babyID uint) error { return nil }
func (m *mockPermissionRepo) GrantFamilyAdminPermission(babyID uint) error {
	return nil
}
func (m *mockPermissionRepo) HasAdminPermission(userID, babyID uint) (bool, error) {
	if m.hasAdminErr != nil {
		return false, m.hasAdminErr
	}
	return m.hasAdmin, nil
}
func (m *mockPermissionRepo) HasRecordDeletePermission(userID, babyID uint) (bool, error) {
	if m.hasDeleteErr != nil {
		return false, m.hasDeleteErr
	}
	return m.hasDelete, nil
}
func (m *mockPermissionRepo) HasRecordExportPermission(userID, babyID uint) (bool, error) {
	if m.hasExportErr != nil {
		return false, m.hasExportErr
	}
	return m.hasExport, nil
}