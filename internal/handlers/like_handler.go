package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LikeHandler struct {
	likeRepo       *repository.LikeRepository
	commentRepo    *repository.CommentRepository
	permissionRepo *repository.PermissionRepository
	logger         *zap.Logger
}

func NewLikeHandler(
	likeRepo *repository.LikeRepository,
	commentRepo *repository.CommentRepository,
	permissionRepo *repository.PermissionRepository,
	logger *zap.Logger,
) *LikeHandler {
	return &LikeHandler{
		likeRepo:       likeRepo,
		commentRepo:    commentRepo,
		permissionRepo: permissionRepo,
		logger:         logger,
	}
}

// LikeRecord 点赞成长记录
func (h *LikeHandler) LikeRecord(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	// 检查是否已点赞
	hasLiked, err := h.likeRepo.HasLiked(uint(recordIDUint), uint(userID))
	if err != nil {
		h.logger.Error("Failed to check if already liked", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if hasLiked {
		response.Error(c, http.StatusBadRequest, "您已点赞过此记录")
		return
	}

	// 创建点赞记录
	like := &models.Like{
		GrowthRecordID: uint(recordIDUint),
		UserID:         uint(userID),
	}

	if err := h.likeRepo.Create(like); err != nil {
		h.logger.Error("Failed to create like", zap.Error(err))
		response.ServerError(c, "点赞失败")
		return
	}

	// 增加记录的点赞计数
	if err := h.likeRepo.IncrementLikeCount(uint(recordIDUint)); err != nil {
		h.logger.Error("Failed to increment like count", zap.Error(err))
	}

	h.logger.Info("Liked growth record",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("record_id", uint(recordIDUint)))

	response.Success(c, gin.H{
		"message": "点赞成功",
	})
}

// UnlikeRecord 取消点赞
func (h *LikeHandler) UnlikeRecord(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	// 检查是否已点赞
	hasLiked, err := h.likeRepo.HasLiked(uint(recordIDUint), uint(userID))
	if err != nil {
		h.logger.Error("Failed to check if already liked", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasLiked {
		response.Error(c, http.StatusBadRequest, "您未点赞过此记录")
		return
	}

	// 删除点赞记录
	if err := h.likeRepo.Delete(uint(recordIDUint), uint(userID)); err != nil {
		h.logger.Error("Failed to unlike record", zap.Error(err))
		response.ServerError(c, "取消点赞失败")
		return
	}

	// 减少记录的点赞计数
	if err := h.likeRepo.DecrementLikeCount(uint(recordIDUint)); err != nil {
		h.logger.Error("Failed to decrement like count", zap.Error(err))
	}

	h.logger.Info("Unliked growth record",
		zap.Uint("user_id", uint(userID)),
		zap.String("record_id", recordID))

	response.Success(c, gin.H{
		"message": "取消点赞成功",
	})
}

// GetRecordLikes 获取记录的点赞者列表
func (h *LikeHandler) GetRecordLikes(c *gin.Context) {
	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	likes, err := h.likeRepo.GetByGrowthRecord(uint(recordIDUint))
	if err != nil {
		h.logger.Error("Failed to get record likes", zap.Error(err))
		response.ServerError(c, "获取点赞列表失败")
		return
	}

	response.Success(c, likes)
}

// GetLikeCount 获取记录的点赞数
func (h *LikeHandler) GetLikeCount(c *gin.Context) {
	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	count, err := h.likeRepo.GetLikeCount(uint(recordIDUint))
	if err != nil {
		h.logger.Error("Failed to get like count", zap.Error(err))
		response.ServerError(c, "获取点赞数失败")
		return
	}

	response.Success(c, gin.H{
		"count":  count,
		"liked":  false, // 需要前端调用另一接口获取
	})
}

// HasLikedCheck 检查用户是否点赞了某条记录
func (h *LikeHandler) HasLikedCheck(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordID := c.Param("id")
	recordIDUint, err := strconv.ParseUint(recordID, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的记录ID")
		return
	}

	hasLiked, err := h.likeRepo.HasLiked(uint(recordIDUint), uint(userID))
	if err != nil {
		h.logger.Error("Failed to check if liked", zap.Error(err))
		response.ServerError(c, "检查点赞状态失败")
		return
	}

	response.Success(c, gin.H{
		"has_liked": hasLiked,
	})
}

// GetUserLikes 获取用户的所有点赞记录
func (h *LikeHandler) GetUserLikes(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	likes, err := h.likeRepo.GetByUser(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get user likes", zap.Error(err))
		response.ServerError(c, "获取点赞记录失败")
		return
	}

	response.Success(c, likes)
}

// GetTopLikedRecords 获取获赞最多的记录
func (h *LikeHandler) GetTopLikedRecords(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Query("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 检查权限
	hasPermission, err := h.permissionRepo.HasAdminPermission(uint(userID), uint(babyID))
	if err != nil {
		h.logger.Error("Failed to check permission", zap.Error(err))
		response.ServerError(c, "系统错误")
		return
	}

	if !hasPermission {
		response.Forbidden(c, "无权查看")
		return
	}

	limit := 10
	topRecords, err := h.likeRepo.GetTopLikedRecords(uint(babyID), limit)
	if err != nil {
		h.logger.Error("Failed to get top liked records", zap.Error(err))
		response.ServerError(c, "获取热门记录失败")
		return
	}

	response.Success(c, topRecords)
}
