package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		allowedOrigins []string
		requestOrigin  string
		wantAllowed    bool
	}{
		{
			name:           "Allowed Origin",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "http://localhost:3000",
			wantAllowed:    true,
		},
		{
			name:           "Disallowed Origin",
			allowedOrigins: []string{"http://localhost:3000"},
			requestOrigin:  "http://evil.com",
			wantAllowed:    false,
		},
		{
			name:           "Wildcard Origin",
			allowedOrigins: []string{"*"},
			requestOrigin:  "http://anywhere.com",
			wantAllowed:    true,
		},
		{
			name:           "Multiple Allowed Origins",
			allowedOrigins: []string{"http://localhost:3000", "http://mydomain.com"},
			requestOrigin:  "http://mydomain.com",
			wantAllowed:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Server with specific CORS config
			router := gin.New()
			s := &Server{
				router:      router,
				corsOrigins: tt.allowedOrigins,
			}
			router.Use(s.corsMiddleware())
			
			// Dummy endpoint
			router.GET("/ping", func(c *gin.Context) {
				c.String(200, "pong")
			})

			req, _ := http.NewRequest("GET", "/ping", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)
			
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllowed {
				assert.Equal(t, tt.requestOrigin, allowOrigin)
			} else {
				assert.Empty(t, allowOrigin)
			}
		})
	}
}
