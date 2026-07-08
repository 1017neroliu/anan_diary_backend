package service

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository/interfaces"
	"errors"
)

var (
	ErrBabyNotFound  = errors.New("宝宝档案不存在")
	ErrDuplicateRole = errors.New("该宝宝已有该角色的家庭成员")
	ErrFreeLimitFull = errors.New("免费名额已满，请联系管理员")
	ErrNameRequired  = errors.New("姓名不能为空")
	ErrRoleRequired  = errors.New("角色不能为空")
)

// BabyService 宝宝档案业务逻辑
type BabyService struct {
	babyRepo   interfaces.BabyRepository
	memberRepo interfaces.FamilyMemberRepository
	authz      *AuthzService
}

// NewBabyService 创建宝宝服务
func NewBabyService(
	babyRepo interfaces.BabyRepository,
	memberRepo interfaces.FamilyMemberRepository,
	authz *AuthzService,
) *BabyService {
	return &BabyService{
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		authz:      authz,
	}
}

// CreateBabyRequest 创建宝宝请求
type CreateBabyRequest struct {
	Name      string `json:"name" binding:"required"`       // 宝宝昵称
	BirthDate string `json:"birth_date" binding:"required"` // 出生日期
	BabyID    string `json:"baby_id" binding:"required"`    // 出生证编号
	UserRole  string `json:"user_role"`                     // 创建者角色（如爸爸/妈妈）
}

// CreateBabyResult 创建宝宝结果
type CreateBabyResult struct {
	Baby     *models.Baby
	IsNew    bool // true=新建, false=加入现有
	MemberID uint // 创建者作为家庭成员的ID
}

// CreateBaby 创建或加入宝宝档案
func (s *BabyService) CreateBaby(userID uint, req CreateBabyRequest) (*CreateBabyResult, error) {
	if req.Name == "" || req.BirthDate == "" || req.BabyID == "" {
		return nil, ErrValidation("宝宝昵称、出生日期和出生证编号为必填项")
	}

	// 检查是否已存在同名出生证编号的宝宝
	existing, err := s.babyRepo.GetByBabyID(req.BabyID)
	if err == nil && existing != nil {
		// 已存在宝宝，检查当前用户是否已是其中角色
		members, err := s.memberRepo.GetByBabyID(existing.ID)
		if err != nil {
			return nil, ErrInternal("获取家庭成员失败")
		}
		for _, m := range members {
			if m.UserID != nil && *m.UserID == userID && m.Role == req.UserRole {
				return nil, ErrDuplicateRole
			}
		}
		// 添加为家庭成员
		memberName := req.UserRole
		if memberName == "" {
			memberName = "家庭成员"
		}
		member := &models.FamilyMember{
			BabyID: existing.ID,
			UserID: &userID,
			Name:   memberName,
			Role:   req.UserRole,
		}
		if err := s.memberRepo.Create(member); err != nil {
			return nil, ErrInternal("添加家庭成员失败")
		}
		return &CreateBabyResult{Baby: existing, IsNew: false, MemberID: member.ID}, nil
	}

	// 新建宝宝档案
	baby := &models.Baby{
		UserID:    userID,
		Name:      req.Name,
		BirthDate: req.BirthDate,
		BabyID:    req.BabyID,
	}
	if err := s.babyRepo.Create(baby); err != nil {
		return nil, ErrInternal("创建宝宝档案失败")
	}

	// 添加创建者为家庭成员
	memberName := req.UserRole
	if memberName == "" {
		memberName = "爸爸"
	}
	member := &models.FamilyMember{
		BabyID: baby.ID,
		UserID: &userID,
		Name:   memberName,
		Role:   req.UserRole,
	}
	if err := s.memberRepo.Create(member); err != nil {
		return nil, ErrInternal("添加创建者到家庭成员失败")
	}

	return &CreateBabyResult{Baby: baby, IsNew: true, MemberID: member.ID}, nil
}

// GetBabyByID 获取宝宝信息（需验证成员身份）
func (s *BabyService) GetBabyByID(userID, babyID uint) (*models.Baby, error) {
	ok, _ := s.authz.IsMember(userID, babyID)
	if !ok {
		return nil, ErrNotMember
	}
	return s.babyRepo.GetByID(babyID)
}

// DeleteBaby 删除宝宝（仅管理员可操作）
func (s *BabyService) DeleteBaby(userID, babyID uint) error {
	if err := s.authz.RequireAdmin(userID, babyID); err != nil {
		return err
	}
	return s.babyRepo.Delete(babyID)
}
