package middleware

import (
	"testing"
	"time"
)

func init() {
	// 测试用固定密钥
	SetJWTConfig("test-secret-key", time.Hour)
}

func TestGenerateAndParseToken(t *testing.T) {
	userID := uint(42)
	email := "test@example.com"

	token, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() 返回空 token")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("claims.UserID = %d, want %d", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("claims.Email = %q, want %q", claims.Email, email)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	tests := []string{
		"",                       // 空字符串
		"invalid-token",          // 非 JWT
		"header.payload",         // 不完整
		"a.b.c",                  // 格式对但内容错
	}
	for _, tt := range tests {
		t.Run("token="+tt, func(t *testing.T) {
			_, err := ParseToken(tt)
			if err == nil {
				t.Error("ParseToken() 期望返回错误，但返回 nil")
			}
		})
	}
}

func TestSetJWTConfig_OverridesDefaults(t *testing.T) {
	// 重设密钥并生成 token，验证可解析
	SetJWTConfig("another-secret", 30*time.Minute)
	token, err := GenerateToken(1, "u@e.com")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Errorf("ParseToken() with new secret error = %v", err)
	}
	// 恢复测试密钥
	SetJWTConfig("test-secret-key", time.Hour)
}