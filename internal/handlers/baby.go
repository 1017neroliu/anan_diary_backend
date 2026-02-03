package handlers

import (
	"net/http"
	"strconv"
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BabyHandler struct {
	babyRepo  *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	logger    *zap.Logger
}

func NewBabyHandler(babyRepo *repository.BabyRepository, memberRepo *repository.FamilyMemberRepository, logger *zap.Logger) *BabyHandler {
	return &BabyHandler{
		babyRepo:  babyRepo,
		memberRepo: memberRepo,
		logger:    logger,
	}
}

// CreateBabyRequest 创建宝宝请求
type CreateBabyRequest struct {
	Name      string `json:"name" binding:"required"`
	BirthDate string `json:"birth_date" binding:"required"`
	BabyID    string `json:"baby_id" binding:"required"`
	UserEmail string `json:"user_email"` // 从前端传递用户的邮箱
	UserName  string `json:"user_name"`  // 从前端传递用户的姓名
	UserRole  string `json:"user_role"`  // 从前端传递用户的角色
}

// CreateBaby 创建宝宝档案
func (h *BabyHandler) CreateBaby(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req CreateBabyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 检查出生证编号是否已存在
	existingBaby, err := h.babyRepo.GetByBabyID(req.BabyID)
	if err == nil && existingBaby != nil {
		response.Error(c, http.StatusBadRequest, "该出生证编号已被使用")
		return
	}

	// 创建宝宝档案
	baby := &models.Baby{
		UserID:    userID,
		Name:      req.Name,
		BirthDate: req.BirthDate,
		BabyID:    req.BabyID,
	}

	if err := h.babyRepo.Create(baby); err != nil {
		h.logger.Error("Failed to create baby", zap.Error(err))
		response.ServerError(c, "创建宝宝档案失败")
		return
	}

	// 创建家庭成员记录
	member := &models.FamilyMember{
		BabyID: baby.ID,
		Name:   req.UserName,
		Role:   req.UserRole,
	}

	if err := h.memberRepo.Create(member); err != nil {
		h.logger.Error("Failed to create family member", zap.Error(err))
		response.ServerError(c, "创建宝宝档案失败")
		return
	}

	h.logger.Info("Baby profile created with family member",
		zap.Uint("baby_id", baby.ID),
		zap.String("name", baby.Name),
		zap.String("member_name", member.Name),
		zap.String("member_role", member.Role))

	response.Success(c, baby)
}

// GetBabies 获取当前用户的宝宝列表
func (h *BabyHandler) GetBabies(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babies, err := h.babyRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get babies", zap.Error(err))
		response.ServerError(c, "获取宝宝列表失败")
		return
	}

	response.Success(c, babies)
}

// GetBabyByID 获取宝宝详细信息
func (h *BabyHandler) GetBabyByID(c *gin.Context) {
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

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 验证权限
	if baby.UserID != userID {
		response.Forbidden(c, "无权访问此宝宝档案")
		return
	}

	response.Success(c, baby)
}

// DeleteBaby 删除宝宝档案
func (h *BabyHandler) DeleteBaby(c *gin.Context) {
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

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 验证权限
	if baby.UserID != userID {
		response.Forbidden(c, "无权删除此宝宝档案")
		return
	}

	if err := h.babyRepo.Delete(uint(babyID)); err != nil {
		h.logger.Error("Failed to delete baby", zap.Error(err))
		response.ServerError(c, "删除失败")
		return
	}

	h.logger.Info("Baby profile deleted", zap.Uint("baby_id", uint(babyID)))
	response.Success(c, gin.H{"message": "删除成功"})
}

// GetCurrentBaby 获取当前用户绑定的第一个宝宝
func (h *BabyHandler) GetCurrentBaby(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babies, err := h.babyRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get babies", zap.Error(err))
		response.ServerError(c, "获取宝宝列表失败")
		return
	}

	if len(babies) == 0 {
		response.Error(c, http.StatusNotFound, "请先创建宝宝档案")
		return
	}

	response.Success(c, babies[0])
}
