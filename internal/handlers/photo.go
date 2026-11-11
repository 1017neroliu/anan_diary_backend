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

// PhotoHandler 宝宝照片（家庭相册）处理器
// 模型：照片挂在 baby 维度，家庭成员共享；
// 展示层每次打开页面按"1+点赞数"权重随机抽样散布在两侧空白区。
type PhotoHandler struct {
	babyRepo    *repository.BabyRepository
	authz       *service.AuthzService
	db          *gorm.DB
	logger      *zap.Logger
	maxUploadMB int
}

func NewPhotoHandler(babyRepo *repository.BabyRepository, authz *service.AuthzService, logger *zap.Logger, db *gorm.DB, maxUploadMB int) *PhotoHandler {
	return &PhotoHandler{
		babyRepo:    babyRepo,
		authz:       authz,
		db:          db,
		logger:      logger,
		maxUploadMB: maxUploadMB,
	}
}

// photoItem 列表项（附带点赞统计与"我是否已赞"）
type photoItem struct {
	ID           uint      `json:"id"`
	URL          string    `json:"url"`
	UploaderID   uint      `json:"uploader_id"`
	UploaderName string    `json:"uploader_name"`
	LikeCount    int64     `json:"like_count"`
	LikedByMe    bool      `json:"liked_by_me"`
	CreatedAt    time.Time `json:"created_at"`
}

// List 照片列表（按上传时间升序，前端轮播/散布顺序稳定）
// GET /api/photo/:baby_id
func (h *PhotoHandler) List(c *gin.Context) {
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
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权查看相册")
		return
	}

	// make 初始化保证空结果序列化为 [] 而非 null
	items := make([]photoItem, 0)
	if err := h.db.Table("baby_photos AS p").
		Select(`p.id, p.image_path AS url, p.uploader_id,
			COALESCE(u.name, u.email) AS uploader_name, p.created_at,
			(SELECT COUNT(*) FROM photo_likes l WHERE l.photo_id = p.id) AS like_count,
			EXISTS(SELECT 1 FROM photo_likes m WHERE m.photo_id = p.id AND m.user_id = ?) AS liked_by_me`,
			userID).
		Joins("LEFT JOIN users u ON u.id = p.uploader_id").
		Where("p.baby_id = ?", babyID).
		Order("p.created_at ASC").
		Find(&items).Error; err != nil {
		h.logger.Error("Failed to list photos", zap.Error(err))
		response.ServerError(c, "获取相册失败")
		return
	}

	response.Success(c, items)
}

// Upload 上传照片（支持多文件，字段名 photos）
// POST /api/photo/:baby_id
func (h *PhotoHandler) Upload(c *gin.Context) {
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
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权上传照片")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		response.ValidationError(c, "请选择要上传的照片")
		return
	}
	files := form.File["photos"]
	if len(files) == 0 {
		response.ValidationError(c, "请选择要上传的照片")
		return
	}
	if len(files) > 20 {
		response.ValidationError(c, "一次最多上传 20 张照片")
		return
	}

	uploadDir := "uploads/photos"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("Failed to create photos dir", zap.Error(err))
		response.ServerError(c, "创建上传目录失败")
		return
	}

	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	type failItem struct {
		Filename string `json:"filename"`
		Reason   string `json:"reason"`
	}
	// 用 make 初始化：Go nil slice 会被序列化为 null，前端遍历 length 会报错；空切片则序列化为 []
	uploaded := make([]photoItem, 0, len(files))
	failed := make([]failItem, 0)

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !allowedExts[ext] {
			failed = append(failed, failItem{Filename: file.Filename, Reason: "只支持 JPG/PNG/GIF/WEBP"})
			continue
		}
		if file.Size > int64(h.maxUploadMB)*1024*1024 {
			failed = append(failed, failItem{Filename: file.Filename, Reason: fmt.Sprintf("超过 %dMB 限制", h.maxUploadMB)})
			continue
		}

		filename := fmt.Sprintf("photo_%d_%d%s", babyID, time.Now().UnixNano(), ext)
		filePath := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			h.logger.Error("Failed to save photo", zap.String("filename", file.Filename), zap.Error(err))
			failed = append(failed, failItem{Filename: file.Filename, Reason: "保存失败"})
			continue
		}

		photo := models.BabyPhoto{
			BabyID:     uint(babyID),
			UploaderID: uint(userID),
			ImagePath:  fmt.Sprintf("/%s/%s", uploadDir, filename),
		}
		if err := h.db.Create(&photo).Error; err != nil {
			os.Remove(filePath)
			failed = append(failed, failItem{Filename: file.Filename, Reason: "保存失败"})
			continue
		}

		uploaded = append(uploaded, photoItem{
			ID:           photo.ID,
			URL:          photo.ImagePath,
			UploaderID:   photo.UploaderID,
			UploaderName: "",
			CreatedAt:    photo.CreatedAt,
		})
	}

	h.logger.Info("Photos uploaded",
		zap.Uint("baby_id", uint(babyID)),
		zap.Int("success", len(uploaded)),
		zap.Int("failed", len(failed)))

	response.Success(c, gin.H{
		"uploaded": uploaded,
		"failed":   failed,
	})
}

// Delete 删除照片（上传者本人或家庭管理员）
// DELETE /api/photo/:id
func (h *PhotoHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	photoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的照片 ID")
		return
	}

	var photo models.BabyPhoto
	if err := h.db.First(&photo, photoID).Error; err != nil {
		response.NotFound(c, "照片不存在")
		return
	}

	if photo.UploaderID != uint(userID) {
		if err := h.authz.RequireAdmin(uint(userID), photo.BabyID); err != nil {
			response.Forbidden(c, "只有上传者或家庭管理员可以删除照片")
			return
		}
	}

	// 先删记录，再删物理文件（文件残留无害，记录为准）
	if err := h.db.Delete(&models.BabyPhoto{}, photoID).Error; err != nil {
		h.logger.Error("Failed to delete photo record", zap.Error(err))
		response.ServerError(c, "删除失败")
		return
	}
	if err := os.Remove(strings.TrimPrefix(photo.ImagePath, "/")); err != nil && !os.IsNotExist(err) {
		h.logger.Warn("Failed to remove photo file", zap.String("path", photo.ImagePath), zap.Error(err))
	}

	h.logger.Info("Photo deleted", zap.Uint("photo_id", uint(photoID)), zap.Uint("operator_id", uint(userID)))
	response.Success(c, nil)
}

// Like 点赞照片
// POST /api/photo-like/:id
func (h *PhotoHandler) Like(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	photoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的照片 ID")
		return
	}

	var photo models.BabyPhoto
	if err := h.db.First(&photo, photoID).Error; err != nil {
		response.NotFound(c, "照片不存在")
		return
	}
	if err := h.authz.RequireMember(uint(userID), photo.BabyID); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员")
		return
	}

	like := models.PhotoLike{PhotoID: uint(photoID), UserID: uint(userID)}
	if err := h.db.Create(&like).Error; err != nil {
		// 唯一约束冲突 = 重复点赞，按幂等成功处理
		response.Success(c, gin.H{"message": "已点赞", "liked": true})
		return
	}

	response.Success(c, gin.H{"message": "点赞成功", "liked": true})
}

// Unlike 取消点赞
// DELETE /api/photo-like/:id
func (h *PhotoHandler) Unlike(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	photoID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的照片 ID")
		return
	}

	if err := h.db.Where("photo_id = ? AND user_id = ?", photoID, userID).
		Delete(&models.PhotoLike{}).Error; err != nil {
		h.logger.Error("Failed to unlike photo", zap.Error(err))
		response.ServerError(c, "取消点赞失败")
		return
	}

	response.Success(c, gin.H{"message": "已取消点赞", "liked": false})
}
