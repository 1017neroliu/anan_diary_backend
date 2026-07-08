package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FamilyMemberHandler struct {
	memberRepo *repository.FamilyMemberRepository
	authz      *service.AuthzService
	logger     *zap.Logger
}

func NewFamilyMemberHandler(
	memberRepo *repository.FamilyMemberRepository,
	authz *service.AuthzService,
	logger *zap.Logger,
) *FamilyMemberHandler {
	return &FamilyMemberHandler{
		memberRepo: memberRepo,
		authz:      authz,
		logger:     logger,
	}
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	BabyID uint   `json:"baby_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

// AddFamilyMember 添加家庭成员（管理员可操作）
func (h *FamilyMemberHandler) AddFamilyMember(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 权限校验：管理员可添加成员
	if err := h.authz.RequireManageFamily(uint(userID), req.BabyID); err != nil {
		response.Forbidden(c, "无权操作此宝宝档案")
		return
	}

	// 检查免费名额是否已满
	freeLimit := models.FamilyMember{}.GetFreeLimit()
	currentMembers, err := h.memberRepo.CountByBabyID(req.BabyID)
	if err != nil {
		h.logger.Error("Failed to count members", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if int(currentMembers) >= freeLimit {
		h.logger.Warn("Free limit reached",
			zap.Uint("baby_id", req.BabyID),
			zap.Int64("current", currentMembers),
			zap.Int("limit", freeLimit))
		response.Error(c, http.StatusBadRequest, "免费名额已满，最多绑定2位家庭成员。如需添加更多成员，请联系客服升级。")
		return
	}

	// 创建家庭成员
	member := &models.FamilyMember{
		BabyID: req.BabyID,
		Name:   req.Name,
		Role:   req.Role,
	}

	if err := h.memberRepo.Create(member); err != nil {
		h.logger.Error("Failed to create family member", zap.Error(err))
		response.ServerError(c, "添加失败")
		return
	}

	response.Success(c, gin.H{
		"id":    member.ID,
		"name":  member.Name,
		"role":  member.Role,
		"limit": freeLimit,
	})
}

// GetFamilyMembers 获取宝宝的家庭成员列表（需为该宝宝成员）
func (h *FamilyMemberHandler) GetFamilyMembers(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 权限校验：仅家庭成员可查看
	if err := h.authz.RequireMember(uint(userID), uint(babyID)); err != nil {
		response.Forbidden(c, "无权查看此宝宝的家庭成员")
		return
	}

	members, err := h.memberRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get members", zap.Error(err))
		response.ServerError(c, "获取成员列表失败")
		return
	}

	response.Success(c, members)
}
