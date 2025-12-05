package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestStaticFileSecurity_RootLeakage 确保根目录下的敏感文件不会被静态文件服务泄露
func TestStaticFileSecurity_RootLeakage(t *testing.T) {
	// 1. 准备环境：模拟根目录下有敏感文件
	sensitiveFiles := []string{"config.json", ".env", "main.go", "config.db"}
	for _, file := range sensitiveFiles {
		err := os.WriteFile(file, []byte("SECRET_CONTENT"), 0600)
		if err != nil {
			t.Fatalf("Failed to create mock sensitive file %s: %v", file, err)
		}
		defer os.Remove(file) // 清理
	}

	// 2. 启动 Server（使用嵌入的前端文件）
	gin.SetMode(gin.TestMode)
	s := &Server{
		router: gin.New(),
	}
	s.serveFrontend() // 开启静态服务（使用嵌入的文件系统）

	// 3. 执行测试：尝试访问敏感文件
	for _, file := range sensitiveFiles {
		req, _ := http.NewRequest("GET", "/"+file, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// 断言：
		// 1. 绝对不能返回文件内容 "SECRET_CONTENT"
		// 2. 期望行为：SPA 模式返回 200 OK，内容是嵌入的 index.html

		body := w.Body.String()

		// 检查内容是否泄露
		if body == "SECRET_CONTENT" {
			t.Errorf("🚨 SECURITY FAILURE: Sensitive file %s was leaked via HTTP!", file)
		}

		// 验证返回 200 且内容是 HTML（嵌入的 index.html）
		assert.Equal(t, http.StatusOK, w.Code, "Should return 200 (SPA fallback)")
		assert.Contains(t, body, "<!doctype html>", "Should return index.html, not file content")
		assert.Contains(t, body, "<div id=\"root\">", "Should contain React root element")
		assert.NotContains(t, body, "SECRET_CONTENT", "Should NOT contain sensitive file content")
	}
}

// TestStaticFileRouting_Assets 确保静态资源路由正确工作
func TestStaticFileRouting_Assets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{
		router: gin.New(),
	}
	s.serveFrontend()

	// 测试 /assets 路由
	t.Run("Assets JS file", func(t *testing.T) {
		// 请求任意 .js 文件（实际文件名会变化，但路由应该工作）
		req, _ := http.NewRequest("GET", "/assets/", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		// 目录列表或 404 都可以接受，关键是不是 500
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	// 测试 /icons 路由
	t.Run("Icons SVG file", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/icons/nofx.svg", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Should return 200 for icon file")
		assert.Contains(t, w.Body.String(), "<svg", "Should return SVG content")
	})

	// 测试 /images 路由
	t.Run("Images file", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/images/", nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		// 目录列表或 404 都可以接受，关键是不是 500
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})
}
