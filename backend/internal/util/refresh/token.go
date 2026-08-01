package refresh

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// NewToken 生成随机的 Refresh Token，客户端只保存该不透明字符串
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 Refresh Token 失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Key 返回 Redis 中存储 Refresh Token 的键，避免直接暴露原始 token
func Key(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("auth:refresh:%x", sum[:16])
}
