package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ApprovalHandler struct {
	approvalRepo *repository.ApprovalRepository
	permissionRepo *repository.PermissionRepository
	memberRepo *repository.FamilyMemberRepository
	logger *zap.Logger
}

func NewApprovalHandler(
	approvalRepo *repository.ApprovalRepository,
	permissionRepo *repository.PermissionRepository,
	memberRepo *repository.FamilyMemberRepository,
	logger *zap.Logger,
) *ApprovalHandler {
	return &ApprovalHandler{
		approvalRepo:   approvalRepo,
		permissionRepo: permissionRepo,
		memberRepo:     memberRepo,
		logger:         logger,
	}
}

// GetApprovals 获取宝宝的待审批列表
func (h *ApprovalHandler) GetApprovals(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	_ = babyIDStr
	babyID := c.GetUint("baby_id")

	// 检查权限：只有创建者和管理员可以查看
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权查看审批列表")
		return
	}

	// 获取待审批列表
	approvals, err := h.approvalRepo.GetPendingByBabyID(babyID)
	if err != nil {
		h.logger.Error("Failed to get approvals", zap.Error(err))
		response.ServerError(c, "获取审批列表失败")
		return
	}

	response.Success(c, approvals)
}

// GetApprovalDetail 获取审批详情
func (h *ApprovalHandler) GetApprovalDetail(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID := c.Param("id")
	_ = approvalID
	var approvalIDUint uint
	_ = c.GetUint("baby_id") // 验证 baby_id 存在

	// 获取审批详情
	approval, err := h.approvalRepo.GetByID(approvalIDUint)
	if err != nil {
		response.NotFound(c, "申请记录不存在")
		return
	}

	// 验证宝宝属于当前用户
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), approval.BabyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权查看此申请")
		return
	}

	response.Success(c, approval)
}

// ApproveMember 批准成员申请
func (h *ApprovalHandler) ApproveMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID := c.Param("id")
	_ = approvalID
	var approvalIDUint uint
	_ = c.GetUint("baby_id") // 验证 baby_id 存在

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 获取审批详情
	approval, err := h.approvalRepo.GetByID(approvalIDUint)
	if err != nil {
		response.NotFound(c, "申请记录不存在")
		return
	}

	// 验证宝宝属于当前用户
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), approval.BabyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权审批此申请")
		return
	}

	// 只能审批待审批的申请
	if approval.Status != "pending" {
		response.Error(c, http.StatusBadRequest, "该申请已被处理")
		return
	}

	// 批准申请
	if err := h.approvalRepo.Approve(approvalIDUint, uint(userID), req.Notes); err != nil {
		h.logger.Error("Failed to approve member", zap.Error(err))
		response.ServerError(c, "审批失败")
		return
	}

	// 创建家庭成员记录
	member := &models.FamilyMember{
		BabyID: approval.BabyID,
		UserID: &approval.UserID,
		Name:   approval.Name,
		Role:   approval.Role,
	}

	if err := h.memberRepo.Create(member); err != nil {
		h.logger.Error("Failed to create family member", zap.Error(err))
		response.ServerError(c, "创建家庭成员失败")
		return
	}

	// 授予权限
	permissions := []*models.UserPermission{
		{
			UserID:         approval.UserID,
			BabyID:         approval.BabyID,
			PermissionType: "view_records",
			PermissionValue: true,
		},
	}

	if hasPermission {
		permissions = append(permissions, &models.UserPermission{
			UserID:         approval.UserID,
			BabyID:         approval.BabyID,
			PermissionType: "post_comment",
			PermissionValue: true,
		})
		permissions = append(permissions, &models.UserPermission{
			UserID:         approval.UserID,
			BabyID:         approval.BabyID,
			PermissionType: "like_record",
			PermissionValue: true,
		})
	}

	if hasPermission {
		permissions = append(permissions, &models.UserPermission{
			UserID:         approval.UserID,
			BabyID:         approval.BabyID,
			PermissionType: "post_comment",
			PermissionValue: true,
		})
		permissions = append(permissions, &models.UserPermission{
			UserID:         approval.UserID,
			BabyID:         approval.BabyID,
			PermissionType: "like_record",
			PermissionValue: true,
		})
	}

	if err := h.permissionRepo.BatchSetPermissions(permissions); err != nil {
		h.logger.Error("Failed to set permissions", zap.Error(err))
		response.ServerError(c, "设置权限失败")
		return
	}

	h.logger.Info("Approved family membership",
		zap.Uint("approval_id", approvalIDUint),
		zap.Uint("baby_id", approval.BabyID),
		zap.Uint("user_id", approval.UserID),
		zap.String("role", approval.Role))

	response.Success(c, gin.H{
		"message": "审批通过",
	})
}

// RejectMember 拒绝成员申请
func (h *ApprovalHandler) RejectMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID := c.Param("id")
	_ = approvalID
	var approvalIDUint uint
	_ = c.GetUint("baby_id") // 验证 baby_id 存在

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 获取审批详情
	approval, err := h.approvalRepo.GetByID(approvalIDUint)
	if err != nil {
		response.NotFound(c, "申请记录不存在")
		return
	}

	// 验证宝宝属于当前用户
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), approval.BabyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权审批此申请")
		return
	}

	// 只能审批待审批的申请
	if approval.Status != "pending" {
		response.Error(c, http.StatusBadRequest, "该申请已被处理")
		return
	}

	// 拒绝申请
	if err := h.approvalRepo.Reject(approvalIDUint, uint(userID), req.Notes); err != nil {
		h.logger.Error("Failed to reject member", zap.Error(err))
		response.ServerError(c, "拒绝失败")
		return
	}

	h.logger.Info("Rejected family membership",
		zap.Uint("approval_id", approvalIDUint),
		zap.Uint("baby_id", approval.BabyID))

	response.Success(c, gin.H{
		"message": "已拒绝",
	})
}

// CancelMyRequest 取消我的申请
func (h *ApprovalHandler) CancelMyRequest(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID := c.Param("id")
	_ = approvalID
	var approvalIDUint uint
	_ = c.GetUint("baby_id") // 验证 baby_id 存在

	// 获取审批详情
	approval, err := h.approvalRepo.GetByID(approvalIDUint)
	if err != nil {
		response.NotFound(c, "申请记录不存在")
		return
	}

	// 只能取消自己的待审批申请
	if approval.UserID != uint(userID) {
		response.Forbidden(c, "只能取消自己的申请")
		return
	}

	if approval.Status != "pending" {
		response.Error(c, http.StatusBadRequest, "只能取消待审批的申请")
		return
	}

	// 取消申请
	if err := h.approvalRepo.Cancel(approvalIDUint, uint(userID)); err != nil {
		h.logger.Error("Failed to cancel approval", zap.Error(err))
		response.ServerError(c, "取消申请失败")
		return
	}

	h.logger.Info("Cancelled family membership approval",
		zap.Uint("approval_id", approvalIDUint),
		zap.Uint("user_id", uint(userID)))

	response.Success(c, gin.H{
		"message": "已取消申请",
	})
}

// GetMyApprovals 获取我的申请列表
func (h *ApprovalHandler) GetMyApprovals(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取我的所有申请
	approvals, err := h.approvalRepo.GetByUser(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get my approvals", zap.Error(err))
		response.ServerError(c, "获取申请列表失败")
		return
	}

	response.Success(c, approvals)
}
