package handlers

import (
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscriptionHandler struct {
	subscriptionRepo *repository.SubscriptionRepository
	logger           *zap.Logger
}

func NewSubscriptionHandler(
	subscriptionRepo *repository.SubscriptionRepository,
	logger *zap.Logger,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// GetSubscriptionDetail 获取订阅详情
func (h *SubscriptionHandler) GetSubscriptionDetail(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取订阅详情
	detail, err := h.subscriptionRepo.GetSubscriptionDetail(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get subscription detail", zap.Error(err))
		response.ServerError(c, "获取订阅详情失败")
		return
	}

	response.Success(c, detail)
}

// CheckSubscriptionStatus 检查订阅状态
func (h *SubscriptionHandler) CheckSubscriptionStatus(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 检查订阅状态
	isActive, hasSubscription, err := h.subscriptionRepo.CheckSubscriptionStatus(uint(userID))
	if err != nil {
		h.logger.Error("Failed to check subscription status", zap.Error(err))
		response.ServerError(c, "检查订阅状态失败")
		return
	}

	response.Success(c, gin.H{
		"is_active":        isActive,
		"has_subscription": hasSubscription,
	})
}

// GetSubscriptionQuota 获取存储配额信息
func (h *SubscriptionHandler) GetSubscriptionQuota(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取订阅详情
	detail, err := h.subscriptionRepo.GetSubscriptionDetail(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get subscription detail", zap.Error(err))
		response.ServerError(c, "获取配额信息失败")
		return
	}

	// 获取存储限制
	storageLimit, err := h.subscriptionRepo.GetStorageQuota(detail["plan"].(string))
	if err != nil {
		h.logger.Error("Failed to get storage quota", zap.Error(err))
		response.ServerError(c, "获取配额信息失败")
		return
	}

	response.Success(c, gin.H{
		"plan":                detail["plan"],
		"is_active":           detail["is_active"],
		"start_date":          detail["start_date"],
		"end_date":            detail["end_date"],
		"max_members":         detail["max_members"],
		"storage_limit":       storageLimit,
		"supports_multi_baby": detail["supports_multi_baby"],
		"has_pdf_export":      detail["has_pdf_export"],
	})
}

// HasMultiBabySupport 检查是否支持多宝宝
func (h *SubscriptionHandler) HasMultiBabySupport(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取订阅详情
	detail, err := h.subscriptionRepo.GetSubscriptionDetail(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get subscription detail", zap.Error(err))
		response.ServerError(c, "检查失败")
		return
	}

	supportsMultiBaby, err := h.subscriptionRepo.SupportsMultiBaby(detail["plan"].(string))
	if err != nil {
		h.logger.Error("Failed to check multi baby support", zap.Error(err))
		response.ServerError(c, "检查失败")
		return
	}

	response.Success(c, gin.H{
		"supports_multi_baby": supportsMultiBaby,
	})
}

// HasPDFExport 检查是否有PDF导出功能
func (h *SubscriptionHandler) HasPDFExport(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取订阅详情
	detail, err := h.subscriptionRepo.GetSubscriptionDetail(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get subscription detail", zap.Error(err))
		response.ServerError(c, "检查失败")
		return
	}

	hasPDFExport, err := h.subscriptionRepo.HasPDFExport(detail["plan"].(string))
	if err != nil {
		h.logger.Error("Failed to check PDF export", zap.Error(err))
		response.ServerError(c, "检查失败")
		return
	}

	response.Success(c, gin.H{
		"has_pdf_export": hasPDFExport,
	})
}

// GetActiveSubscriptions 获取活跃订阅列表（管理员用）
func (h *SubscriptionHandler) GetActiveSubscriptions(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 检查权限：只有管理员可以查看
	// TODO: 需要实现 HasAdminPermission 的 0 参数版本
	// 暂时跳过权限检查，或者根据实际需求添加权限验证

	// 获取订阅统计
	stats, err := h.subscriptionRepo.GetSubscriptionStats()
	if err != nil {
		h.logger.Error("Failed to get subscription stats", zap.Error(err))
		response.ServerError(c, "获取订阅统计失败")
		return
	}

	response.Success(c, stats)
}

// GetSubscriptionPlans 获取订阅计划列表
func (h *SubscriptionHandler) GetSubscriptionPlans(c *gin.Context) {
	// 获取配额信息
	plans := map[string]interface{}{
		"free": map[string]interface{}{
			"max_members":         4,
			"storage_limit_gb":    1,
			"supports_multi_baby": false,
			"has_pdf_export":      false,
			"price_monthly":       0,
			"plan_name":           "免费版",
		},
		"premium": map[string]interface{}{
			"max_members":         10,
			"storage_limit_gb":    5,
			"supports_multi_baby": true,
			"has_pdf_export":      true,
			"price_monthly":       19.90,
			"plan_name":           "高级版",
		},
		"family": map[string]interface{}{
			"max_members":         20,
			"storage_limit_gb":    10,
			"supports_multi_baby": true,
			"has_pdf_export":      true,
			"price_monthly":       49.90,
			"plan_name":           "家庭版",
		},
	}

	response.Success(c, plans)
}

// RecordPayment 记录支付（模拟）
func (h *SubscriptionHandler) RecordPayment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req struct {
		Plan   string  `json:"plan" binding:"required"`
		Amount float64 `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 模拟支付成功，创建订阅
	if err := h.subscriptionRepo.RecordPayment(uint(userID), req.Plan, req.Amount); err != nil {
		h.logger.Error("Failed to record payment", zap.Error(err))
		response.ServerError(c, "支付失败")
		return
	}

	h.logger.Info("Recorded payment",
		zap.Uint("user_id", uint(userID)),
		zap.String("plan", req.Plan),
		zap.Float64("amount", req.Amount))

	response.Success(c, gin.H{
		"message": "支付成功",
	})
}

// RenewSubscription 续期订阅
func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	months := 1
	var req struct {
		Months int `json:"months"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	if req.Months > 0 {
		months = req.Months
	}

	// 获取活跃订阅
	subscription, err := h.subscriptionRepo.GetActiveByUserID(uint(userID))
	if err != nil {
		response.Error(c, http.StatusNotFound, "没有找到活跃订阅")
		return
	}

	// 续期
	if err := h.subscriptionRepo.RenewMultipleMonths(subscription.ID, months); err != nil {
		h.logger.Error("Failed to renew subscription", zap.Error(err))
		response.ServerError(c, "续期失败")
		return
	}

	h.logger.Info("Renewed subscription",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("subscription_id", subscription.ID),
		zap.Int("months", months))

	response.Success(c, gin.H{
		"message": "续期成功",
	})
}

// CancelSubscription 取消订阅
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取活跃订阅
	subscription, err := h.subscriptionRepo.GetActiveByUserID(uint(userID))
	if err != nil {
		response.Error(c, http.StatusNotFound, "没有找到活跃订阅")
		return
	}

	// 取消订阅
	if err := h.subscriptionRepo.ExpireSubscription(subscription.ID); err != nil {
		h.logger.Error("Failed to cancel subscription", zap.Error(err))
		response.ServerError(c, "取消订阅失败")
		return
	}

	h.logger.Info("Cancelled subscription",
		zap.Uint("user_id", uint(userID)),
		zap.Uint("subscription_id", subscription.ID))

	response.Success(c, gin.H{
		"message": "订阅已取消",
	})
}

// GetUsageSummary 获取使用情况汇总
func (h *SubscriptionHandler) GetUsageSummary(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取统计数据（需要替换为实际的数据源）
	// TODO: 添加 statsRepo 或其他方式获取统计数据
	var stats map[string]interface{} = make(map[string]interface{})
	stats["total_records"] = 0
	stats["storage_used_bytes"] = 0

	// 获取配额
	quota, err := h.subscriptionRepo.GetStorageQuota("free") // 默认使用免费版配额
	if err != nil {
		h.logger.Error("Failed to get quota", zap.Error(err))
		response.ServerError(c, "获取配额失败")
		return
	}

	// 计算使用百分比
	usagePercent := float64(stats["storage_used_bytes"].(int64)) / float64(quota) * 100

	response.Success(c, gin.H{
		"total_records":             stats["total_records"],
		"storage_used_gb":           float64(stats["storage_used_bytes"].(int64)) / 1024 / 1024 / 1024,
		"storage_used_gb_formatted": stats["storage_used_bytes"].(int64) / 1024 / 1024 / 1024,
		"storage_limit_gb":          quota / 1024 / 1024 / 1024,
		"usage_percent":             usagePercent,
	})
}
