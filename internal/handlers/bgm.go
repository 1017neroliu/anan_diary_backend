package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BGMHandler 背景音乐处理器
// 模型：家庭成员共享的歌单（按 baby 维度），成员可上传，上传者或管理员可删除。
type BGMHandler struct {
	babyRepo    *repository.BabyRepository
	authz       *service.AuthzService
	db          *gorm.DB
	logger      *zap.Logger
	maxUploadMB int
}

func NewBGMHandler(babyRepo *repository.BabyRepository, authz *service.AuthzService, logger *zap.Logger, db *gorm.DB, maxUploadMB int) *BGMHandler {
	return &BGMHandler{
		babyRepo:    babyRepo,
		authz:       authz,
		db:          db,
		logger:      logger,
		maxUploadMB: maxUploadMB,
	}
}

// bgmListItem 列表项（附带上传者昵称）
type bgmListItem struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	UploaderID   uint      `json:"uploader_id"`
	UploaderName string    `json:"uploader_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// List 获取宝宝的 BGM 列表
// GET /api/bgm/:baby_id
func (h *BGMHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID, err := strconv.ParseUint(c.Param("baby_id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	if _, err := h.babyRepo.GetByID(uint(babyID)); err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}
	if err := h.authz.RequireMember(uint(userID), uint(babyID)); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权查看歌单")
		return
	}

	type row struct {
		ID           uint      `json:"id"`
		Title        string    `json:"title"`
		FilePath     string    `json:"file_path"`
		UploaderID   uint      `json:"uploader_id"`
		UploaderName string    `json:"uploader_name"`
		CreatedAt    time.Time `json:"created_at"`
	}
	var rows []row
	if err := h.db.Table("bgms").
		Select("bgms.id, bgms.title, bgms.file_path, bgms.uploader_id, COALESCE(users.name, users.email) AS uploader_name, bgms.created_at").
		Joins("LEFT JOIN users ON users.id = bgms.uploader_id").
		Where("bgms.baby_id = ?", babyID).
		Order("bgms.created_at ASC").
		Find(&rows).Error; err != nil {
		h.logger.Error("Failed to list bgms", zap.Error(err))
		response.ServerError(c, "获取歌单失败")
		return
	}

	list := make([]bgmListItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, bgmListItem{
			ID:           r.ID,
			Title:        r.Title,
			URL:          r.FilePath,
			UploaderID:   r.UploaderID,
			UploaderName: r.UploaderName,
			CreatedAt:    r.CreatedAt,
		})
	}

	response.Success(c, list)
}

// Upload 上传 BGM
// POST /api/bgm/:baby_id  (multipart: file)
func (h *BGMHandler) Upload(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyID, err := strconv.ParseUint(c.Param("baby_id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	if _, err := h.babyRepo.GetByID(uint(babyID)); err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}
	if err := h.authz.RequireMember(uint(userID), uint(babyID)); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权上传音乐")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ValidationError(c, "请选择音频文件")
		return
	}

	// 扩展名白名单
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".mp3": true, ".m4a": true, ".ogg": true, ".wav": true, ".flac": true}
	if !allowedExts[ext] {
		response.ValidationError(c, "只支持 MP3、M4A、OGG、WAV、FLAC 格式的音频")
		return
	}

	// 大小限制
	if file.Size > int64(h.maxUploadMB)*1024*1024 {
		response.ValidationError(c, fmt.Sprintf("音频文件不能超过 %dMB", h.maxUploadMB))
		return
	}

	// 保存文件
	uploadDir := "uploads/bgm"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("Failed to create bgm upload dir", zap.Error(err))
		response.ServerError(c, "创建上传目录失败")
		return
	}
	filename := fmt.Sprintf("bgm_%d_%d%s", babyID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		h.logger.Error("Failed to save bgm file", zap.Error(err))
		response.ServerError(c, "保存音频失败")
		return
	}

	// 曲名：上传文件名去扩展名（去空格），超长截断
	title := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	title = strings.TrimSpace(title)
	if title == "" {
		title = filename
	}
	if len([]rune(title)) > 100 {
		title = string([]rune(title)[:100])
	}

	bgm := models.BGM{
		BabyID:     uint(babyID),
		UploaderID: uint(userID),
		Title:      title,
		FilePath:   fmt.Sprintf("/%s/%s", uploadDir, filename),
	}
	if err := h.db.Create(&bgm).Error; err != nil {
		// 入库失败时回滚物理文件，避免孤儿文件
		os.Remove(filePath)
		h.logger.Error("Failed to create bgm record", zap.Error(err))
		response.ServerError(c, "保存歌单记录失败")
		return
	}

	h.logger.Info("BGM uploaded",
		zap.Uint("baby_id", uint(babyID)),
		zap.Uint("uploader_id", uint(userID)),
		zap.String("title", bgm.Title))

	response.Success(c, bgmListItem{
		ID:           bgm.ID,
		Title:        bgm.Title,
		URL:          bgm.FilePath,
		UploaderID:   bgm.UploaderID,
		UploaderName: "",
		CreatedAt:    bgm.CreatedAt,
	})
}

// Delete 删除 BGM（上传者本人或管理员）
// DELETE /api/bgm/:id
func (h *BGMHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	bgmID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的音乐 ID")
		return
	}

	var bgm models.BGM
	if err := h.db.First(&bgm, bgmID).Error; err != nil {
		response.NotFound(c, "音乐不存在")
		return
	}

	// 权限：上传者本人 或 家庭管理员
	if bgm.UploaderID != uint(userID) {
		if err := h.authz.RequireAdmin(uint(userID), bgm.BabyID); err != nil {
			response.Forbidden(c, "只有上传者或家庭管理员可以删除")
			return
		}
	}

	// 删除物理文件（相对 cwd 的路径，与上传时的 "uploads/bgm" 一致）
	if err := os.Remove(strings.TrimPrefix(bgm.FilePath, "/")); err != nil && !os.IsNotExist(err) {
		h.logger.Warn("Failed to remove bgm file", zap.String("path", bgm.FilePath), zap.Error(err))
	}

	if err := h.db.Delete(&models.BGM{}, bgmID).Error; err != nil {
		h.logger.Error("Failed to delete bgm record", zap.Error(err))
		response.ServerError(c, "删除失败")
		return
	}

	h.logger.Info("BGM deleted", zap.Uint("bgm_id", uint(bgmID)), zap.Uint("operator_id", uint(userID)))
	response.Success(c, nil)
}
