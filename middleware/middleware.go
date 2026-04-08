package middleware

import (
	"context"
	"log"
	"main/internal/jwt"
	"main/internal/model"
	"main/internal/router"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 定义上下文中的键名常量
const (
	ContextUserKey    = "user"
	ContextAccountKey = "account"
	ContextRoleKey    = "role"
)

// ==================== 基础中间件 ====================

// Logger 日志中间件，记录请求信息
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := c.Request.Method
		// 请求路由
		reqUri := c.Request.RequestURI
		// 状态码
		statusCode := c.Writer.Status()
		// 请求IP
		clientIP := c.ClientIP()

		// 获取请求ID
		requestID, _ := c.Get("RequestID")

		// 日志格式
		log.Printf("[GIN] %s | %3d | %13v | %15s | %s | %s | %s",
			startTime.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
			requestID,
		)
	}
}

// Recovery 恢复中间件，处理panic情况
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID, _ := c.Get("RequestID")

		if err, ok := recovered.(string); ok {
			log.Printf("[PANIC] RequestID: %s, Error: %s", requestID, err)
			c.JSON(http.StatusInternalServerError, router.Response{
				Code:    router.ErrorCode,
				Message: "服务器内部错误，请稍后重试",
				Data:    nil,
			})
		} else {
			log.Printf("[PANIC] RequestID: %s, Unknown error", requestID)
			c.JSON(http.StatusInternalServerError, router.Response{
				Code:    router.ErrorCode,
				Message: "未知服务器内部错误",
				Data:    nil,
			})
		}
		c.Abort()
	})
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Token, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-New-Token, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ==================== 认证中间件 ====================

// AuthRequired 必需的认证中间件，验证用户是否登录（使用JWT）
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			// 也可以从cookie获取
			tokenString, _ = c.Cookie("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, router.Response{
				Code:    router.ErrorCode,
				Message: "未授权访问，请先登录",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// 去除Bearer前缀
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		// 解析token
		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, router.Response{
				Code:    router.ErrorCode,
				Message: "登录已过期，请重新登录",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// 获取用户完整信息
		user, err := model.GetUserByAccount(claims.Account)
		if err != nil {
			c.JSON(http.StatusInternalServerError, router.Response{
				Code:    router.ErrorCode,
				Message: "获取用户信息失败",
				Data:    nil,
			})
			c.Abort()
			return
		}
		if user == nil {
			c.JSON(http.StatusUnauthorized, router.Response{
				Code:    router.ErrorCode,
				Message: "用户不存在或已被删除",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// 检查token是否需要刷新（即将过期）
		newToken, err := jwt.RefreshToken(tokenString)
		if err == nil && newToken != tokenString {
			// 如果token已刷新，在响应头中返回新token
			c.Writer.Header().Set("X-New-Token", newToken)
		}

		// 将用户信息存入上下文
		c.Set(ContextUserKey, user)
		c.Set(ContextAccountKey, user.Account)
		c.Set(ContextRoleKey, user.Role)

		c.Next()
	}
}

// AuthOptional 可选认证中间件（有token就验证，没有也可以继续）
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			tokenString, _ = c.Cookie("token")
		}

		if tokenString != "" {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			tokenString = strings.TrimSpace(tokenString)

			claims, err := jwt.ParseToken(tokenString)
			if err == nil {
				user, err := model.GetUserByAccount(claims.Account)
				if err == nil && user != nil {
					c.Set(ContextUserKey, user)
					c.Set(ContextAccountKey, user.Account)
					c.Set(ContextRoleKey, user.Role)
				}
			}
		}

		c.Next()
	}
}

// ==================== 权限中间件 ====================

// RoleRequired 角色权限中间件，验证用户是否具有指定角色
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取角色
		userRole, exists := c.Get(ContextRoleKey)
		if !exists {
			c.JSON(http.StatusForbidden, router.Response{
				Code:    router.ErrorCode,
				Message: "无法获取用户角色信息",
				Data:    nil,
			})
			c.Abort()
			return
		}

		// 检查角色是否在允许的列表中
		roleStr := userRole.(string)
		for _, role := range roles {
			if role == roleStr {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, router.Response{
			Code:    router.ErrorCode,
			Message: "权限不足，需要角色：" + strings.Join(roles, ", "),
			Data:    nil,
		})
		c.Abort()
	}
}

// ==================== 限流中间件 ====================

// RateLimiter 限流中间件
func RateLimiter(limit int) gin.HandlerFunc {
	// 使用令牌桶实现限流
	limiter := make(chan struct{}, limit)

	// 定时添加令牌
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			for i := 0; i < limit; i++ {
				select {
				case limiter <- struct{}{}:
				default:
				}
			}
		}
	}()

	return func(c *gin.Context) {
		select {
		case <-limiter:
			c.Next()
		default:
			c.JSON(http.StatusTooManyRequests, router.Response{
				Code:    router.ErrorCode,
				Message: "请求过于频繁，请稍后重试",
				Data:    nil,
			})
			c.Abort()
		}
	}
}

// ==================== 请求ID中间件 ====================

// RequestID 为每个请求生成唯一ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("RequestID", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

// ==================== 超时控制中间件 ====================

// TimeoutMiddleware 请求超时控制中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建一个带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 将新的上下文附加到请求
		c.Request = c.Request.WithContext(ctx)

		// 创建一个通道来处理请求完成
		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			requestID, _ := c.Get("RequestID")
			log.Printf("[TIMEOUT] RequestID: %s, Path: %s", requestID, c.Request.URL.Path)

			c.JSON(http.StatusGatewayTimeout, router.Response{
				Code:    router.ErrorCode,
				Message: "请求处理超时，请稍后重试",
				Data:    nil,
			})
			c.Abort()
		}
	}
}
