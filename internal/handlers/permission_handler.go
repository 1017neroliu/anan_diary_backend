package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PermissionHandler struct {
	permissionRepo *repository.PermissionRepository
	logger         *zap.Logger
}

func NewPermissionHandler(
	permissionRepo *repository.PermissionRepository,
	logger *zap.Logger,
) *PermissionHandler {
	return &PermissionHandler{
		permissionRepo: permissionRepo,
		logger:         logger,
	}
}

// GetMyPermissions 获取我的权限列表
func (h *PermissionHandler) GetMyPermissions(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	// 获取权限列表
	permissions, err := h.permissionRepo.GetByUserAndBaby(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to get permissions", zap.Error(err))
		response.ServerError(c, "获取权限列表失败")
		return
	}

	response.Success(c, permissions)
}

// UpdatePermission 更新权限
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	var req struct {
		PermissionType string `json:"permission_type" binding:"required"`
		PermissionValue bool  `json:"permission_value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 检查权限：只有管理员可以修改权限
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权修改权限")
		return
	}

	// 设置权限
	permission := &models.UserPermission{
		UserID: uint(userID),
		BabyID:         babyID,
		PermissionType: req.PermissionType,
		PermissionValue: req.PermissionValue,
	}

	if err := h.permissionRepo.SetPermission(permission); err != nil {
		h.logger.Error("Failed to set permission", zap.Error(err))
		response.ServerError(c, "更新权限失败")
		return
	}

	h.logger.Info("Updated user permission",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("baby_id", babyID),
		zap.String("permission_type", req.PermissionType),
		zap.Bool("permission_value", req.PermissionValue))

	response.Success(c, gin.H{
		"message": "权限更新成功",
	})
}

// RemovePermission 移除权限
func (h *PermissionHandler) RemovePermission(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	permissionType := c.Param("type")

	// 检查权限：只有管理员可以移除权限
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权移除权限")
		return
	}

	// 移除权限
	if err := h.permissionRepo.RemovePermission(uint(userID), babyID, permissionType); err != nil {
		h.logger.Error("Failed to remove permission", zap.Error(err))
		response.ServerError(c, "移除权限失败")
		return
	}

	h.logger.Info("Removed user permission",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("baby_id", babyID),
		zap.String("permission_type", permissionType))

	response.Success(c, gin.H{
		"message": "权限移除成功",
	})
}

// GetFamilyMembers 获取宝宝的家庭成员及其权限
func (h *PermissionHandler) GetFamilyMembers(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	// 检查权限：只有管理员可以查看
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权查看家庭成员列表")
		return
	}

	// 获取家庭成员及其权限
	members, err := h.permissionRepo.GetFamilyMembersWithPermissions(babyID)
	if err != nil {
		h.logger.Error("Failed to get family members", zap.Error(err))
		response.ServerError(c, "获取家庭成员列表失败")
		return
	}

	response.Success(c, members)
}

// GetMyFamilyMembers 获取我作为成员的家庭成员及其权限
func (h *PermissionHandler) GetMyFamilyMembers(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	// 获取家庭成员及其权限（不需要管理员权限，只要是成员即可查看）
	members, err := h.permissionRepo.GetFamilyMembersWithPermissions(babyID)
	if err != nil {
		h.logger.Error("Failed to get family members", zap.Error(err))
		response.ServerError(c, "获取家庭成员列表失败")
		return
	}

	response.Success(c, members)
}

// CheckPermission 检查权限
func (h *PermissionHandler) CheckPermission(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	permissionType := c.Param("type")

	// 检查权限
	hasPermission, err := h.permissionRepo.HasPermission(uint(userID), babyID, permissionType)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "检查权限失败")
		return
	}

	response.Success(c, gin.H{
		"has_permission": hasPermission,
	})
}

// IsMember 检查是否是成员
func (h *PermissionHandler) IsMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	// 检查是否是成员
	isMember, err := h.permissionRepo.IsMember(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check member status", zap.Error(err))
		response.ServerError(c, "检查成员状态失败")
		return
	}

	response.Success(c, gin.H{
		"is_member": isMember,
	})
}

// GrantAdminPermission 授予管理员权限
func (h *PermissionHandler) GrantAdminPermission(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的user_id")
		return
	}

	// 检查权限：只有创建者和管理员可以授予管理员权限
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权授予管理员权限")
		return
	}

	// 授予管理员权限
	permissions := []*models.UserPermission{
		{
			UserID:         uint(targetUserID),
			BabyID:         babyID,
			PermissionType: "manage_family",
			PermissionValue: true,
		},
		{
			UserID:         uint(targetUserID),
			BabyID:         babyID,
			PermissionType: "view_records",
			PermissionValue: true,
		},
		{
			UserID:         uint(targetUserID),
			BabyID:         babyID,
			PermissionType: "delete_records",
			PermissionValue: true,
		},
		{
			UserID:         uint(targetUserID),
			BabyID:         babyID,
			PermissionType: "export_data",
			PermissionValue: true,
		},
	}

	if err := h.permissionRepo.BatchSetPermissions(permissions); err != nil {
		h.logger.Error("Failed to grant admin permission", zap.Error(err))
		response.ServerError(c, "授予管理员权限失败")
		return
	}

	h.logger.Info("Granted admin permission",
		zap.Uint("by", uint(userID)),
		zap.Uint("to", uint(targetUserID)),
		zap.Uint("baby_id", babyID))

	response.Success(c, gin.H{
		"message": "管理员权限授予成功",
	})
}

// RemoveMember 移除成员
func (h *PermissionHandler) RemoveMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID := c.GetUint("baby_id")

	memberIDStr := c.Param("user_id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的user_id")
		return
	}

	// 检查权限：只有创建者和管理员可以移除成员
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), babyID)
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权移除成员")
		return
	}

	// 检查是否是创建者
	if uint(memberID) == babyID {
		response.Error(c, 400, "不能移除创建者")
		return
	}

	// 移除权限
	if err := h.permissionRepo.RemoveUserPermissions(uint(memberID)); err != nil {
		h.logger.Error("Failed to remove permissions", zap.Error(err))
		response.ServerError(c, "移除权限失败")
		return
	}

	_ = memberID

	h.logger.Info("Removed member",
		zap.Uint("by", uint(userID)),
		zap.Uint("removed", uint(memberID)),
		zap.Uint("baby_id", babyID))

	response.Success(c, gin.H{
		"message": "成员移除成功",
	})
}
