package handlers

import (
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatsHandler struct {
	statsRepo      *repository.StatsRepository
	permissionRepo *repository.PermissionRepository
	logger         *zap.Logger
}

func NewStatsHandler(
	statsRepo *repository.StatsRepository,
	permissionRepo *repository.PermissionRepository,
	logger *zap.Logger,
) *StatsHandler {
	return &StatsHandler{
		statsRepo:      statsRepo,
		permissionRepo: permissionRepo,
		logger:         logger,
	}
}

// GetUserStats 获取用户统计
func (h *StatsHandler) GetUserStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	stats, err := h.statsRepo.GetStats(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get user stats", zap.Error(err))
		response.ServerError(c, "获取统计数据失败")
		return
	}

	response.Success(c, stats)
}

// GetMemberContributions 获取宝宝成员贡献统计
func (h *StatsHandler) GetMemberContributions(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 获取贡献统计
	contributions, err := h.statsRepo.GetStatsByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get contributions", zap.Error(err))
		response.ServerError(c, "获取贡献统计失败")
		return
	}

	response.Success(c, contributions)
}

// GetDailyRecordCount 获取今日记录数
func (h *StatsHandler) GetDailyRecordCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	count, err := h.statsRepo.GetTodayRecordCount(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get daily record count", zap.Error(err))
		response.ServerError(c, "获取今日记录数失败")
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// GetWeeklyRecordCount 获取本周记录数
func (h *StatsHandler) GetWeeklyRecordCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	count, err := h.statsRepo.GetWeeklyRecordCount(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get weekly record count", zap.Error(err))
		response.ServerError(c, "获取本周记录数失败")
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// GetMonthlyRecordCount 获取本月记录数
func (h *StatsHandler) GetMonthlyRecordCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	count, err := h.statsRepo.GetMonthlyRecordCount(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get monthly record count", zap.Error(err))
		response.ServerError(c, "获取本月记录数失败")
		return
	}

	response.Success(c, gin.H{
		"count": count,
	})
}

// GetContributionRanking 获取贡献排行榜
func (h *StatsHandler) GetContributionRanking(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 获取贡献统计
	contributions, err := h.statsRepo.GetStatsByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get contributions", zap.Error(err))
		response.ServerError(c, "获取贡献统计失败")
		return
	}

	// 按记录数排序
	for i := 0; i < len(contributions); i++ {
		for j := i + 1; j < len(contributions); j++ {
			ci := contributions[i]
			cj := contributions[j]
			ri, _ := ci["record_count"].(int64)
			rj, _ := cj["record_count"].(int64)
			if ri < rj {
				contributions[i], contributions[j] = contributions[j], contributions[i]
			}
		}
	}

	response.Success(c, contributions)
}

// GetMemberHighlight 获取本周/本月亮点成员
func (h *StatsHandler) GetMemberHighlight(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 获取贡献统计
	contributions, err := h.statsRepo.GetStatsByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get contributions", zap.Error(err))
		response.ServerError(c, "获取贡献统计失败")
		return
	}

	// 计算总记录数
	var totalRecords int64
	for _, c := range contributions {
		recordCount, _ := c["record_count"].(int64)
		totalRecords += recordCount
	}

	// 计算每位成员的贡献占比
	for _, c := range contributions {
		recordCount, _ := c["record_count"].(int64)
		ratio := float64(recordCount) / float64(totalRecords) * 100
		c["contribution_ratio"] = ratio
	}

	// 找出贡献最多的成员
	var topMember interface{}
	var maxCount int64
	for _, c := range contributions {
		recordCount, _ := c["record_count"].(int64)
		if recordCount > maxCount {
			maxCount = recordCount
			topMember = c
		}
	}

	response.Success(c, gin.H{
		"contributions": contributions,
		"top_member":    topMember,
	})
}
