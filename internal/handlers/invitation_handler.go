package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type InvitationHandler struct {
	invitationRepo *repository.InvitationRepository
	approvalRepo   *repository.ApprovalRepository
	permissionRepo *repository.PermissionRepository
	memberRepo     *repository.FamilyMemberRepository
	authz          *service.AuthzService
	logger         *zap.Logger
}

func NewInvitationHandler(
	invitationRepo *repository.InvitationRepository,
	approvalRepo *repository.ApprovalRepository,
	permissionRepo *repository.PermissionRepository,
	memberRepo *repository.FamilyMemberRepository,
	authz *service.AuthzService,
	logger *zap.Logger,
) *InvitationHandler {
	return &InvitationHandler{
		invitationRepo: invitationRepo,
		approvalRepo:   approvalRepo,
		permissionRepo: permissionRepo,
		memberRepo:     memberRepo,
		authz:          authz,
		logger:         logger,
	}
}

// CreateInviteCodeRequest 创建邀请码请求（通过baby_id）
type CreateInviteCodeRequest struct {
	BabyID uint `json:"baby_id" binding:"required"`
}

// CreateInviteCode 创建邀请码（简化版，用于注册流程）
func (h *InvitationHandler) CreateInviteCode(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req CreateInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 验证宝宝属于当前用户
	_, err := h.memberRepo.GetBabyByID(req.BabyID)
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 检查权限：只有创建者和管理员可以创建邀请码
	// 走 authz.IsAdmin（含创建者兜底），不直接查权限表——否则新注册的创建者
	// 因 user_permissions 表为空会被误判为非管理员，无法生成邀请码。
	isAdmin, err := h.authz.IsAdmin(uint(userID), req.BabyID)
	if err != nil {
		h.logger.Error("Failed to check admin", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !isAdmin {
		response.Forbidden(c, "无权操作此宝宝档案")
		return
	}

	// 检查是否有活跃邀请码
	activeCode, err := h.invitationRepo.GetActiveCode(req.BabyID)
	if err == nil && activeCode != nil {
		response.Error(c, http.StatusBadRequest, "该宝宝已有活跃邀请码，请使用现有邀请码")
		return
	}

	// 生成新的邀请码
	code := "INV-" + strings.ToUpper(uuid.New().String()[:8])

	invitation := &models.Invitation{
		Code:      code,
		BabyID:    req.BabyID,
		CreatedBy: uint(userID),
		Status:    models.InvitationStatusActive,
		MaxUses:   10,
		UsedCount: 0,
	}

	if err := h.invitationRepo.Create(invitation); err != nil {
		h.logger.Error("Failed to create invitation", zap.Error(err))
		response.ServerError(c, "创建邀请码失败")
		return
	}

	response.Success(c, gin.H{
		"code":    code,
		"baby_id": req.BabyID,
		"message": "邀请码创建成功",
	})
}

// JoinFamilyRequest 加入家庭请求
type JoinFamilyRequest struct {
	BabyID    uint   `json:"baby_id" binding:"required"`
	Code      string `json:"code" binding:"required"` // 邀请码
	Role      string `json:"role" binding:"required"` // 角色
	Name      string `json:"name" binding:"required"` // 姓名
	Reason    string `json:"reason"`                  // 申请理由
	UserAgent string `json:"user_agent"`              // 用户代理
}

// JoinFamily 通过邀请码加入家庭
func (h *InvitationHandler) JoinFamily(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req JoinFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 验证宝宝是否存在
	_, err := h.memberRepo.GetBabyByID(req.BabyID)
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 验证邀请码
	invitation, err := h.invitationRepo.GetByCode(req.Code)
	if err != nil {
		response.Error(c, http.StatusNotFound, "邀请码不存在或已失效")
		return
	}

	// 验证邀请码是否属于该宝宝
	if invitation.BabyID != req.BabyID {
		response.Error(c, http.StatusBadRequest, "邀请码不匹配")
		return
	}

	// 验证邀请码状态
	if invitation.Status != models.InvitationStatusActive {
		response.Error(c, http.StatusBadRequest, "邀请码已失效")
		return
	}

	// 检查是否达到最大使用次数
	hasReachedMax, err := h.invitationRepo.CheckInvitationUsed(invitation.ID)
	if err != nil {
		h.logger.Error("Failed to check invitation usage", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}
	if hasReachedMax {
		response.Error(c, http.StatusBadRequest, "邀请码使用次数已达上限")
		return
	}

	// 检查是否已有该角色的批准成员
	hasApprovedRole, err := h.approvalRepo.HasApprovedRole(uint(userID), req.BabyID, req.Role)
	if err != nil {
		h.logger.Error("Failed to check approved role", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}
	if hasApprovedRole {
		response.Error(c, http.StatusBadRequest, "您已是该角色的家庭成员，无需重复申请")
		return
	}

	// 检查是否有待审批的申请
	hasPending, err := h.approvalRepo.GetByUserAndBabyRole(uint(userID), req.BabyID, req.Role)
	if err == nil && hasPending != nil && hasPending.Status == models.ApprovalStatusPending {
		response.Error(c, http.StatusBadRequest, "您已有待审批的申请，请等待审批")
		return
	}

	// 检查是否已有该宝宝的家庭成员记录（旧逻辑兼容）
	memberCount, err := h.memberRepo.CountByBabyID(req.BabyID)
	if err != nil {
		h.logger.Error("Failed to count members", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}
	var fm models.FamilyMember
	if int(memberCount) >= fm.GetFreeLimit() {
		response.Error(c, http.StatusBadRequest, "家庭名额已满，请联系管理员")
		return
	}

	// 记录邀请码使用
	if err := h.invitationRepo.RecordInvitationUsage(invitation.ID, uint(userID), req.UserAgent); err != nil {
		h.logger.Error("Failed to record invitation usage", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	// 创建审批申请
	approval := &models.MembershipApproval{
		BabyID:    req.BabyID,
		UserID:    uint(userID),
		Role:      req.Role,
		Name:      req.Name,
		Reason:    req.Reason,
		Status:    models.ApprovalStatusPending,
		CreatedAt: models.GetTimeNow(),
		UpdatedAt: models.GetTimeNow(),
	}

	if err := h.approvalRepo.Create(approval); err != nil {
		h.logger.Error("Failed to create approval", zap.Error(err))
		response.ServerError(c, "创建申请失败")
		return
	}

	h.logger.Info("Created family membership approval",
		zap.Uint("baby_id", req.BabyID),
		zap.Uint("user_id", uint(userID)),
		zap.String("role", req.Role),
		zap.String("name", req.Name),
		zap.String("status", models.ApprovalStatusPending))

	response.Success(c, gin.H{
		"approval_id": approval.ID,
		"message":     "申请已提交，请等待管理员审批",
	})
}

// GetInviteCode 获取宝宝的邀请码
func (h *InvitationHandler) GetInviteCode(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 检查权限
	isAdmin, err := h.authz.IsAdmin(uint(userID), uint(babyID))
	if err != nil {
		h.logger.Error("Failed to check admin", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !isAdmin {
		response.Forbidden(c, "无权查看邀请码")
		return
	}

	// 获取邀请码
	invitation, err := h.invitationRepo.GetActiveCode(uint(babyID))
	if err != nil {
		response.NotFound(c, "暂无邀请码，请先生成")
		return
	}

	response.Success(c, invitation)
}

// GetInvitationCodes 获取宝宝的邀请码列表
func (h *InvitationHandler) GetInvitationCodes(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 检查权限
	isAdmin, err := h.authz.IsAdmin(uint(userID), uint(babyID))
	if err != nil {
		h.logger.Error("Failed to check admin", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !isAdmin {
		response.Forbidden(c, "无权查看邀请码列表")
		return
	}

	// 获取邀请码列表
	invitations, err := h.invitationRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get invitations", zap.Error(err))
		response.ServerError(c, "获取邀请码列表失败")
		return
	}

	response.Success(c, invitations)
}

// RevokeInviteCode 撤销邀请码
func (h *InvitationHandler) RevokeInviteCode(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 检查权限
	isAdmin, err := h.authz.IsAdmin(uint(userID), uint(babyID))
	if err != nil {
		h.logger.Error("Failed to check admin", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !isAdmin {
		response.Forbidden(c, "无权撤销邀请码")
		return
	}

	// 撤销所有活跃邀请码
	if err := h.invitationRepo.UpdateStatus(uint(babyID), models.InvitationStatusRevoked); err != nil {
		h.logger.Error("Failed to revoke invitations", zap.Error(err))
		response.ServerError(c, "撤销邀请码失败")
		return
	}

	h.logger.Info("Revoked all invitation codes", zap.Uint("baby_id", uint(babyID)), zap.Uint("by", uint(userID)))

	response.Success(c, gin.H{
		"message": "邀请码已全部撤销",
	})
}
