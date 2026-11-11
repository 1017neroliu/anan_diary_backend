// Package interfaces 定义 Repository 接口层，便于测试 mock 与依赖注入。
package interfaces

import "anan-diary/ad-backend/internal/models"

//go:generate mockgen -destination=../mocks/mock_repos.go -package=mocks anan-diary/ad-backend/internal/repository/interfaces BabyRepository,FamilyMemberRepository,GrowthRecordRepository,PermissionRepository,ApprovalRepository,InvitationRepository,CommentRepository,LikeRepository,StatsRepository,SubscriptionRepository,UserRepository

// BabyRepository 宝宝数据访问接口
type BabyRepository interface {
	Create(baby *models.Baby) error
	GetByID(id uint) (*models.Baby, error)
	GetByUserID(userID uint) ([]models.Baby, error)
	GetByBabyID(babyID string) (*models.Baby, error)
	Delete(id uint) error
	Update(baby *models.Baby) error
}

// FamilyMemberRepository 家庭成员数据访问接口
type FamilyMemberRepository interface {
	Create(member *models.FamilyMember) error
	GetByBabyID(babyID uint) ([]models.FamilyMember, error)
	CountByBabyID(babyID uint) (int64, error)
	RemoveByUserAndBaby(userID, babyID uint) error
}

// GrowthRecordRepository 成长记录数据访问接口
type GrowthRecordRepository interface {
	Create(record *models.GrowthRecord) error
	GetByID(id uint) (*models.GrowthRecord, error)
	GetByBabyID(babyID uint) ([]models.GrowthRecord, error)
	GetByBabyAndDate(babyID uint, date string) (*models.GrowthRecord, error)
	Update(id uint, content, tags, media string) error
	Delete(id uint) error
}

// PermissionRepository 权限数据访问接口
type PermissionRepository interface {
	Create(permission *models.UserPermission) error
	GetByUserAndBaby(userID, babyID uint) ([]models.UserPermission, error)
	GetPermission(userID, babyID uint, permissionType string) (bool, error)
	HasPermission(userID, babyID uint, permissionType string) (bool, error)
	SetPermission(permission *models.UserPermission) error
	BatchSetPermissions(permissions []*models.UserPermission) error
	RemovePermission(userID, babyID uint, permissionType string) error
	RemoveUserPermissions(userID uint) error
	RemoveBabyPermissions(babyID uint) error
	GrantFamilyAdminPermission(babyID uint) error
	HasAdminPermission(userID, babyID uint) (bool, error)
	HasRecordDeletePermission(userID, babyID uint) (bool, error)
	HasRecordExportPermission(userID, babyID uint) (bool, error)
}

// ApprovalRepository 审批数据访问接口
type ApprovalRepository interface {
	Create(approval *models.MembershipApproval) error
	GetByID(id uint) (*models.MembershipApproval, error)
	GetPendingByBabyID(babyID uint) ([]models.MembershipApproval, error)
	GetByUserAndBaby(userID, babyID uint, status string) (*models.MembershipApproval, error)
	GetByUserAndBabyRole(userID, babyID uint, role string) (*models.MembershipApproval, error)
	GetByUser(userID uint) ([]models.MembershipApproval, error)
	Update(approval *models.MembershipApproval) error
	Approve(approvalID, reviewerID uint, notes string) error
	Reject(approvalID, reviewerID uint, notes string) error
	Cancel(approvalID, userID uint) error
	GetApprovedMembers(babyID uint) ([]models.MembershipApproval, error)
	GetMemberCount(babyID uint) (int64, error)
	HasApprovedRole(userID, babyID uint, role string) (bool, error)
	HasAnyApprovedRole(userID, babyID uint) (bool, error)
}

// InvitationRepository 邀请码数据访问接口
type InvitationRepository interface {
	Create(invitation *models.Invitation) error
	GetByCode(code string) (*models.Invitation, error)
	GetByBabyID(babyID uint) ([]models.Invitation, error)
	GetActiveCode(babyID uint) (*models.Invitation, error)
	Update(invitation *models.Invitation) error
	UpdateStatus(babyID uint, status string) error
	CheckInvitationUsed(invitationID uint) (bool, error)
	RecordInvitationUsage(invitationID, userID uint, ipAddress string) error
	GetInvitationUsageCount(invitationID uint) (int64, error)
}

// CommentRepository 评论数据访问接口
type CommentRepository interface {
	Create(comment *models.Comment) error
	GetByID(id uint) (*models.Comment, error)
	GetByGrowthRecord(growthRecordID uint) ([]models.Comment, error)
	GetByUser(userID uint) ([]models.Comment, error)
	Update(comment *models.Comment) error
	Delete(id uint) error
	GetCommentCount(growthRecordID uint) (int64, error)
}

// LikeRepository 点赞数据访问接口
type LikeRepository interface {
	Create(like *models.Like) error
	Delete(growthRecordID, userID uint) error
	GetByUser(userID uint) ([]models.Like, error)
	GetByGrowthRecord(growthRecordID uint) ([]models.Like, error)
	GetLikeCount(growthRecordID uint) (int64, error)
	HasLiked(growthRecordID, userID uint) (bool, error)
	GetTopLikedRecords(babyID uint, limit int) ([]models.GrowthRecord, error)
}

// StatsRepository 用户统计数据访问接口
type StatsRepository interface {
	GetOrCreate(userID uint) (*models.UserStats, error)
	IncrementRecordCount(userID uint) error
	DecrementRecordCount(userID uint) error
	IncrementLikeCount(userID uint) error
	DecrementLikeCount(userID uint) error
	IncrementCommentCount(userID uint) error
	DecrementCommentCount(userID uint) error
	UpdateStorageUsed(userID uint, sizeBytes int64) error
	GetStats(userID uint) (*models.UserStats, error)
}

// SubscriptionRepository 订阅数据访问接口
type SubscriptionRepository interface {
	Create(subscription *models.Subscription) error
	GetActiveByUserID(userID uint) (*models.Subscription, error)
	GetByUserID(userID uint) ([]models.Subscription, error)
	GetStorageQuota(plan string) (int64, error)
	GetMaxMembers(plan string) (int64, error)
	HasPDFExport(plan string) (bool, error)
	CheckSubscriptionStatus(userID uint) (bool, bool, error)
}

// UserRepository 用户数据访问接口
type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
}
