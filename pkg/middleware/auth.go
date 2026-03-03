package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// Claims JWT声明
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "缺少认证信息",
			})
			c.Abort()
			return
		}

		// 从 "Bearer <token>" 中提取token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "认证格式错误",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			fmt.Printf("[DEBUG] Token parse error: %v\n", err)
			c.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "无效的token",
			})
			c.Abort()
			return
		}

		fmt.Printf("[DEBUG] Token claims type: %T, valid: %v\n", token.Claims, token.Valid)

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			fmt.Printf("[DEBUG] Token parsed successfully, user_id: %d, email: %s\n", claims.UserID, claims.Email)
			fmt.Printf("[DEBUG] Setting user_id to context as uint: %d\n", claims.UserID)
			// 将用户信息存入上下文 - 使用 int 类型
			c.Set("user_id", int(claims.UserID))
			c.Set("email", claims.Email)
			userID := c.GetInt("user_id")
			fmt.Printf("[DEBUG] After Set, user_id in context: %d\n", userID)
			if userID == 0 {
				fmt.Printf("[DEBUG] ERROR: user_id is 0 after setting to %d\n", claims.UserID)
			}
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "token无效",
			})
			c.Abort()
		}
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(int), true
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}
	return email.(string), true
}

// JWTClaims JWT claims结构
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

// GenerateToken 生成JWT token
func GenerateToken(userID uint, email string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(24 * time.Hour * 7) // 7天有效期

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
