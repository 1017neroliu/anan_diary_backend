package middleware

import (
	"net/http"
	"strings"
	"time"

	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// jwtSecret 与 jwtExpire 由应用启动时通过 SetJWTConfig 注入，
// 不再硬编码在源码中。未注入时使用开发兜底值。
var (
	jwtSecret = []byte("dev-secret-key-change-in-production")
	jwtExpire = 7 * 24 * time.Hour
)

// SetJWTConfig 由 main.go 在启动时注入 JWT 密钥与过期时间。
func SetJWTConfig(secret string, expire time.Duration) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
	if expire > 0 {
		jwtExpire = expire
	}
}

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
			c.JSON(http.StatusUnauthorized, response.Response{
				Code:    401,
				Message: "无效的token",
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// 将用户信息存入上下文 - 使用 int 类型
			c.Set("user_id", int(claims.UserID))
			c.Set("email", claims.Email)
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
	id, ok := userID.(int)
	if !ok {
		return 0, false
	}
	return id, true
}

// GetEmail 从上下文获取邮箱
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	if !ok {
		return "", false
	}
	return e, true
}

// GenerateToken 生成JWT token
func GenerateToken(userID uint, email string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(jwtExpire)

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
