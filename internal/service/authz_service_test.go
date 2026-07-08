package service

import (
	"testing"

	"anan-diary/ad-backend/internal/models"
)

func makePointer[T any](v T) *T { return &v }

// ==============================
// IsCreator 测试
// ==============================

func TestAuthzService_IsCreator(t *testing.T) {
	babyID := uint(1)
	baby := &models.Baby{ID: babyID, UserID: 10}

	t.Run("创建者应返回 true", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsCreator(10, babyID)
		if err != nil || !ok {
			t.Errorf("IsCreator(10) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("非创建者应返回 false", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsCreator(20, babyID)
		if err != nil || ok {
			t.Errorf("IsCreator(20) = %v, %v; 期望 false, nil", ok, err)
		}
	})

	t.Run("不存在的宝宝应返回 false", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: nil, err: errMockSentinel}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsCreator(10, 999)
		if err != nil || ok {
			t.Errorf("IsCreator 对于不存在宝宝 = %v, %v; 期望 false, nil", ok, err)
		}
	})
}

// ==============================
// IsMember 测试
// ==============================

func TestAuthzService_IsMember(t *testing.T) {
	babyID := uint(1)
	ownerID := uint(10)
	baby := &models.Baby{ID: babyID, UserID: ownerID}

	t.Run("创建者视为成员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsMember(ownerID, babyID)
		if err != nil || !ok {
			t.Errorf("IsMember(创建者) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("拥有权限视为成员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{
			perms: []models.UserPermission{
				{UserID: 20, BabyID: babyID, PermissionType: "view_records", PermissionValue: true},
			},
		}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsMember(20, babyID)
		if err != nil || !ok {
			t.Errorf("IsMember(有权限) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("在 family_members 中视为成员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		memberRepo := &mockMemberRepo{
			members: []models.FamilyMember{
				{BabyID: babyID, UserID: makePointer(uint(30))},
			},
		}
		svc := NewAuthzService(&mockPermissionRepo{}, memberRepo, babyRepo)
		ok, err := svc.IsMember(30, babyID)
		if err != nil || !ok {
			t.Errorf("IsMember(家庭成员) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("陌生人不应视为成员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsMember(99, babyID)
		if err != nil || ok {
			t.Errorf("IsMember(陌生人) = %v, %v; 期望 false, nil", ok, err)
		}
	})

	t.Run("不存在的宝宝不应视为成员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: nil, err: errMockSentinel}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsMember(10, 999)
		if err != nil || ok {
			t.Errorf("IsMember(不存在宝宝) = %v, %v; 期望 false, nil", ok, err)
		}
	})
}

// ==============================
// IsAdmin 测试
// ==============================

func TestAuthzService_IsAdmin(t *testing.T) {
	babyID := uint(1)
	ownerID := uint(10)
	baby := &models.Baby{ID: babyID, UserID: ownerID}

	t.Run("显式 manage_family 权限视为管理员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{hasAdmin: true}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsAdmin(20, babyID)
		if err != nil || !ok {
			t.Errorf("IsAdmin(有 manage_family) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("创建者视为管理员（兜底）", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsAdmin(ownerID, babyID)
		if err != nil || !ok {
			t.Errorf("IsAdmin(创建者) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("无权限非创建者不应视为管理员", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.IsAdmin(99, babyID)
		if err != nil || ok {
			t.Errorf("IsAdmin(无权限) = %v, %v; 期望 false, nil", ok, err)
		}
	})
}

// ==============================
// CanDeleteRecord 测试
// ==============================

func TestAuthzService_CanDeleteRecord(t *testing.T) {
	babyID := uint(1)
	ownerID := uint(10)
	baby := &models.Baby{ID: babyID, UserID: ownerID}

	t.Run("管理员可删除任意记录", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{hasAdmin: true}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanDeleteRecord(20, babyID, 99) // 不是记录作者
		if err != nil || !ok {
			t.Errorf("CanDeleteRecord(管理员) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("有 delete_records 权限可删除", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{hasDelete: true}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanDeleteRecord(30, babyID, 99)
		if err != nil || !ok {
			t.Errorf("CanDeleteRecord(有 delete 权限) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("记录作者可删除自己的记录", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanDeleteRecord(50, babyID, 50)
		if err != nil || !ok {
			t.Errorf("CanDeleteRecord(作者) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("陌生人不能删除", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanDeleteRecord(99, babyID, 50)
		if err != nil || ok {
			t.Errorf("CanDeleteRecord(陌生人) = %v, %v; 期望 false, nil", ok, err)
		}
	})
}

// ==============================
// CanExportData 测试
// ==============================

func TestAuthzService_CanExportData(t *testing.T) {
	babyID := uint(1)
	ownerID := uint(10)
	baby := &models.Baby{ID: babyID, UserID: ownerID}

	t.Run("管理员可以导出", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{hasAdmin: true}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanExportData(20, babyID)
		if err != nil || !ok {
			t.Errorf("CanExportData(管理员) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("有 export_data 权限可以导出", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		permRepo := &mockPermissionRepo{hasExport: true}
		svc := NewAuthzService(permRepo, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanExportData(30, babyID)
		if err != nil || !ok {
			t.Errorf("CanExportData(有 export 权限) = %v, %v; 期望 true, nil", ok, err)
		}
	})

	t.Run("陌生人不能导出", func(t *testing.T) {
		babyRepo := &mockBabyRepo{baby: baby}
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, babyRepo)
		ok, err := svc.CanExportData(99, babyID)
		if err != nil || ok {
			t.Errorf("CanExportData(陌生人) = %v, %v; 期望 false, nil", ok, err)
		}
	})
}

// ==============================
// Require* 断言测试
// ==============================

func TestAuthzService_RequireMember(t *testing.T) {
	babyID := uint(1)
	baby := &models.Baby{ID: babyID, UserID: 10}

	t.Run("成员通过断言", func(t *testing.T) {
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, &mockBabyRepo{baby: baby})
		if err := svc.RequireMember(10, babyID); err != nil {
			t.Errorf("RequireMember(创建者) = %v; 期望 nil", err)
		}
	})

	t.Run("非成员不通过断言", func(t *testing.T) {
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, &mockBabyRepo{baby: baby})
		if err := svc.RequireMember(99, babyID); err != ErrNotMember {
			t.Errorf("RequireMember(陌生人) = %v; 期望 %v", err, ErrNotMember)
		}
	})
}

func TestAuthzService_RequireAdmin(t *testing.T) {
	babyID := uint(1)
	baby := &models.Baby{ID: babyID, UserID: 10}

	t.Run("管理员通过断言", func(t *testing.T) {
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, &mockBabyRepo{baby: baby})
		if err := svc.RequireAdmin(10, babyID); err != nil {
			t.Errorf("RequireAdmin(创建者) = %v; 期望 nil", err)
		}
	})

	t.Run("非管理员不通过断言", func(t *testing.T) {
		svc := NewAuthzService(&mockPermissionRepo{}, &mockMemberRepo{}, &mockBabyRepo{baby: baby})
		if err := svc.RequireAdmin(99, babyID); err != ErrNotAdmin {
			t.Errorf("RequireAdmin(非管理员) = %v; 期望 %v", err, ErrNotAdmin)
		}
	})
}