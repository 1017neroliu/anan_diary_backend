package handlers

import (
	"net/http"
	"strconv"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FamilyMemberHandler struct {
	memberRepo *repository.FamilyMemberRepository
	logger     *zap.Logger
}

func NewFamilyMemberHandler(memberRepo *repository.FamilyMemberRepository, logger *zap.Logger) *FamilyMemberHandler {
	return &FamilyMemberHandler{
		memberRepo: memberRepo,
		logger:     logger,
	}
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	BabyID uint   `json:"baby_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

// AddFamilyMember 添加家庭成员
// 核心逻辑：1) 检查免费名额是否已满 2) 免费名额为2人
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

	// 验证宝宝属于当前用户
	baby, err := h.memberRepo.GetBabyByID(req.BabyID)
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}
	if baby.UserID != uint(userID) {
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

// GetFamilyMembers 获取宝宝的家庭成员列表
func (h *FamilyMemberHandler) GetFamilyMembers(c *gin.Context) {
	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	members, err := h.memberRepo.GetByBabyID(uint(babyID))
	if err != nil {
		response.ServerError(c, "获取成员列表失败")
		return
	}

	response.Success(c, members)
}
