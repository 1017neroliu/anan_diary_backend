package handlers

import (
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PermissionHandler struct {
	permissionService *service.PermissionService
	logger            *zap.Logger
}

func NewPermissionHandler(
	permissionService *service.PermissionService,
	logger *zap.Logger,
) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		logger:            logger,
	}
}

// GetMyPermissions 获取我的权限列表
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
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

	permissions, err := h.permissionService.GetMyPermissions(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to get permissions", zap.Error(err))
		response.ServerError(c, "获取权限列表失败")
		return
	}
	response.Success(c, permissions)
}

// UpdatePermission 更新权限（管理员操作，需指定 target_user_id）
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
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

	var req struct {
		TargetUserID    uint   `json:"target_user_id" binding:"required"`
		PermissionType  string `json:"permission_type" binding:"required"`
		PermissionValue bool   `json:"permission_value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	if err := h.permissionService.SetUserPermission(uint(userID), req.TargetUserID, babyID, req.PermissionType, req.PermissionValue); err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "权限更新成功"})
}

// RemovePermission 移除权限（管理员操作，需指定 target_user_id）
func (h *PermissionHandler) RemovePermission(c *gin.Context) {
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

	permissionType := c.Param("type")
	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的user_id")
		return
	}

	if err := h.permissionService.RemoveUserPermission(uint(userID), uint(targetUserID), babyID, permissionType); err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "权限移除成功"})
}

// GetFamilyMembers 获取家庭成员的审批信息（管理员可查看）
func (h *PermissionHandler) GetFamilyMembers(c *gin.Context) {
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

	members, err := h.permissionService.ListFamilyMembers(uint(userID), babyID)
	if err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, members)
}

// GetMyFamilyMembers 获取家庭成员（成员视角）
func (h *PermissionHandler) GetMyFamilyMembers(c *gin.Context) {
	// 与 GetFamilyMembers 一致，但后续可加成员级别的过滤
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

	members, err := h.permissionService.ListFamilyMembers(uint(userID), babyID)
	if err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, members)
}

// CheckPermission 检查权限
// 路由 GET /permission/check/:type，baby_id 通过 query 传递。
func (h *PermissionHandler) CheckPermission(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}
	babyID, err := parseUintQuery(c, "baby_id")
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}
	permissionType := c.Param("type")

	hasPermission, err := h.permissionService.CheckPermission(uint(userID), babyID, permissionType)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "检查权限失败")
		return
	}
	response.Success(c, gin.H{"has_permission": hasPermission})
}

// IsMember 检查是否是成员
func (h *PermissionHandler) IsMember(c *gin.Context) {
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

	isMember, err := h.permissionService.CheckMembership(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check member", zap.Error(err))
		response.ServerError(c, "检查成员状态失败")
		return
	}
	response.Success(c, gin.H{"is_member": isMember})
}

// GrantAdminPermission 授予管理员权限
func (h *PermissionHandler) GrantAdminPermission(c *gin.Context) {
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

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的user_id")
		return
	}

	if err := h.permissionService.GrantAdmin(uint(userID), uint(targetUserID), babyID); err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "管理员权限授予成功"})
}

// RemoveMember 移除成员
// 路由 DELETE /permission/member/:user_id，baby_id 通过 query 传递。
func (h *PermissionHandler) RemoveMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}
	babyID, err := parseUintQuery(c, "baby_id")
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	memberIDStr := c.Param("user_id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的user_id")
		return
	}

	if err := h.permissionService.RemoveMember(uint(userID), uint(memberID), babyID); err != nil {
		h.responseServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "成员移除成功"})
}

// responseServiceError 将 Service 错误映射为 HTTP 响应
func (h *PermissionHandler) responseServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotAdmin):
		response.Forbidden(c, "无权操作，仅管理员可执行")
	case errors.Is(err, service.ErrForbidden):
		response.Forbidden(c, "无权限执行此操作")
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(c, "资源不存在")
	default:
		var se *service.ServiceError
		if errors.As(err, &se) {
			switch se.Code {
			case "VALIDATION":
				response.ValidationError(c, se.Msg)
			case "INTERNAL":
				h.logger.Error("permission service error", zap.String("msg", se.Msg))
				response.ServerError(c, se.Msg)
			default:
				response.ServerError(c, se.Msg)
			}
			return
		}
		h.logger.Error("permission service error", zap.Error(err))
		response.ServerError(c, "系统错误")
	}
}
