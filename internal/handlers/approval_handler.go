package handlers

import (
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ApprovalHandler 审批请求处理层
type ApprovalHandler struct {
	approvalService *service.ApprovalService
	logger          *zap.Logger
}

// NewApprovalHandler 创建审批 handler
func NewApprovalHandler(
	approvalService *service.ApprovalService,
	logger *zap.Logger,
) *ApprovalHandler {
	return &ApprovalHandler{
		approvalService: approvalService,
		logger:          logger,
	}
}

// GetApprovals 获取宝宝的待审批列表
func (h *ApprovalHandler) GetApprovals(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID, err := parseUintParam(c, "baby_id")
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	approvals, err := h.approvalService.ListPending(uint(userID), babyID)
	if err != nil {
		h.respondServiceError(c, err)
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

	approvalID, err := parseUintParam(c, "approval_id")
	if err != nil {
		response.ValidationError(c, "无效的申请ID")
		return
	}

	approval, err := h.approvalService.GetDetail(uint(userID), approvalID)
	if err != nil {
		h.respondServiceError(c, err)
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

	approvalID, err := parseUintParam(c, "approval_id")
	if err != nil {
		response.ValidationError(c, "无效的申请ID")
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	if err := h.approvalService.Approve(uint(userID), approvalID, req.Notes); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "审批通过"})
}

// RejectMember 拒绝成员申请
func (h *ApprovalHandler) RejectMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID, err := parseUintParam(c, "approval_id")
	if err != nil {
		response.ValidationError(c, "无效的申请ID")
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	if err := h.approvalService.Reject(uint(userID), approvalID, req.Notes); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "已拒绝"})
}

// CancelMyRequest 取消我的申请
func (h *ApprovalHandler) CancelMyRequest(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvalID, err := parseUintParam(c, "approval_id")
	if err != nil {
		response.ValidationError(c, "无效的申请ID")
		return
	}

	if err := h.approvalService.Cancel(uint(userID), approvalID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "已取消申请"})
}

// GetMyApprovals 获取我的申请列表
func (h *ApprovalHandler) GetMyApprovals(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	approvals, err := h.approvalService.ListMine(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get my approvals", zap.Error(err))
		response.ServerError(c, "获取申请列表失败")
		return
	}
	response.Success(c, approvals)
}

// respondServiceError 将 Service 层错误映射为 HTTP 响应
func (h *ApprovalHandler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotMember):
		response.Forbidden(c, "不是该宝宝的家庭成员")
	case errors.Is(err, service.ErrNotAdmin):
		response.Forbidden(c, "无权操作，仅管理员可执行")
	case errors.Is(err, service.ErrForbidden):
		response.Forbidden(c, "无权限执行此操作")
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(c, "申请记录不存在")
	default:
		var se *service.ServiceError
		if errors.As(err, &se) {
			switch se.Code {
			case "VALIDATION":
				response.ValidationError(c, se.Msg)
			case "INTERNAL":
				h.logger.Error("approval service internal error", zap.String("msg", se.Msg))
				response.ServerError(c, se.Msg)
			default:
				response.ServerError(c, se.Msg)
			}
			return
		}
		h.logger.Error("approval service error", zap.Error(err))
		response.ServerError(c, "系统错误")
	}
}

// parseUintParam 解析路径参数为 uint
func parseUintParam(c *gin.Context, name string) (uint, error) {
	val := c.Param(name)
	id, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// parseUintQuery 解析查询参数为 uint
func parseUintQuery(c *gin.Context, name string) (uint, error) {
	val := c.Query(name)
	if val == "" {
		return 0, fmt.Errorf("missing query param %s", name)
	}
	id, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
