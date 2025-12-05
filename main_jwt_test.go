package main

import (
	"testing"
)

// TestCheckJWTSecret 验证 JWT 安全检查逻辑
func TestCheckJWTSecret(t *testing.T) {
	// 默认的不安全密钥（与 main.go 中保持一致）
	defaultSecret := "your-jwt-secret-key-change-in-production-make-it-long-and-random"

	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{
			name:    "Default secret should fail",
			secret:  defaultSecret,
			wantErr: true,
		},
		{
			name:    "Empty secret should fail",
			secret:  "",
			wantErr: true,
		},
		{
			name:    "Short secret should fail (less than 32 chars)",
			secret:  "too-short-secret-key",
			wantErr: true,
		},
		{
			name:    "Secure secret should pass",
			secret:  "this-is-a-very-secure-and-long-secret-key-generated-randomly",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJWTSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// 注意：validateJWTSecret 函数已在 main.go 中定义，此处直接调用