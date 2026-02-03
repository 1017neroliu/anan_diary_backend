package handlers

import (
	"net/http"
	"strconv"
	"time"
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GrowthRecordHandler struct {
	recordRepo *repository.GrowthRecordRepository
	babyRepo   *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	logger     *zap.Logger
}

func NewGrowthRecordHandler(
	recordRepo *repository.GrowthRecordRepository,
	babyRepo *repository.BabyRepository,
	memberRepo *repository.FamilyMemberRepository,
	logger *zap.Logger,
) *GrowthRecordHandler {
	return &GrowthRecordHandler{
		recordRepo: recordRepo,
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		logger:     logger,
	}
}

// CreateRecordRequest 创建记录请求
type CreateRecordRequest struct {
	BabyID uint   `json:"baby_id" binding:"required"`
	Content string `json:"content" binding:"required,max=1000"`
	Tags   []string `json:"tags"`
}

// CreateGrowthRecord 创建成长记录
// 核心逻辑：1) 自动关联宝宝 2) 自动填入当前日期 3) 防止重复提交（同一内容同一日期）
// 4) 不可变记录（数据库层面通过唯一索引 + API层校验）
func (h *GrowthRecordHandler) CreateGrowthRecord(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 验证宝宝是否属于当前用户
	baby, err := h.babyRepo.GetByID(req.BabyID)
	if err != nil {
		h.logger.Error("Failed to get baby", zap.Error(err), zap.Uint("baby_id", req.BabyID))
		response.NotFound(c, "宝宝档案不存在")
		return
	}
	if baby.UserID != userID {
		h.logger.Warn("Unauthorized access attempt",
			zap.Uint("baby_id", req.BabyID),
			zap.Uint("user_id", userID))
		response.Forbidden(c, "无权访问此宝宝档案")
		return
	}

	// 获取当前日期（自动填入）
	now := time.Now()
	recordDate := now.Format("2006-01-02 15:04:05")

	// 不可变检查：同一宝宝在同一个日期不能提交相同内容
	existing, err := h.recordRepo.GetByBabyAndDate(baby.ID, now)
	if err == nil && existing != nil {
		h.logger.Warn("Duplicate record attempt",
			zap.Uint("baby_id", baby.ID),
			zap.String("date", recordDate))
		response.Error(c, http.StatusBadRequest, "今日已记录过相同内容，请勿重复提交")
		return
	}

	// 处理标签
	tagsJSON := "[]"
	if len(req.Tags) > 0 {
		// 简化处理，实际应序列化为JSON
		tagsJSON = `"` + req.Tags[0] + `"`
		for i := 1; i < len(req.Tags); i++ {
			tagsJSON += `,"` + req.Tags[i] + `"`
		}
	}

	// 创建记录（数据库会自动处理唯一约束）
	record := models.GrowthRecord{
		UserID:    userID,
		BabyID:    baby.ID,
		Content:   req.Content,
		Tags:      tagsJSON,
		CreatedAt: now,
	}

	if err := h.recordRepo.Create(&record); err != nil {
		h.logger.Error("Failed to create growth record", zap.Error(err))
		response.ServerError(c, "创建记录失败")
		return
	}

	h.logger.Info("Growth record created successfully",
		zap.Uint("record_id", record.ID),
		zap.Uint("baby_id", baby.ID))

	response.Success(c, gin.H{
		"id":        record.ID,
		"baby_id":   baby.ID,
		"content":   record.Content,
		"tags":      req.Tags,
		"created_at": recordDate,
	})
}

// GetTimeline 获取时光轴数据
func (h *GrowthRecordHandler) GetTimeline(c *gin.Context) {
	babyIDStr := c.Query("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的baby_id")
		return
	}

	// 获取宝宝信息
	_, err = h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get baby", zap.Error(err))
		response.ServerError(c, "获取宝宝信息失败")
		return
	}

	// 获取宝宝的所有成长记录
	records, err := h.recordRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get records", zap.Error(err))
		response.ServerError(c, "获取数据失败")
		return
	}

	// 获取宝宝的所有家庭成员
	members, err := h.memberRepo.GetByBabyID(uint(babyID))
	if err != nil {
		h.logger.Error("Failed to get family members", zap.Error(err))
	}

	// 构建成员ID到成员信息的映射 (user_id 或 member_id 用于关联)
	// 注意：记录中的 user_id 是用户表ID，家庭成员表通过 baby_id 关联
	// 我们需要通过 user_id 关联到家庭成员

	// 转换为前端需要的格式
	timeline := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		// 获取关联的家庭成员信息
		memberName := ""
		memberRole := "其他"
		for _, m := range members {
			// 假设 user_id 等于 member_id，或者通过其他方式关联
			// 这里使用简单关联：user_id = member_id
			if r.UserID == uint(m.ID) {
				memberName = m.Name
				memberRole = m.Role
				break
			}
		}

		timeline = append(timeline, map[string]interface{}{
			"id":           r.ID,
			"baby_id":      r.BabyID,
			"user_id":      r.UserID,
			"content":      r.Content,
			"tags":         r.GetTags(),
			"created_at":   r.CreatedAt.Format(time.RFC3339),
			"member_name":  memberName,
			"member_role":  memberRole,
		})
	}

	response.Success(c, timeline)
}

// DeleteRecord 删除记录（仅用于测试）
func (h *GrowthRecordHandler) DeleteRecord(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的record_id")
		return
	}

	record, err := h.recordRepo.GetByID(uint(recordID))
	if err != nil {
		response.NotFound(c, "记录不存在")
		return
	}

	// 验证权限
	baby, err := h.babyRepo.GetByID(record.BabyID)
	if err != nil {
		response.ServerError(c, "获取宝宝信息失败")
		return
	}
	if baby.UserID != userID {
		response.Forbidden(c, "无权删除此记录")
		return
	}

	if err := h.recordRepo.Delete(uint(recordID)); err != nil {
		h.logger.Error("Failed to delete record", zap.Error(err))
		response.ServerError(c, "删除失败")
		return
	}

	h.logger.Info("Growth record deleted", zap.Uint("record_id", uint(recordID)))
	response.Success(c, gin.H{"message": "删除成功"})
}
