package handlers

import (
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MilestoneHandler struct {
	milestoneRepo *repository.MilestoneRepository
	logger        *zap.Logger
}

func NewMilestoneHandler(milestoneRepo *repository.MilestoneRepository, logger *zap.Logger) *MilestoneHandler {
	return &MilestoneHandler{
		milestoneRepo: milestoneRepo,
		logger:        logger,
	}
}

// GetMilestones 获取所有里程碑标签
func (h *MilestoneHandler) GetMilestones(c *gin.Context) {
	tags, err := h.milestoneRepo.GetAll()
	if err != nil {
		h.logger.Error("Failed to get milestones", zap.Error(err))
		response.ServerError(c, "获取里程碑失败")
		return
	}

	response.Success(c, tags)
}

// CheckMilestones 检查并返回匹配的里程碑
func (h *MilestoneHandler) CheckMilestones(c *gin.Context) {
	babyID := c.Query("baby_id")
	birthDate := c.Query("birth_date")

	if babyID == "" || birthDate == "" {
		response.ValidationError(c, "缺少必要参数")
		return
	}

	// 实际项目中应从数据库获取宝宝出生日期并计算
	// 这里简化处理，返回默认里程碑
	milestones := map[int]string{
		7:   "第一周",
		14:  "第二周",
		21:  "第三周",
		30:  "满月",
		42:  "六周",
		56:  "八周",
		100: "百日",
		90:  "三个月",
		180: "半岁",
		270: "九个月",
		365: "一周岁",
	}

	var matchingMilestones []string
	for offset, name := range milestones {
		if offset == 30 { // 简化处理：直接返回满月、百日等
			matchingMilestones = append(matchingMilestones, name)
		}
	}

	response.Success(c, matchingMilestones)
}
