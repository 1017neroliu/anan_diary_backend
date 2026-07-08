package models

import (
	"time"
)

// Invitation 邀请码模型
type Invitation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"uniqueIndex;size:50;not null" json:"code"`              // 邀请码
	BabyID    uint      `gorm:"not null;index" json:"baby_id"`                         // 关联的宝宝ID
	CreatedBy uint      `gorm:"index;not null" json:"created_by"`                      // 创建者用户ID
	Status    string    `gorm:"size:20;not null;default:'active';index" json:"status"` // active/expired/revoked
	MaxUses   int       `gorm:"default:10" json:"max_uses"`                            // 最大使用次数
	UsedCount int       `gorm:"default:0" json:"used_count"`                           // 已使用次数
	Reason    string    `gorm:"type:text;column:description" json:"reason,omitempty"`  // 邀请说明
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	Creator User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Baby    Baby `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
}

// TableName 指定表名
func (Invitation) TableName() string {
	return "invitations"
}

// InviteUse 邀请码使用记录
type InviteUse struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	InvitationID uint      `gorm:"index;not null" json:"invitation_id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	IPAddress    string    `gorm:"type:inet" json:"ip_address,omitempty"` // IP地址
	UsedAt       time.Time `gorm:"autoCreateTime" json:"used_at"`
}

// TableName 指定表名
func (InviteUse) TableName() string {
	return "invitation_uses"
}

// MembershipApproval 家庭成员审批模型
type MembershipApproval struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	BabyID     uint       `gorm:"index;not null" json:"baby_id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Role       string     `gorm:"size:20;not null;index" json:"role"` // 爸爸/妈妈/爷爷/奶奶/姥姥/姥爷/哥哥/姐姐
	Name       string     `gorm:"size:100;not null" json:"name"`
	Reason     string     `gorm:"type:text" json:"reason,omitempty"`
	Status     string     `gorm:"size:20;not null;default:'pending';index" json:"status"` // pending/approved/rejected/cancelled
	ReviewedBy *uint      `gorm:"index" json:"reviewed_by,omitempty"`                     // 审批人ID
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`                                  // 审批时间
	Notes      string     `gorm:"type:text" json:"notes,omitempty"`                       // 审批备注
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// 关联关系
	Baby     Baby                `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
	User     *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Reviewer *User               `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	Parent   *MembershipApproval `gorm:"foreignKey:ID" json:"parent,omitempty"` // 用于查看申请来源
}

// TableName 指定表名
func (MembershipApproval) TableName() string {
	return "membership_approvals"
}

// UserPermission 用户权限模型
type UserPermission struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	BabyID          uint      `gorm:"index;not null" json:"baby_id"`
	PermissionType  string    `gorm:"size:50;not null;index" json:"permission_type"` // manage_family, view_records, delete_records, export_data, manage_invitations, ...
	PermissionValue bool      `gorm:"not null" json:"permission_value"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Baby Baby `gorm:"foreignKey:BabyID" json:"baby,omitempty"`
}

// TableName 指定表名
func (UserPermission) TableName() string {
	return "user_permissions"
}

// Like 成长记录点赞模型
type Like struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	GrowthRecordID uint      `gorm:"index;not null" json:"growth_record_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联关系
	GrowthRecord GrowthRecord `gorm:"foreignKey:GrowthRecordID" json:"growth_record,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Like) TableName() string {
	return "likes"
}

// Comment 成长记录评论模型
type Comment struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	GrowthRecordID uint      `gorm:"index;not null" json:"growth_record_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	ParentID       *uint     `gorm:"index" json:"parent_id,omitempty"` // 父评论ID（支持嵌套评论）
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联关系
	GrowthRecord GrowthRecord `gorm:"foreignKey:GrowthRecordID" json:"growth_record,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Parent       *Comment     `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children     []*Comment   `gorm:"foreignKey:ParentID" json:"children,omitempty"` // 改为指针切片
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// UserStats 用户统计模型
type UserStats struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	TotalRecords     int       `gorm:"default:0" json:"total_records"`
	TotalLikes       int       `gorm:"default:0" json:"total_likes"`
	TotalComments    int       `gorm:"default:0" json:"total_comments"`
	StorageUsedBytes int64     `gorm:"default:0" json:"storage_used_bytes"` // 已使用存储空间（字节）
	BabyCount        int       `gorm:"default:0" json:"baby_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (UserStats) TableName() string {
	return "user_stats"
}

// Subscription 会员订阅模型
type Subscription struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Plan      string    `gorm:"size:50;not null" json:"plan"` // free/premium/family
	IsActive  bool      `gorm:"index" json:"is_active"`
	StartDate time.Time `gorm:"not null;column:start_date" json:"start_date"`
	EndDate   time.Time `gorm:"not null;column:end_date" json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联关系
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Subscription) TableName() string {
	return "subscriptions"
}

// StorageQuota 存储配额
type StorageQuota struct {
	Plan              string  `gorm:"primaryKey" json:"plan"` // free/premium/family
	MaxMembers        int     `json:"max_members"`
	StorageLimitGB    int     `json:"storage_limit_gb"`
	SupportsMultiBaby bool    `json:"supports_multi_baby"`
	HasPDFExport      bool    `json:"has_pdf_export"`
	PriceMonthly      float64 `json:"price_monthly"`
}

// GetTimeNow 获取当前时间
func GetTimeNow() time.Time {
	return time.Now()
}

// 常量定义
const (
	// InvitationStatus
	InvitationStatusActive  = "active"
	InvitationStatusExpired = "expired"
	InvitationStatusRevoked = "revoked"

	// MembershipApprovalStatus
	ApprovalStatusPending   = "pending"
	ApprovalStatusApproved  = "approved"
	ApprovalStatusRejected  = "rejected"
	ApprovalStatusCancelled = "cancelled"

	// UserPermissionType
	PermissionManageFamily      = "manage_family"      // 管理家庭（批准/拒绝成员）
	PermissionViewRecords       = "view_records"       // 查看记录
	PermissionDeleteRecords     = "delete_records"     // 删除记录
	PermissionExportData        = "export_data"        // 导出数据
	PermissionManageInvitations = "manage_invitations" // 管理邀请码
	PermissionPostComment       = "post_comment"       // 发表评论
	PermissionLikeRecord        = "like_record"        // 点赞记录

	// StorageQuota
	PlanFree    = "free"
	PlanPremium = "premium"
	PlanFamily  = "family"
)
