package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"supply_chain/middleware"
	"supply_chain/router"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 设置运行模式
	gin.SetMode(gin.DebugMode) // 生产模式 gin.ReleaseMode，开发时可改为 gin.DebugMode

	// 创建 Gin 引擎
	r := gin.New()

	// ==================== 全局中间件 ====================
	// 使用自定义的 Recovery 中间件（替代 gin.Default 中的 Recovery）
	r.Use(middleware.Recovery())

	// 使用自定义的 Logger 中间件
	r.Use(middleware.Logger())

	// 请求ID中间件
	r.Use(middleware.RequestID())

	// CORS 跨域中间件
	r.Use(middleware.CORS())

	// 超时控制中间件（30秒超时）
	r.Use(middleware.TimeoutMiddleware(30 * time.Second))

	// 限流中间件（每秒最多100个请求）
	r.Use(middleware.RateLimiter(100))

	// ==================== 健康检查 ====================
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, router.Response{
			Code:    router.SuccessCode,
			Message: "服务正常运行",
			Data: gin.H{
				"time":    time.Now().Format("2006-01-02 15:04:05"),
				"status":  "ok",
				"version": "1.0.0",
			},
		})
	})

	// ==================== API 路由分组 ====================

	// 公开 API（无需认证）
	public := r.Group("/api/v1")
	{
		// 认证相关
		// public.POST("/register", router.Register)    // 不开放用户注册，仅管理员可创建账户
		public.POST("/login", router.Login)
		public.POST("/refresh-token", router.RefreshToken)

		// 公开的产品查询（使用可选认证，有token就获取用户信息，没有也可以）
		public.GET("/products", middleware.AuthOptional(), router.GetAllProducts)
		public.GET("/products/:product_id", middleware.AuthOptional(), router.GetProduct)
		public.GET("/products/:product_id/history", middleware.AuthOptional(), router.GetSupplyHistoryByProduct)
		public.GET("/supply-history", middleware.AuthOptional(), router.GetAllSupplyHistory)
	}

	// 需要认证的 API
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthRequired())
	{
		// 用户相关（需要登录）
		auth.GET("/user/current", router.GetCurrentUser)
		auth.POST("/logout", router.Logout)

		// 产品管理（需要登录）
		auth.POST("/products", router.CreateProduct)
		auth.PUT("/products/:product_id", router.UpdateProduct)
		auth.PATCH("/products/:product_id/status", router.UpdateProductStatus)
		auth.DELETE("/products/:product_id", router.DeleteProduct)

		// 供应链历史管理（需要登录）
		auth.POST("/supply-history", router.CreateSupplyHistory)
		auth.POST("/supply-history/batch", router.BatchCreateSupplyHistory)
		auth.DELETE("/supply-history", router.DeleteSupplyHistory)
	}

	// 管理员 API（需要认证 + 管理员角色）
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthRequired())
	admin.Use(middleware.RoleRequired("admin"))
	{
		// 用户管理（仅管理员）
		admin.POST("/users", router.CreateUser)
		admin.GET("/users", router.GetAllUsers)
		admin.GET("/users/by-name", router.GetUserByName)
		admin.GET("/users/:account", router.GetUser)
		admin.PUT("/users/:account", router.UpdateUser)
		admin.DELETE("/users/:account", router.DeleteUser)
	}

	// 经理及以上权限 API（需要认证 + 经理或管理员角色）
	manager := r.Group("/api/v1/manager")
	manager.Use(middleware.AuthRequired())
	manager.Use(middleware.RoleRequired("admin", "manager"))
	{
		// 这里可以添加经理级别的特殊权限路由
		// 例如：批量操作、报表查看等
		manager.GET("/reports/products", func(c *gin.Context) {
			c.JSON(http.StatusOK, router.Response{
				Code:    router.SuccessCode,
				Message: "产品报表（仅经理和管理员可见）",
				Data:    nil,
			})
		})
	}

	// ==================== 404 处理 ====================
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, router.Response{
			Code:    router.ErrorCode,
			Message: "请求的接口不存在",
			Data:    nil,
		})
	})

	// ==================== 405 处理 ====================
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, router.Response{
			Code:    router.ErrorCode,
			Message: "请求方法不允许",
			Data:    nil,
		})
	})

	// ==================== 启动服务器 ====================
	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Println("服务器启动在 http://localhost:8080")
		log.Println("API文档: http://localhost:8080/api/v1")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// ==================== 优雅关闭 ====================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	log.Println("服务器已关闭")
}

// 初始化函数，在 main 之前执行
func init() {
	// 这里可以添加初始化操作，例如：
	// - 加载配置文件
	// - 连接数据库
	// - 初始化日志
	// - 创建必要的目录等

	log.Println("初始化供应链管理系统...")

	// 检查必要的配置
	checkConfig()
}

// 检查配置
func checkConfig() {
	// 检查 JWT 密钥是否设置（实际应用中应该从配置文件读取）
	if middleware.JWTSecretKey == "your-secret-key" {
		log.Println("警告: 使用默认 JWT 密钥，请在生产环境中修改")
	}

	// 这里可以添加其他配置检查
	log.Println("配置检查完成")
}
