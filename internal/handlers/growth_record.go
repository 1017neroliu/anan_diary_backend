package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GrowthRecordHandler struct {
	recordRepo *repository.GrowthRecordRepository
	babyRepo   *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	authz      *service.AuthzService
	logger     *zap.Logger
	loc        *time.Location
}

func NewGrowthRecordHandler(
	recordRepo *repository.GrowthRecordRepository,
	babyRepo *repository.BabyRepository,
	memberRepo *repository.FamilyMemberRepository,
	authz *service.AuthzService,
	logger *zap.Logger,
) *GrowthRecordHandler {
	// 加载 Asia/Shanghai 时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Fatal("Failed to load Asia/Shanghai timezone", zap.Error(err))
	}
	return &GrowthRecordHandler{
		recordRepo: recordRepo,
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		authz:      authz,
		logger:     logger,
		loc:        loc,
	}
}

// MediaItem 媒体文件项
type MediaItem struct {
	Type string `json:"type"` // image, video
	URL  string `json:"url"`
}

// CreateRecordRequest 创建记录请求
type CreateRecordRequest struct {
	BabyID  uint        `json:"baby_id" binding:"required"`
	Content string      `json:"content" binding:"required,max=1000"`
	Tags    []string    `json:"tags"`
	Media   []MediaItem `json:"media"`
}

// CreateGrowthRecord 创建成长记录
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

	// 验证宝宝是否存在
	baby, err := h.babyRepo.GetByID(req.BabyID)
	if err != nil {
		h.logger.Error("Failed to get baby", zap.Error(err), zap.Uint("baby_id", req.BabyID))
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 权限校验：仅家庭成员可创建记录
	if err := h.authz.RequireMember(uint(userID), baby.ID); err != nil {
		h.logger.Warn("Unauthorized record creation attempt",
			zap.Uint("baby_id", req.BabyID),
			zap.Uint("user_id", uint(userID)))
		response.Forbidden(c, "您不是该宝宝的家庭成员，无法添加记录")
		return
	}

	// 获取当前日期
	now := time.Now()
	recordDate := now.Format("2006-01-02 15:04:05")

	// 不可变检查
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
		tagsJSON = `"` + req.Tags[0] + `"`
		for i := 1; i < len(req.Tags); i++ {
			tagsJSON += `,"` + req.Tags[i] + `"`
		}
	}
	tagsJSON = "[" + tagsJSON + "]"

	// 处理媒体文件
	mediaJSON := "[]"
	if len(req.Media) > 0 {
		mediaData, err := json.Marshal(req.Media)
		if err == nil {
			mediaJSON = string(mediaData)
		}
	}

	// 创建记录
	record := models.GrowthRecord{
		UserID:    uint(userID),
		BabyID:    baby.ID,
		Content:   req.Content,
		Tags:      tagsJSON,
		Media:     mediaJSON,
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

	// 数据库中的时间已经是 Asia/Shanghai 时间，但 GORM 将其视为 UTC
	// 所以需要创建一个固定时区（UTC+8）来正确表示
	shanghai := time.FixedZone("CST", 8*3600)
	localTime := time.Date(record.CreatedAt.Year(), record.CreatedAt.Month(), record.CreatedAt.Day(),
		record.CreatedAt.Hour(), record.CreatedAt.Minute(), record.CreatedAt.Second(),
		record.CreatedAt.Nanosecond(), shanghai)

	response.Success(c, gin.H{
		"id":         record.ID,
		"baby_id":    baby.ID,
		"content":    record.Content,
		"tags":       req.Tags,
		"media":      req.Media,
		"created_at": localTime.Format(time.RFC3339),
	})
}

// GetTimeline 获取时光轴数据
func (h *GrowthRecordHandler) GetTimeline(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Query("baby_id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 权限校验：仅家庭成员可查看时光轴
	if err := h.authz.RequireMember(uint(userID), uint(babyID)); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权查看时光轴")
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

	// 转换为前端需要的格式
	timeline := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		memberName := ""
		memberRole := "其他"
		for _, m := range members {
			if r.UserID != 0 && m.UserID != nil && r.UserID == *m.UserID {
				memberName = m.Name
				memberRole = m.Role
				break
			}
		}

		// 解析媒体文件
		var mediaItems []MediaItem
		if r.Media != "" && r.Media != "[]" {
			json.Unmarshal([]byte(r.Media), &mediaItems)
		}

		// 数据库中的时间已经是 Asia/Shanghai 时间，但 GORM 将其视为 UTC
		// 所以需要创建一个固定时区（UTC+8）来正确表示
		shanghai := time.FixedZone("CST", 8*3600)
		localTime := time.Date(r.CreatedAt.Year(), r.CreatedAt.Month(), r.CreatedAt.Day(),
			r.CreatedAt.Hour(), r.CreatedAt.Minute(), r.CreatedAt.Second(),
			r.CreatedAt.Nanosecond(), shanghai)

		timeline = append(timeline, map[string]interface{}{
			"id":          r.ID,
			"baby_id":     r.BabyID,
			"user_id":     r.UserID,
			"content":     r.Content,
			"tags":        r.GetTags(),
			"media":       mediaItems,
			"created_at":  localTime.Format(time.RFC3339),
			"member_name": memberName,
			"member_role": memberRole,
		})
	}

	response.Success(c, timeline)
}

// DeleteRecord 删除记录
func (h *GrowthRecordHandler) DeleteRecord(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 record_id")
		return
	}

	record, err := h.recordRepo.GetByID(uint(recordID))
	if err != nil {
		response.NotFound(c, "记录不存在")
		return
	}

	// 权限校验：管理员、拥有 delete_records 权限，或记录作者可删除
	canDelete, err := h.authz.CanDeleteRecord(uint(userID), record.BabyID, record.UserID)
	if err != nil || !canDelete {
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

// UploadMedia 上传媒体文件（图片/视频）
func (h *GrowthRecordHandler) UploadMedia(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 处理上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ValidationError(c, "请上传文件")
		return
	}

	// 文件大小限制（20MB）
	if file.Size > 20<<20 {
		response.ValidationError(c, "文件大小不能超过 20MB")
		return
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedImageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowedVideoExts := []string{".mp4", ".webm", ".mov", ".avi"}

	var mediaType string
	var uploadDir string

	isImage := false
	for _, allowedExt := range allowedImageExts {
		if ext == allowedExt {
			isImage = true
			break
		}
	}

	if isImage {
		mediaType = "image"
		uploadDir = "uploads/records/images"
	} else {
		isVideo := false
		for _, allowedExt := range allowedVideoExts {
			if ext == allowedExt {
				isVideo = true
				break
			}
		}
		if !isVideo {
			response.ValidationError(c, "只支持图片（JPG、PNG、GIF、WEBP）和视频（MP4、WebM、MOV、AVI）格式")
			return
		}
		mediaType = "video"
		uploadDir = "uploads/records/videos"
	}

	// 创建上传目录
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("Failed to create upload directory", zap.Error(err))
		response.ServerError(c, "创建上传目录失败")
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("record_%d_%d%s", userID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		h.logger.Error("Failed to save uploaded file", zap.Error(err))
		response.ServerError(c, "保存文件失败")
		return
	}

	// 生成访问 URL
	mediaURL := fmt.Sprintf("/%s/%s", uploadDir, filename)

	h.logger.Info("Media uploaded successfully",
		zap.Uint("user_id", uint(userID)),
		zap.String("media_type", mediaType),
		zap.String("media_url", mediaURL))

	response.Success(c, gin.H{
		"type": mediaType,
		"url":  mediaURL,
	})
}

// GetRecordMedia 获取记录的媒体文件（需为该宝宝成员）
func (h *GrowthRecordHandler) GetRecordMedia(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	recordIDStr := c.Param("id")
	recordID, err := strconv.ParseUint(recordIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 record_id")
		return
	}

	record, err := h.recordRepo.GetByID(uint(recordID))
	if err != nil {
		response.NotFound(c, "记录不存在")
		return
	}

	// 权限校验：仅家庭成员可查看媒体
	if err := h.authz.RequireMember(uint(userID), record.BabyID); err != nil {
		response.Forbidden(c, "无权查看此记录的媒体文件")
		return
	}

	// 解析媒体文件
	var mediaItems []MediaItem
	if record.Media != "" && record.Media != "[]" {
		if err := json.Unmarshal([]byte(record.Media), &mediaItems); err != nil {
			h.logger.Error("Failed to parse media", zap.Error(err))
			response.ServerError(c, "解析媒体文件失败")
			return
		}
	}

	response.Success(c, mediaItems)
}
