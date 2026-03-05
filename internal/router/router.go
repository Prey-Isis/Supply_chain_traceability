package router

import (
	"errors"
	"net/http"
	"strings"
	"supply_chain/middleware"
	"supply_chain/model"
	"time"

	"github.com/gin-gonic/gin"
)

// 标准响应结构
type Response struct {
	Code    int         `json:"code"`    // 状态码：0-成功，非0-失败
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 返回数据
}

// 常量定义
const (
	SuccessCode = 0
	ErrorCode   = 1

	MaxFieldLength = 25 // 字段最大长度
)

// ==================== 认证相关处理函数 ====================

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	UserName string `json:"UserName" binding:"required,max=25"`
	Account  string `json:"Account" binding:"required,max=25"`
	Password string `json:"PassWord" binding:"required,max=25"`
	Role     string `json:"Role" binding:"required,oneof=admin manager user"` // 限定角色范围
}

// Register 用户注册
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查账号是否已存在
	existingUser, err := model.GetUserByAccount(req.Account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查用户是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, Response{
			Code:    ErrorCode,
			Message: "账号已存在",
			Data:    nil,
		})
		return
	}

	// 创建用户
	user := &model.User{
		UserName: req.UserName,
		Account:  req.Account,
		Password: req.Password, // 注意：实际应用中应该对密码进行哈希处理
		Role:     req.Role,
	}

	if err := model.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "注册失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 注册成功后不返回敏感信息
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "注册成功",
		Data: gin.H{
			"account":  user.Account,
			"userName": user.UserName,
			"role":     user.Role,
		},
	})
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Account  string `json:"Account" binding:"required,max=25"`
	Password string `json:"PassWord" binding:"required,max=25"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token     string `json:"token"`
	Account   string `json:"account"`
	UserName  string `json:"userName"`
	Role      string `json:"role"`
	ExpiresIn int64  `json:"expiresIn"` // token过期时间戳
}

// Login 用户登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 获取用户信息
	user, err := model.GetUserByAccount(req.Account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "登录失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    ErrorCode,
			Message: "账号或密码错误",
			Data:    nil,
		})
		return
	}

	// 验证密码（实际应用中应该使用哈希比较）
	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    ErrorCode,
			Message: "账号或密码错误",
			Data:    nil,
		})
		return
	}

	// 生成JWT token
	token, err := middleware.GenerateToken(user.Account, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "生成token失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 设置cookie（可选）
	c.SetCookie("token", token, int(middleware.TokenExpire.Seconds()), "/", "", false, true)

	// 返回登录信息
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "登录成功",
		Data: LoginResponse{
			Token:     token,
			Account:   user.Account,
			UserName:  user.UserName,
			Role:      user.Role,
			ExpiresIn: time.Now().Add(middleware.TokenExpire).Unix(),
		},
	})
}

// Logout 用户登出
func Logout(c *gin.Context) {
	// 清除cookie
	c.SetCookie("token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "登出成功",
		Data:    nil,
	})
}

// RefreshToken 刷新token
func RefreshToken(c *gin.Context) {
	// 从请求头获取token
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		tokenString, _ = c.Cookie("token")
	}

	if tokenString == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请提供token",
			Data:    nil,
		})
		return
	}

	// 去除Bearer前缀
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	// 刷新token
	newToken, err := middleware.RefreshToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    ErrorCode,
			Message: "token无效或已过期",
			Data:    nil,
		})
		return
	}

	// 解析token获取用户信息
	claims, _ := middleware.ParseToken(newToken)

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "token刷新成功",
		Data: gin.H{
			"token":     newToken,
			"account":   claims.Account,
			"role":      claims.Role,
			"expiresIn": claims.ExpiresAt,
		},
	})
}

// GetCurrentUser 获取当前登录用户信息
func GetCurrentUser(c *gin.Context) {
	// 从上下文中获取用户信息（由AuthRequired中间件设置）
	user, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    ErrorCode,
			Message: "未登录",
			Data:    nil,
		})
		return
	}

	currentUser := user.(*model.User)

	// 不返回密码等敏感信息
	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取用户信息成功",
		Data: gin.H{
			"account":  currentUser.Account,
			"userName": currentUser.UserName,
			"role":     currentUser.Role,
		},
	})
}

// ==================== 用户管理相关处理函数（需要权限） ====================

// CreateUserRequest 创建用户请求结构
type CreateUserRequest struct {
	UserName string `json:"UserName" binding:"required,max=25"`
	Account  string `json:"Account" binding:"required,max=25"`
	Password string `json:"PassWord" binding:"required,max=25"`
	Role     string `json:"Role" binding:"required,oneof=admin manager user"`
}

// CreateUser 创建用户（管理员功能）
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查账号是否已存在
	existingUser, err := model.GetUserByAccount(req.Account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查用户是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, Response{
			Code:    ErrorCode,
			Message: "账号已存在",
			Data:    nil,
		})
		return
	}

	// 创建用户
	user := &model.User{
		UserName: req.UserName,
		Account:  req.Account,
		Password: req.Password,
		Role:     req.Role,
	}

	if err := model.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "创建用户失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 不返回密码
	user.Password = ""

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "用户创建成功",
		Data:    user,
	})
}

// GetUser 根据账号获取用户信息（管理员功能）
func GetUser(c *gin.Context) {
	account := strings.TrimSpace(c.Param("account"))
	if account == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号不能为空",
			Data:    nil,
		})
		return
	}

	if len(account) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	user, err := model.GetUserByAccount(account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取用户失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "用户不存在",
			Data:    nil,
		})
		return
	}

	// 不返回密码
	user.Password = ""

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取用户成功",
		Data:    user,
	})
}

// GetUserByName 根据用户名获取用户（管理员功能）
func GetUserByName(c *gin.Context) {
	userName := strings.TrimSpace(c.Query("user_name"))
	if userName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "用户名不能为空",
			Data:    nil,
		})
		return
	}

	if len(userName) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "用户名长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	user, err := model.GetUserByName(userName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取用户失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "用户不存在",
			Data:    nil,
		})
		return
	}

	// 不返回密码
	user.Password = ""

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取用户成功",
		Data:    user,
	})
}

// GetAllUsers 获取所有用户（管理员功能）
func GetAllUsers(c *gin.Context) {
	users, err := model.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取用户列表失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 不返回密码
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取用户列表成功",
		Data:    users,
	})
}

// 更新用户请求结构
type UpdateUserRequest struct {
	UserName string `json:"UserName" binding:"required,max=25"`
	Password string `json:"PassWord" binding:"required,max=25"`
	Role     string `json:"Role" binding:"required,oneof=admin manager user"`
}

// UpdateUser 更新用户信息（管理员功能）
func UpdateUser(c *gin.Context) {
	account := strings.TrimSpace(c.Param("account"))
	if account == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号不能为空",
			Data:    nil,
		})
		return
	}

	if len(account) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查用户是否存在
	existingUser, err := model.GetUserByAccount(account)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查用户是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if existingUser == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "用户不存在",
			Data:    nil,
		})
		return
	}

	// 更新用户
	user := &model.User{
		UserName: req.UserName,
		Account:  account,
		Password: req.Password,
		Role:     req.Role,
	}

	if err := model.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "更新用户失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 不返回密码
	user.Password = ""

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "用户更新成功",
		Data:    user,
	})
}

// DeleteUser 删除用户（管理员功能）
func DeleteUser(c *gin.Context) {
	account := strings.TrimSpace(c.Param("account"))
	if account == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号不能为空",
			Data:    nil,
		})
		return
	}

	if len(account) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "账号长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	if err := model.DeleteUser(account); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    ErrorCode,
				Message: "用户不存在",
				Data:    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "删除用户失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "用户删除成功",
		Data:    nil,
	})
}

// ==================== 以下产品相关处理函数保持不变 ====================

// 创建产品请求结构
type CreateProductRequest struct {
	ProductId     string                 `json:"Product_Id" binding:"required,max=25"`
	Name          string                 `json:"Name" binding:"required,max=25"`
	CurrentHolder string                 `json:"Current_Holder" binding:"required,max=25"`
	Status        string                 `json:"Status" binding:"required,max=25"`
	SupplyHistory []model.Supply_History `json:"Supply_History"`
}

// validateSupplyHistory 验证供应链历史记录数据长度
func validateSupplyHistory(history *model.Supply_History) error {
	if len(history.ProductId) > MaxFieldLength {
		return errors.New("product_id长度不能超过25个字符")
	}
	if len(history.ProductName) > MaxFieldLength {
		return errors.New("product_name长度不能超过25个字符")
	}
	if len(history.NodeName) > MaxFieldLength {
		return errors.New("node_name长度不能超过25个字符")
	}
	if len(history.Location) > MaxFieldLength {
		return errors.New("location长度不能超过25个字符")
	}
	if len(history.Action) > MaxFieldLength {
		return errors.New("action长度不能超过25个字符")
	}
	if len(history.OperationRole) > MaxFieldLength {
		return errors.New("operation_role长度不能超过25个字符")
	}
	if len(history.Description) > MaxFieldLength {
		return errors.New("description长度不能超过25个字符")
	}
	return nil
}

// CreateProduct 创建产品
func CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查产品是否已存在
	existingProduct, err := model.GetProductById(req.ProductId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查产品是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if existingProduct != nil {
		c.JSON(http.StatusConflict, Response{
			Code:    ErrorCode,
			Message: "产品ID已存在",
			Data:    nil,
		})
		return
	}

	// 验证历史记录中的数据长度
	for _, history := range req.SupplyHistory {
		if err := validateSupplyHistory(&history); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    ErrorCode,
				Message: "历史记录数据验证失败：" + err.Error(),
				Data:    nil,
			})
			return
		}
	}

	// 创建产品
	product := &model.Product{
		ProductId:     req.ProductId,
		Name:          req.Name,
		CurrentHolder: req.CurrentHolder,
		Status:        req.Status,
		SupplyHistory: req.SupplyHistory,
	}

	if err := model.CreateProduct(product); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "创建产品失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "产品创建成功",
		Data:    product,
	})
}

// GetProduct 根据ID获取产品
func GetProduct(c *gin.Context) {
	productId := strings.TrimSpace(c.Param("product_id"))
	if productId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID不能为空",
			Data:    nil,
		})
		return
	}

	if len(productId) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	product, err := model.GetProductById(productId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取产品失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if product == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "产品不存在",
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取产品成功",
		Data:    product,
	})
}

// GetAllProducts 获取所有产品
func GetAllProducts(c *gin.Context) {
	products, err := model.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取产品列表失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取产品列表成功",
		Data:    products,
	})
}

// 更新产品请求结构
type UpdateProductRequest struct {
	Name          string `json:"Name" binding:"required,max=25"`
	CurrentHolder string `json:"Current_Holder" binding:"required,max=25"`
	Status        string `json:"Status" binding:"required,max=25"`
}

// UpdateProduct 更新产品信息
func UpdateProduct(c *gin.Context) {
	productId := strings.TrimSpace(c.Param("product_id"))
	if productId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID不能为空",
			Data:    nil,
		})
		return
	}

	if len(productId) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查产品是否存在
	existingProduct, err := model.GetProductById(productId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查产品是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if existingProduct == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "产品不存在",
			Data:    nil,
		})
		return
	}

	// 更新产品
	product := &model.Product{
		ProductId:     productId,
		Name:          req.Name,
		CurrentHolder: req.CurrentHolder,
		Status:        req.Status,
	}

	if err := model.UpdateProduct(product); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "更新产品失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "产品更新成功",
		Data:    product,
	})
}

// 更新产品状态请求结构
type UpdateProductStatusRequest struct {
	Status string `json:"Status" binding:"required,max=25"`
}

// UpdateProductStatus 更新产品状态
func UpdateProductStatus(c *gin.Context) {
	productId := strings.TrimSpace(c.Param("product_id"))
	if productId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID不能为空",
			Data:    nil,
		})
		return
	}

	if len(productId) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	var req UpdateProductStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if err := model.UpdateProductStatus(productId, req.Status); err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    ErrorCode,
				Message: "产品不存在",
				Data:    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "更新产品状态失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "产品状态更新成功",
		Data:    gin.H{"product_id": productId, "status": req.Status},
	})
}

// DeleteProduct 删除产品
func DeleteProduct(c *gin.Context) {
	productId := strings.TrimSpace(c.Param("product_id"))
	if productId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID不能为空",
			Data:    nil,
		})
		return
	}

	if len(productId) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	if err := model.DeleteProduct(productId); err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    ErrorCode,
				Message: "产品不存在",
				Data:    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "删除产品失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "产品删除成功",
		Data:    nil,
	})
}

// ==================== 供应链历史相关处理函数 ====================

// 创建供应链历史请求结构
type CreateSupplyHistoryRequest struct {
	ProductId     string `json:"Product_Id" binding:"required,max=25"`
	ProductName   string `json:"Product_Name" binding:"required,max=25"`
	NodeName      string `json:"Node_Name" binding:"required,max=25"`
	Location      string `json:"Location" binding:"required,max=25"`
	Action        string `json:"Action" binding:"required,max=25"`
	OperationRole string `json:"Operation_Role" binding:"required,max=25"`
	Description   string `json:"Description" binding:"max=25"`
}

// CreateSupplyHistory 创建供应链历史记录
func CreateSupplyHistory(c *gin.Context) {
	var req CreateSupplyHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 检查产品是否存在
	product, err := model.GetProductById(req.ProductId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "检查产品是否存在失败：" + err.Error(),
			Data:    nil,
		})
		return
	}
	if product == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    ErrorCode,
			Message: "产品不存在",
			Data:    nil,
		})
		return
	}

	history := &model.Supply_History{
		ProductId:     req.ProductId,
		ProductName:   req.ProductName,
		NodeName:      req.NodeName,
		Location:      req.Location,
		Action:        req.Action,
		OperationRole: req.OperationRole,
		Description:   req.Description,
	}

	if err := model.CreateSupplyHistory(history); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "创建供应链历史记录失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "供应链历史记录创建成功",
		Data:    history,
	})
}

// GetSupplyHistoryByProduct 根据产品ID获取历史记录
func GetSupplyHistoryByProduct(c *gin.Context) {
	productId := strings.TrimSpace(c.Param("product_id"))
	if productId == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID不能为空",
			Data:    nil,
		})
		return
	}

	if len(productId) > MaxFieldLength {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "产品ID长度不能超过25个字符",
			Data:    nil,
		})
		return
	}

	histories, err := model.GetSupplyHistoryByProductId(productId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取供应链历史记录失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取供应链历史记录成功",
		Data:    histories,
	})
}

// GetAllSupplyHistory 获取所有供应链历史记录
func GetAllSupplyHistory(c *gin.Context) {
	histories, err := model.GetAllSupplyHistory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "获取供应链历史记录列表失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "获取供应链历史记录列表成功",
		Data:    histories,
	})
}

// 删除供应链历史请求结构
type DeleteSupplyHistoryRequest struct {
	ProductId  string `json:"Product_Id" binding:"required,max=25"`
	CreateTime string `json:"Create_Time" binding:"required"`
}

// DeleteSupplyHistory 删除指定的历史记录
func DeleteSupplyHistory(c *gin.Context) {
	var req DeleteSupplyHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	if err := model.DeleteSupplyHistory(req.ProductId, req.CreateTime); err != nil {
		if err.Error() == "supply history not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    ErrorCode,
				Message: "供应链历史记录不存在",
				Data:    nil,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "删除供应链历史记录失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "供应链历史记录删除成功",
		Data:    nil,
	})
}

// BatchCreateSupplyHistoryRequest 批量创建供应链历史请求结构
type BatchCreateSupplyHistoryRequest struct {
	Histories []model.Supply_History `json:"histories" binding:"required,min=1"`
}

// BatchCreateSupplyHistory 批量创建供应链历史记录
func BatchCreateSupplyHistory(c *gin.Context) {
	var req BatchCreateSupplyHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    ErrorCode,
			Message: "请求参数错误：" + err.Error(),
			Data:    nil,
		})
		return
	}

	// 验证每条记录的数据长度和产品存在性
	for _, history := range req.Histories {
		if err := validateSupplyHistory(&history); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    ErrorCode,
				Message: "历史记录数据验证失败：" + err.Error(),
				Data:    nil,
			})
			return
		}

		// 检查产品是否存在
		product, err := model.GetProductById(history.ProductId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    ErrorCode,
				Message: "检查产品是否存在失败：" + err.Error(),
				Data:    nil,
			})
			return
		}
		if product == nil {
			c.JSON(http.StatusNotFound, Response{
				Code:    ErrorCode,
				Message: "产品 " + history.ProductId + " 不存在",
				Data:    nil,
			})
			return
		}
	}

	if err := model.BatchCreateSupplyHistory(req.Histories); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    ErrorCode,
			Message: "批量创建供应链历史记录失败：" + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    SuccessCode,
		Message: "批量创建供应链历史记录成功",
		Data:    req.Histories,
	})
}
