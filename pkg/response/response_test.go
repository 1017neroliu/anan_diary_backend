package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("无法解析响应体: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func TestSuccess(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { Success(c, gin.H{"foo": "bar"}) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w)
	if resp.Code != 200 {
		t.Errorf("code = %d, want 200", resp.Code)
	}
	if resp.Message != "操作成功" {
		t.Errorf("message = %q, want %q", resp.Message, "操作成功")
	}
}

func TestError(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { Error(c, http.StatusBadRequest, "参数错误") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	resp := decodeBody(t, w)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if resp.Message != "参数错误" {
		t.Errorf("message = %q, want %q", resp.Message, "参数错误")
	}
}

func TestValidationError(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { ValidationError(c, "校验失败") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ValidationError status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestNotFound(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { NotFound(c, "找不到") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("NotFound status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUnauthorized(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { Unauthorized(c, "未登录") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestForbidden(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { Forbidden(c, "禁止访问") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Forbidden status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestServerError(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { ServerError(c, "内部错误") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ServerError status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}