package handlers

import (
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo   *repository.UserRepository
	babyRepo   *repository.BabyRepository
	memberRepo *repository.FamilyMemberRepository
	db         *gorm.DB
	logger     *zap.Logger
}

func NewAuthHandler(userRepo *repository.UserRepository, babyRepo *repository.BabyRepository, memberRepo *repository.FamilyMemberRepository, logger *zap.Logger, db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		db:         db,
		logger:     logger,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=爸爸 妈妈 爷爷 奶奶 姥姥 姥爷 哥哥 姐姐"`
	Name     string `json:"name" binding:"required"`    // 家庭成员名字
	BabyID   string `json:"baby_id" binding:"required"` // 出生证编号，用于关联或创建宝宝
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 检查邮箱是否已注册
	existingUser, err := h.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		h.logger.Warn("Email already registered", zap.String("email", req.Email))
		response.Error(c, http.StatusBadRequest, "该邮箱已被注册")
		return
	}

	// 查询是否已存在相同的 baby_id
	existingBaby, err := h.babyRepo.GetByBabyID(req.BabyID)
	if err != nil {
		// record not found 是正常情况，说明宝宝不存在，可以继续创建
		if err.Error() != "record not found" {
			h.logger.Error("Failed to query baby by baby_id", zap.Error(err))
			response.ServerError(c, "查询宝宝档案失败")
			return
		}
		existingBaby = nil
	}

	// 检查是否已有该角色的家庭成员
	if existingBaby != nil {
		members, err := h.memberRepo.GetByBabyID(existingBaby.ID)
		if err != nil {
			h.logger.Error("Failed to get family members", zap.Error(err))
			response.ServerError(c, "查询家庭成员失败")
			return
		}

		for _, member := range members {
			if member.Role == req.Role {
				h.logger.Warn("Role already exists for baby", zap.String("baby_id", req.BabyID), zap.String("role", req.Role))
				response.Error(c, http.StatusBadRequest, fmt.Sprintf("该宝宝已有%s，请勿重复注册", req.Role))
				return
			}
		}
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	// 创建用户
	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         req.Role,
		Name:         req.Name,
	}

	if err := h.userRepo.Create(user); err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	// 注册即签发 token，前端无需再次登录即可进入应用
	token, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	// 根据是否已存在宝宝来决定是创建还是关联
	if existingBaby != nil {
		// 已存在宝宝，创建家庭成员关联
		member := &models.FamilyMember{
			BabyID: existingBaby.ID,
			UserID: &user.ID,
			Name:   req.Name,
			Role:   req.Role,
		}

		if err := h.memberRepo.Create(member); err != nil {
			h.logger.Error("Failed to create family member", zap.Error(err))
			response.ServerError(c, "注册失败")
			return
		}

		h.logger.Info("Linked family member to existing baby",
			zap.Uint("user_id", user.ID),
			zap.String("email", user.Email),
			zap.String("name", req.Name),
			zap.String("role", req.Role),
			zap.Uint("baby_id", existingBaby.ID))

		response.Success(c, gin.H{
			"token":   token,
			"user_id": user.ID,
			"email":   user.Email,
			"name":    req.Name,
			"role":    req.Role,
			"baby_id": existingBaby.ID,
			"message": "注册成功，已关联宝宝档案",
		})
		return
	}

	// 不存在宝宝，创建宝宝档案并创建家庭成员关联
	baby := &models.Baby{
		UserID:    user.ID,
		Name:      req.Name,                        // 使用用户的名字作为宝宝名字
		BirthDate: time.Now().Format("2006-01-02"), // 使用当前日期作为出生日期
		BabyID:    req.BabyID,
	}

	if err := h.babyRepo.Create(baby); err != nil {
		h.logger.Error("Failed to create baby", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	// 创建家庭成员记录
	member := &models.FamilyMember{
		BabyID: baby.ID,
		UserID: &user.ID,
		Name:   req.Name,
		Role:   req.Role,
	}

	if err := h.memberRepo.Create(member); err != nil {
		h.logger.Error("Failed to create family member", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	h.logger.Info("User registered and baby profile created",
		zap.Uint("user_id", user.ID),
		zap.String("email", user.Email),
		zap.String("name", req.Name),
		zap.String("role", req.Role),
		zap.Uint("baby_id", baby.ID))

	response.Success(c, gin.H{
		"token":   token,
		"user_id": user.ID,
		"email":   user.Email,
		"name":    req.Name,
		"role":    req.Role,
		"baby_id": baby.ID,
		"message": "注册成功，宝宝档案已创建",
	})
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "请求参数错误")
		return
	}

	// 查找用户
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		h.logger.Warn("Login failed: user not found", zap.String("email", req.Email))
		response.Error(c, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.logger.Warn("Login failed: wrong password", zap.String("email", req.Email))
		response.Error(c, http.StatusUnauthorized, "邮箱或密码错误")
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		response.ServerError(c, "登录失败")
		return
	}

	// 查询用户的宝宝列表
	var babyIDs []uint
	type MemberRole struct {
		BabyID uint
		UserID *uint
	}
	var members []MemberRole
	err = h.db.Table("family_members").Select("baby_id, user_id").Where("user_id = ?", user.ID).Find(&members).Error
	if err == nil {
		for _, m := range members {
			babyIDs = append(babyIDs, m.BabyID)
		}
	}

	// 查询用户自己创建的宝宝
	var myBabies []struct {
		ID uint
	}
	err = h.db.Table("babies").Where("user_id = ?", user.ID).Select("id").Find(&myBabies).Error
	if err == nil {
		for _, b := range myBabies {
			if !contains(babyIDs, b.ID) {
				babyIDs = append(babyIDs, b.ID)
			}
		}
	}

	// 取第一个宝宝ID（如果有）
	var babyID *uint
	if len(babyIDs) > 0 {
		babyID = &babyIDs[0]
	}

	h.logger.Info("User logged in successfully",
		zap.Uint("user_id", user.ID),
		zap.Int("baby_count", len(babyIDs)))

	response.Success(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"email":    user.Email,
		"role":     user.Role,
		"baby_id":  babyID,
		"has_baby": len(babyIDs) > 0,
	})
}

// contains 辅助函数：检查uint切片是否包含某个值
func contains(slice []uint, item uint) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetCurrentUser 获取当前登录用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetInt("user_id")
	h.logger.Info("GetCurrentUser called", zap.Uint("user_id", uint(userID)))

	if userID == 0 {
		h.logger.Warn("UserID is 0, token may not be valid")
		response.Error(c, http.StatusUnauthorized, "无效的token")
		return
	}

	user, err := h.userRepo.FindByID(uint(userID))
	if err != nil {
		h.logger.Warn("Failed to get current user", zap.Error(err))
		response.Error(c, http.StatusUnauthorized, "获取用户信息失败")
		return
	}

	h.logger.Info("GetCurrentUser success", zap.Uint("id", user.ID), zap.String("email", user.Email))

	response.Success(c, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
		"name":  user.Name,
	})
}
