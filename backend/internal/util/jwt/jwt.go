package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

const accessTokenType = "access"

// Claims 是 Access Token 的声明结构
type Claims struct {
	UserID    uint   `json:"user_id"`
	TokenType string `json:"token_type"`
	jwtgo.RegisteredClaims
}

// Manager 负责 Access Token 的生成和校验
type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) GenerateAccessToken(userID uint, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		TokenType: accessTokenType,
		RegisteredClaims: jwtgo.RegisteredClaims{
			IssuedAt:  jwtgo.NewNumericDate(now),
			ExpiresAt: jwtgo.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseAccessToken 校验 Access Token 并返回用户 ID
func (m *Manager) ParseAccessToken(tokenString string) (uint, error) {
	token, err := jwtgo.ParseWithClaims(tokenString, &Claims{}, func(token *jwtgo.Token) (any, error) {
		if _, ok := token.Method.(*jwtgo.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwtgo.WithValidMethods([]string{jwtgo.SigningMethodHS256.Alg()}))
	if err != nil {
		return 0, fmt.Errorf("解析 Access Token 失败: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return 0, errors.New("Access Token 声明无效")
	}
	if claims.TokenType != accessTokenType {
		return 0, errors.New("Access Token 类型错误")
	}

	return claims.UserID, nil
}
