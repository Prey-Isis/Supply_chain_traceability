package jwt

import (
	"errors"

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
	JWTSecretKey = "your-secret-key" // 实际使用时应该从配置文件中读取
	TokenExpire  = 24 * time.Hour    // token过期时间
)

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
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) < time.Hour {
		return GenerateToken(claims.Account, claims.Role)
	}

	return tokenString, nil
}
