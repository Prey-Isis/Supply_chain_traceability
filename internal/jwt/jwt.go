package jwt

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

// 定义上下文中的键名常量
const (
	ContextUserKey    = "user"
	ContextAccountKey = "account"
	ContextRoleKey    = "role"
)

// 定义JWT常量
const (
	TokenExpire = 24 * time.Hour // token过期时间
)

// JWTSecretKey 从环境变量读取，生产环境务必设置 JWT_SECRET
var JWTSecretKey = getEnvOrDefault("JWT_SECRET", "your-secret-key")

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// CustomClaims 自定义JWT声明
type CustomClaims struct {
	Account string `json:"account"`
	Role    string `json:"role"`
	jwt.StandardClaims
}

// ==================== JWT相关函数 ====================

// GenerateToken 生成JWT token
func GenerateToken(account, role string) (string, error) {
	claims := CustomClaims{
		Account: account,
		Role:    role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpire).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "supply-chain-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecretKey))
}

// ParseToken 解析JWT token
func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken 刷新token
func RefreshToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	// 检查token是否即将过期（剩余时间小于1小时）
	if time.Until(time.Unix(claims.ExpiresAt, 0)) < time.Hour {
		return GenerateToken(claims.Account, claims.Role)
	}

	return tokenString, nil
}
