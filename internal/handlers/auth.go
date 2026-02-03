package handlers

import (
	"net/http"
	"anan-diary/ad-backend/internal/models"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

type AuthHandler struct {
	userRepo    *repository.UserRepository
	babyRepo    *repository.BabyRepository
	memberRepo  *repository.FamilyMemberRepository
	logger      *zap.Logger
}

func NewAuthHandler(userRepo *repository.UserRepository, babyRepo *repository.BabyRepository, memberRepo *repository.FamilyMemberRepository, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo:   userRepo,
		babyRepo:   babyRepo,
		memberRepo: memberRepo,
		logger:     logger,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=爸爸 妈妈 爷爷 奶奶 姥姥 姥爷 哥哥 姐姐"`
	Name     string `json:"name" binding:"required"` // 家庭成员名字
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
	}

	if err := h.userRepo.Create(user); err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		response.ServerError(c, "注册失败")
		return
	}

	h.logger.Info("User registered successfully", zap.Uint("user_id", user.ID))
	response.Success(c, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  req.Name,
		"role":  req.Role,
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

	h.logger.Info("User logged in successfully", zap.Uint("user_id", user.ID))
	response.Success(c, gin.H{
		"token":  token,
		"user_id": user.ID,
		"email":  user.Email,
		"role":   user.Role,
	})
}

// GetCurrentUser 获取当前登录用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetInt("user_id")
	user, err := h.userRepo.FindByID(uint(userID))
	if err != nil {
		h.logger.Warn("Failed to get current user", zap.Error(err))
		response.Error(c, http.StatusUnauthorized, "获取用户信息失败")
		return
	}

	response.Success(c, gin.H{
		"id":      user.ID,
		"email":   user.Email,
		"role":    user.Role,
	})
}
