package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BabyHandler struct {
	babyRepo   *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	authz      *service.AuthzService
	db         *gorm.DB
	logger     *zap.Logger
}

func NewBabyHandler(babyRepo *repository.BabyRepository, memberRepo *repository.FamilyMemberRepository, authz *service.AuthzService, logger *zap.Logger, db *gorm.DB) *BabyHandler {
	return &BabyHandler{
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		authz:      authz,
		db:         db,
		logger:     logger,
	}
}

// parseBirthDate 将前端传来的出生日期（多种常见格式）解析为 "2006-01-02"。
// 空串与 HTTP 响应语义由调用方处理。
func parseBirthDate(raw string) (string, error) {
	// 尝试解析 ISO 格式（带时区）
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Format("2006-01-02"), nil
	}
	// 尝试解析其他常见格式
	if t, err := time.Parse("2006-01-02T15:04:05", raw); err == nil {
		return t.Format("2006-01-02"), nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("invalid birth date format: %s", raw)
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

// CreateBaby 创建宝宝档案或关联家庭成员
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

	// 解析出生日期，将前端发送的日期转换为 "2006-01-02" 格式
	if req.BirthDate == "" {
		response.Error(c, http.StatusBadRequest, "出生日期不能为空")
		return
	}
	birthDateStr, err := parseBirthDate(req.BirthDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的出生日期格式")
		return
	}

	// 检查出生证编号是否已存在
	existingBaby, err := h.babyRepo.GetByBabyID(req.BabyID)
	if err == nil && existingBaby != nil {
		// baby_id 已存在，检查是否已有该角色的家庭成员
		members, err := h.memberRepo.GetByBabyID(existingBaby.ID)
		if err != nil {
			h.logger.Error("Failed to get family members", zap.Error(err))
			response.ServerError(c, "获取家庭成员失败")
			return
		}

		// 检查是否已有该角色的家庭成员
		for _, member := range members {
			if member.Role == req.UserRole {
				response.Error(c, http.StatusBadRequest, fmt.Sprintf("该宝宝已有%s，请勿重复添加", req.UserRole))
				return
			}
		}

		// 创建家庭成员记录
		userIDUint := uint(userID)
		member := &models.FamilyMember{
			BabyID: existingBaby.ID,
			UserID: &userIDUint,
			Name:   req.UserName,
			Role:   req.UserRole,
		}

		if err := h.memberRepo.Create(member); err != nil {
			h.logger.Error("Failed to create family member", zap.Error(err))
			response.ServerError(c, "关联宝宝失败")
			return
		}

		h.logger.Info("Linked family member to existing baby",
			zap.Uint("baby_id", existingBaby.ID),
			zap.String("name", existingBaby.Name),
			zap.String("member_name", member.Name),
			zap.String("member_role", member.Role))

		response.Success(c, existingBaby)
		return
	}

	// 创建宝宝档案（第一次创建）
	baby := &models.Baby{
		UserID:    uint(userID),
		Name:      req.Name,
		BirthDate: birthDateStr,
		BabyID:    req.BabyID,
	}

	if err := h.babyRepo.Create(baby); err != nil {
		h.logger.Error("Failed to create baby", zap.Error(err))
		response.ServerError(c, "创建宝宝档案失败")
		return
	}

	// 获取当前用户的所有宝宝，以获取所有现有的家庭成员
	var userBabies []models.Baby
	if err := h.db.Where("user_id = ?", userID).Find(&userBabies).Error; err == nil {
		for _, existingBaby := range userBabies {
			// 获取该宝宝的所有家庭成员
			members, err := h.memberRepo.GetByBabyID(existingBaby.ID)
			if err == nil {
				// 将所有现有家庭成员链接到新创建的宝宝
				for _, member := range members {
					// 避免重复添加当前用户
					if member.UserID != nil && *member.UserID == uint(userID) {
						continue
					}
					newMember := &models.FamilyMember{
						BabyID: baby.ID,
						Name:   member.Name,
						Role:   member.Role,
					}
					if err := h.memberRepo.Create(newMember); err != nil {
						h.logger.Warn("Failed to create family member link",
							zap.Error(err),
							zap.String("role", member.Role),
							zap.Uint("baby_id", baby.ID))
					}
				}
			}
		}
	}

	// 创建家庭成员记录（当前用户）
	userIDUint := uint(userID)
	member := &models.FamilyMember{
		BabyID: baby.ID,
		UserID: &userIDUint,
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

// GetBabies 获取宝宝列表（包括家庭成员的宝宝）
func (h *BabyHandler) GetBabies(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取宝宝列表：先获取当前用户的宝宝，再获取家庭共享的宝宝
	myBabies, err := h.babyRepo.GetByUserID(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get babies", zap.Error(err))
		response.ServerError(c, "获取宝宝列表失败")
		return
	}

	// 获取所有家庭成员关联的宝宝 ID（通过 family_members 表）
	var babyIDs []uint
	err = h.db.Table("family_members").
		Select("DISTINCT baby_id").
		Where("user_id = ?", userID).
		Scan(&babyIDs).Error

	if err != nil {
		h.logger.Error("Failed to get family member baby IDs", zap.Error(err))
	}

	// 查询所有家庭成员的宝宝
	var sharedBabies []models.Baby
	if len(babyIDs) > 0 {
		err = h.db.Where("id IN ?", babyIDs).Find(&sharedBabies).Error
		if err != nil {
			h.logger.Error("Failed to get shared babies", zap.Error(err))
		}
	}

	// 合并并去重
	allBabies := make([]models.Baby, 0)
	babyIDMap := make(map[uint]bool)

	// 先添加当前用户创建的宝宝
	for _, baby := range myBabies {
		if !babyIDMap[baby.ID] {
			allBabies = append(allBabies, baby)
			babyIDMap[baby.ID] = true
		}
	}

	// 再添加通过家庭成员关系关联的宝宝
	for _, baby := range sharedBabies {
		if !babyIDMap[baby.ID] {
			allBabies = append(allBabies, baby)
			babyIDMap[baby.ID] = true
		}
	}

	response.Success(c, allBabies)
}

// GetBabyByID 获取宝宝详细信息（家庭成员也可以访问）
func (h *BabyHandler) GetBabyByID(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 权限校验：走 authz.RequireMember（创建者 + 权限表 + family_members 三路兜底），
	// 不再手动查 family_members——否则通过审批/权限表加入的成员会被误判为非成员而 403。
	if err := h.authz.RequireMember(uint(userID), baby.ID); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权访问此宝宝档案")
		return
	}

	response.Success(c, baby)
}

// UpdateBabyRequest 更新宝宝档案请求（出生证编号是身份键兼加入凭据，不允许修改）
type UpdateBabyRequest struct {
	Name      string `json:"name" binding:"required"`
	BirthDate string `json:"birth_date" binding:"required"`
}

// UpdateBaby 更新宝宝档案（昵称、出生日期）
func (h *BabyHandler) UpdateBaby(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 权限校验：走 authz.RequireMember（创建者 + 权限表 + family_members 三路兜底）。
	if err := h.authz.RequireMember(uint(userID), baby.ID); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权编辑此宝宝档案")
		return
	}

	var req UpdateBabyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	birthDateStr, err := parseBirthDate(req.BirthDate)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的出生日期格式")
		return
	}
	// birthDateStr 为 "2006-01-02" 定长格式，可直接字符串比较
	if birthDateStr > time.Now().Format("2006-01-02") {
		response.Error(c, http.StatusBadRequest, "出生日期不能晚于今天")
		return
	}

	baby.Name = req.Name
	baby.BirthDate = birthDateStr
	if err := h.babyRepo.Update(baby); err != nil {
		h.logger.Error("Failed to update baby", zap.Error(err))
		response.ServerError(c, "更新失败")
		return
	}

	h.logger.Info("Baby profile updated",
		zap.Uint("baby_id", baby.ID),
		zap.String("name", baby.Name),
		zap.String("birth_date", baby.BirthDate))

	response.Success(c, baby)
}

// DeleteBaby 删除宝宝档案（家庭成员也可以删除）
func (h *BabyHandler) DeleteBaby(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 权限校验：走 authz.RequireMember（创建者 + 权限表 + family_members 三路兜底）。
	if err := h.authz.RequireMember(uint(userID), baby.ID); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权删除此宝宝档案")
		return
	}

	if err := h.babyRepo.Delete(uint(babyID)); err != nil {
		h.logger.Error("Failed to delete baby", zap.Error(err))
		response.ServerError(c, "删除失败")
		return
	}

	// 删除关联的家庭成员
	h.db.Where("baby_id = ?", uint(babyID)).Delete(&models.FamilyMember{})

	// 删除关联的成长记录
	h.db.Where("baby_id = ?", uint(babyID)).Delete(&models.GrowthRecord{})

	h.logger.Info("Baby profile deleted", zap.Uint("baby_id", uint(babyID)))
	response.Success(c, gin.H{"message": "删除成功"})
}

// GetCurrentBaby 获取当前用户的宝宝信息（包括家庭共享的宝宝）
func (h *BabyHandler) GetCurrentBaby(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	// 获取宝宝列表：先获取当前用户的宝宝，再获取家庭共享的宝宝
	myBabies, err := h.babyRepo.GetByUserID(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get babies", zap.Error(err))
		response.ServerError(c, "获取宝宝列表失败")
		return
	}

	// 获取所有家庭成员关联的宝宝 ID（通过 family_members 表）
	var babyIDs []uint
	err = h.db.Table("family_members").
		Select("DISTINCT baby_id").
		Where("user_id = ?", userID).
		Scan(&babyIDs).Error

	if err != nil {
		h.logger.Error("Failed to get family member baby IDs", zap.Error(err))
	}

	// 查询所有家庭成员的宝宝
	var sharedBabies []models.Baby
	if len(babyIDs) > 0 {
		err = h.db.Where("id IN ?", babyIDs).Find(&sharedBabies).Error
		if err != nil {
			h.logger.Error("Failed to get shared babies", zap.Error(err))
		}
	}

	// 合并并去重
	allBabies := make([]models.Baby, 0)
	babyIDMap := make(map[uint]bool)

	// 先添加当前用户创建的宝宝
	for _, baby := range myBabies {
		if !babyIDMap[baby.ID] {
			allBabies = append(allBabies, baby)
			babyIDMap[baby.ID] = true
		}
	}

	// 再添加通过家庭成员关系关联的宝宝
	for _, baby := range sharedBabies {
		if !babyIDMap[baby.ID] {
			allBabies = append(allBabies, baby)
			babyIDMap[baby.ID] = true
		}
	}

	if len(allBabies) == 0 {
		response.Error(c, http.StatusNotFound, "请先创建宝宝档案")
		return
	}

	response.Success(c, allBabies[0])
}

// UploadAvatar 上传宝宝头像
func (h *BabyHandler) UploadAvatar(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	babyIDStr := c.Param("id")
	babyID, err := strconv.ParseUint(babyIDStr, 10, 32)
	if err != nil {
		response.ValidationError(c, "无效的 baby_id")
		return
	}

	// 验证权限
	baby, err := h.babyRepo.GetByID(uint(babyID))
	if err != nil {
		response.NotFound(c, "宝宝档案不存在")
		return
	}

	// 权限校验：走 authz.RequireMember（创建者 + 权限表 + family_members 三路兜底）。
	if err := h.authz.RequireMember(uint(userID), baby.ID); err != nil {
		response.Forbidden(c, "您不是该宝宝的家庭成员，无权上传头像")
		return
	}

	// 处理上传的文件
	file, err := c.FormFile("avatar")
	if err != nil {
		response.ValidationError(c, "请上传头像文件")
		return
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		response.ValidationError(c, "只支持 JPG、PNG、GIF、WEBP 格式的图片")
		return
	}

	// 创建上传目录
	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("Failed to create upload directory", zap.Error(err))
		response.ServerError(c, "创建上传目录失败")
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("baby_%d_%d%s", babyID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		h.logger.Error("Failed to save uploaded file", zap.Error(err))
		response.ServerError(c, "保存头像失败")
		return
	}

	// 生成访问 URL
	avatarURL := fmt.Sprintf("/%s/%s", uploadDir, filename)

	// 更新数据库
	if err := h.db.Model(&models.Baby{}).Where("id = ?", babyID).Update("avatar_url", avatarURL).Error; err != nil {
		h.logger.Error("Failed to update avatar URL", zap.Error(err))
		response.ServerError(c, "更新头像失败")
		return
	}

	h.logger.Info("Baby avatar uploaded",
		zap.Uint("baby_id", uint(babyID)),
		zap.String("avatar_url", avatarURL))

	response.Success(c, gin.H{
		"avatar_url": avatarURL,
	})
}
