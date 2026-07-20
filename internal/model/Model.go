package model

import (
	"database/sql"
	"errors"
	"fmt"
	"main/config"
	"main/pkg/utils"
	"strings"
	"sync"  // ★ Goroutine 同步原语：WaitGroup 用于等待一组 goroutine 全部完成
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	UserName    string `json:"UserName"`
	Account     string `json:"Account"`
	PassWord    string `json:"PassWord"`
	Role        string `json:"Role"`
	Create_Time string `json:"Create_Time"`
	Update_Time string `json:"Update_Time"`
}

type Supply_History struct {
	Product_Id     string `json:"Product_Id"`
	Product_Name   string `json:"Product_Name"`
	Node_Name      string `json:"Node_Name"`
	Location       string `json:"Location"`
	Action         string `json:"Action"`
	Operation_Role string `json:"Operation_Role"`
	Description    string `json:"Description"`
	Create_Time    string `json:"Create_Time"`
}

type Product struct {
	Product_Id     string           `json:"Product_Id"`
	Name           string           `json:"Name"`
	Current_Holder string           `json:"Current_Holder"`
	Status         string           `json:"Status"`
	Create_Time    string           `json:"Create_Time"`
	Update_Time    string           `json:"Update_Time"`
	Supply_History []Supply_History `json:"Supply_History"`
}

var cfg *config.Config = utils.InitConfig("")
var db *sql.DB

// InitDB 初始化数据库连接，使用配置文件中的连接池配置
func InitDB() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	var err error
	db, err = sql.Open(cfg.Database.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// 设置连接池参数
	if cfg.Database.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(10) // 默认值
	}

	if cfg.Database.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(100) // 默认值
	}

	if cfg.Database.MaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.Database.MaxLifetime)
	} else {
		db.SetConnMaxLifetime(time.Hour) // 默认值
	}

	// 测试数据库连接
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// GetDB 获取数据库连接实例（供事务使用）
func GetDB() *sql.DB {
	return db
}

// ==================== User 表操作 ====================

// CreateUser 创建用户（时间字段由数据库自动设置）
func CreateUser(user *User) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "INSERT INTO user (username, account, password, role) VALUES (?, ?, ?, ?)"

	_, err := db.Exec(query,
		user.UserName,
		user.Account,
		user.PassWord,
		user.Role,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByAccount 根据账号获取用户
func GetUserByAccount(account string) (*User, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	user := &User{}
	query := "SELECT username, account, password, role, create_time, update_time FROM user WHERE account = ?"

	err := db.QueryRow(query, account).Scan(
		&user.UserName,
		&user.Account,
		&user.PassWord,
		&user.Role,
		&user.Create_Time,
		&user.Update_Time,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user by account: %w", err)
	}

	return user, nil
}

// GetUserByName 根据用户名获取用户
func GetUserByName(userName string) (*User, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	user := &User{}
	query := "SELECT username, account, password, role, create_time, update_time FROM user WHERE username = ?"

	err := db.QueryRow(query, userName).Scan(
		&user.UserName,
		&user.Account,
		&user.PassWord,
		&user.Role,
		&user.Create_Time,
		&user.Update_Time,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user by name: %w", err)
	}

	return user, nil
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]User, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := "SELECT username, account, password, role, create_time, update_time FROM user ORDER BY create_time DESC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.UserName,
			&user.Account,
			&user.PassWord,
			&user.Role,
			&user.Create_Time,
			&user.Update_Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// UpdateUser 更新用户信息（update_time由数据库自动更新）
func UpdateUser(user *User) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "UPDATE user SET username = ?, password = ?, role = ? WHERE account = ?"

	result, err := db.Exec(query,
		user.UserName,
		user.PassWord,
		user.Role,
		user.Account,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// DeleteUser 删除用户
func DeleteUser(account string) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "DELETE FROM user WHERE account = ?"
	result, err := db.Exec(query, account)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ==================== Supply_History 表操作 ====================

// CreateSupplyHistory 创建供应链历史记录（create_time由数据库自动设置）
func CreateSupplyHistory(history *Supply_History) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	// 由于数据库有外键约束，不需要手动检查产品是否存在
	// 如果Product_Id不存在，数据库会返回外键约束错误

	query := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"

	_, err := db.Exec(query,
		history.Product_Id,
		history.Product_Name,
		history.Node_Name,
		history.Location,
		history.Action,
		history.Operation_Role,
		history.Description,
	)

	if err != nil {
		return fmt.Errorf("failed to create supply history: %w", err)
	}

	return nil
}

// GetSupplyHistoryByProductId 根据产品ID获取历史记录
func GetSupplyHistoryByProductId(productId string) ([]Supply_History, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := "SELECT product_id, product_name, node_name, location, action, operation_role, description, create_time FROM supply_history WHERE product_id = ? ORDER BY id ASC"

	rows, err := db.Query(query, productId)
	if err != nil {
		return nil, fmt.Errorf("failed to get supply history: %w", err)
	}
	defer rows.Close()

	var histories []Supply_History
	for rows.Next() {
		var history Supply_History
		err := rows.Scan(
			&history.Product_Id,
			&history.Product_Name,
			&history.Node_Name,
			&history.Location,
			&history.Action,
			&history.Operation_Role,
			&history.Description,
			&history.Create_Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		histories = append(histories, history)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return histories, nil
}

// GetAllSupplyHistory 获取所有供应链历史记录
func GetAllSupplyHistory() ([]Supply_History, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := "SELECT product_id, product_name, node_name, location, action, operation_role, description, create_time FROM supply_history ORDER BY id ASC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all supply history: %w", err)
	}
	defer rows.Close()

	var histories []Supply_History
	for rows.Next() {
		var history Supply_History
		err := rows.Scan(
			&history.Product_Id,
			&history.Product_Name,
			&history.Node_Name,
			&history.Location,
			&history.Action,
			&history.Operation_Role,
			&history.Description,
			&history.Create_Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		histories = append(histories, history)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return histories, nil
}

// ==================== Product 表操作 ====================

// CreateProduct 创建产品（使用事务确保数据一致性，时间字段由数据库自动设置）
func CreateProduct(product *Product) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 插入产品（不包含时间字段，由数据库自动设置）
	query := "INSERT INTO product (product_id, name, current_holder, status) VALUES (?, ?, ?, ?)"

	_, err = tx.Exec(query,
		product.Product_Id,
		product.Name,
		product.Current_Holder,
		product.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// 如果有历史记录，一并创建
	if len(product.Supply_History) > 0 {
		for i := range product.Supply_History {
			product.Supply_History[i].Product_Id = product.Product_Id

			historyQuery := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"

			_, err = tx.Exec(historyQuery,
				product.Supply_History[i].Product_Id,
				product.Supply_History[i].Product_Name,
				product.Supply_History[i].Node_Name,
				product.Supply_History[i].Location,
				product.Supply_History[i].Action,
				product.Supply_History[i].Operation_Role,
				product.Supply_History[i].Description,
			)

			if err != nil {
				return fmt.Errorf("failed to create supply history: %w", err)
			}
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetProductById 根据产品ID获取产品
func GetProductById(productId string) (*Product, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	product := &Product{
		Product_Id: productId,
	}

	// 获取产品基本信息
	query := "SELECT name, current_holder, status, create_time, update_time FROM product WHERE product_id = ?"

	err := db.QueryRow(query, productId).Scan(
		&product.Name,
		&product.Current_Holder,
		&product.Status,
		&product.Create_Time,
		&product.Update_Time,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get product by id: %w", err)
	}

	// 获取产品的历史记录
	histories, err := GetSupplyHistoryByProductId(productId)
	if err != nil {
		return nil, fmt.Errorf("failed to get product history: %w", err)
	}
	product.Supply_History = histories

	return product, nil
}

// GetAllProducts 获取所有产品
func GetAllProducts() ([]Product, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := "SELECT product_id, name, current_holder, status, create_time, update_time FROM product ORDER BY create_time DESC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.Product_Id,
			&product.Name,
			&product.Current_Holder,
			&product.Status,
			&product.Create_Time,
			&product.Update_Time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}

		// 获取每个产品的历史记录
		histories, err := GetSupplyHistoryByProductId(product.Product_Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get history for product %s: %w", product.Product_Id, err)
		}
		product.Supply_History = histories

		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// ============================================================
// ★ 并发优化版：GetAllProductsConcurrent
// ============================================================
// 【为什么需要这个版本？】
// 原版 GetAllProducts 是串行的 N+1 模式：
//   查询所有产品（1次SQL）→ 遍历每个产品，逐个查历史（N次SQL）
//   总耗时 = 1次查询耗时 + N×每次查询耗时
//   假如有 50 个产品，每个历史查询 30ms，总耗时 = 50×30 = 1500ms
//
// 【Goroutine + Channel 方案怎么解决？】
// 核心思路：把 N 次查历史的操作"并发"做，而不是"一个接一个"做
//   总耗时 ≈ 1次查询耗时 + 30ms（所有历史查询同时发起，只等最慢的那个）
//   100个产品也能在大约 50ms 内完成，而不是 3000ms！
//
// 【涉及的核心概念】
//   1. goroutine  — go 关键字开一个"轻量级线程"，语法: go 函数名()
//   2. Channel    — goroutine 之间传递数据的"管道"，语法: make(chan 类型, 缓冲区大小)
//   3. WaitGroup  — 计数器，用来"等所有 goroutine 跑完"，wg.Add/Wg.Done/wg.Wait
//   4. 闭包变量陷阱 — 循环里启动 goroutine 时，要"捕获"当前循环变量的值，否则出错
// ============================================================

// productResult 每个 goroutine 的返回值"包裹"结构体
// 为什么要定义这个？因为 Channel 只能传一种类型，我们需要同时传回 product_id、历史数据和错误
type productResult struct {
	ProductID string            // 产品ID，用来把结果"贴回"对应的产品上
	Histories []Supply_History  // 该产品的供应链历史记录
	Err       error             // 如果查询出错，把错误放这里（不中断其他 goroutine）
}

func GetAllProductsConcurrent() ([]Product, error) {
	// ===== 第1步：查询所有产品（这一步没法并发，必须等结果）=====
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := "SELECT product_id, name, current_holder, status, create_time, update_time FROM product ORDER BY create_time DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}
	defer rows.Close()

	// 先扫出所有产品，暂不查历史
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.Product_Id, &p.Name, &p.Current_Holder, &p.Status, &p.Create_Time, &p.Update_Time); err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	// 没有产品直接返回
	if len(products) == 0 {
		return products, nil
	}

	// ===== 第2步：创建 Channel 和 WaitGroup =====

	// Channel：goroutine 之间的"快递管道"
	//   make(chan productResult, len(products))
	//   缓冲大小 = 产品数量，这样即使主 goroutine 还没"收货"，子 goroutine 也能把结果丢进去而不阻塞
	resultChan := make(chan productResult, len(products))

	// WaitGroup：一个"计数器"，用来等所有 goroutine 跑完
	//   想象成：你派了 N 个工人去干活，每个工人出发前在计数器上 +1（Add），
	//   干完活后 -1（Done），你只需要 Wait() 等到计数器归零就行
	var wg sync.WaitGroup

	// ===== 第3步：为每个产品启动一个 goroutine，并发查询历史 =====
	for i := range products {
		wg.Add(1) // 计数器 +1："我派了一个工人出去"

		// ★★★ 重要：闭包变量捕获 ★★★
		//   不能直接写 `go func() { ... products[i].Product_Id ... }()`
		//   因为 goroutine 启动有延迟，等它真正执行时，for 循环可能已经跑完了，
		//   此时 i 的值已经变成了最后一个索引，所有 goroutine 读到的都是同一个产品！
		//
		//   解决方法：把循环变量"复制一份"传给 goroutine
		//   方法1（推荐）：for 循环里用 `i := i` 或 `productID := products[i].Product_Id`
		//   方法2：go func(pid string) { ... }(products[i].Product_Id) 作为参数传入
		productID := products[i].Product_Id // ★ 关键：把值"捕获"到局部变量，切断和循环变量的联系

		go func(pid string) {
			// defer wg.Done() 相当于"不管函数正常结束还是 panic，最后一定执行 Done()"
			// 这保证了计数器一定会 -1，不会让 Wait() 永远等下去
			defer wg.Done()

			// 并发执行：调用已有的历史查询函数（它内部是独立的 DB 连接操作，线程安全）
			histories, queryErr := GetSupplyHistoryByProductId(pid)

			// 把结果（成功或失败）塞进 Channel
			// 注意：这里不发 panic，而是把 error 作为结果字段传回去
			//       这样一个产品的历史查失败了，不影响其他产品
			resultChan <- productResult{
				ProductID: pid,
				Histories: histories,
				Err:       queryErr,
			}
		}(productID) // ★ 把捕获好的值当作参数传进去
	}

	// ===== 第4步：等所有 goroutine 跑完，然后关闭 Channel =====
	// 为什么要再开一个 goroutine 来关 Channel？
	//   如果直接在主线等 wg.Wait() 再 close(resultChan)，那主线就卡住了，
	//   下面的 for range resultChan 就永远执行不到（死锁）。
	//   所以把"等 + 关"放到一个独立的 goroutine 里去：
	go func() {
		wg.Wait()         // 阻塞，直到计数器归零（所有子 goroutine 都 Done了）
		close(resultChan) // 关闭 Channel，通知 for range 循环"没有更多数据了，可以退出了"
	}()

	// ===== 第5步：从 Channel 收取结果，组装到 products 里 =====
	//   用 map 建立 product_id → 在 products 切片中的索引 的映射
	//   这样才能把异步返回的历史记录"贴回"对应的产品上
	indexMap := make(map[string]int, len(products))
	for idx, p := range products {
		indexMap[p.Product_Id] = idx
	}

	// for range 一个 Channel：每来一个结果就处理一个，Channel 关闭后自动退出循环
	var firstErr error // 记录第一个遇到的错误（只记录，不中断流程，尽量多返回数据）
	for result := range resultChan {
		if result.Err != nil {
			// 不要中断！记录错误但继续处理其他结果
			// 这样即使某个产品的历史查失败了，其他产品仍然正常返回
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to get history for product %s: %w", result.ProductID, result.Err)
			}
			continue
		}

		// 把历史记录贴到对应产品上
		if idx, ok := indexMap[result.ProductID]; ok {
			products[idx].Supply_History = result.Histories
		}
	}

	// 如果有错误，返回第一个错误（但仍然返回已成功的产品数据）
	if firstErr != nil {
		return products, firstErr
	}

	return products, nil
}

// ============================================================
// ★ 并发优化版：GetProductByIdConcurrent
// ============================================================
// 【原版 GetProductById 的问题】
//   第1步：查产品基本信息（1次 DB 查询）
//   第2步：等第1步完成 → 查供应链历史（1次 DB 查询）
//   总耗时 = 产品查询耗时 + 历史查询耗时（串行累加）
//
// 【并发版的优化】
//   第1步：同时发起产品查询和时间查询（不互相等待）
//   第2步：两个结果都回来 → 组装返回
//   总耗时 = max(产品查询耗时, 历史查询耗时)（只等最慢的那个）
//
// 【和前一个例子（GetAllProductsConcurrent）的区别】
//   GetAllProducts 是"动态数量"的 goroutine（产品有几个就开几个）
//   这个是"固定数量"的 goroutine（永远只开 2 个）
//   两种模式都很常用，覆盖了大部分并发场景
//
// 【这个例子额外演示的概念】
//   - 如何用 2 个 goroutine 并发执行两类完全不同的查询
//   - 如何处理"一条查询返回了空结果"的情况（产品不存在时，丢弃历史查询结果）
// ============================================================

func GetProductByIdConcurrent(productId string) (*Product, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// ===== 定义两个 goroutine 返回的结果结构体 =====
	// 和 GetAllProductsConcurrent 不同，这里只有 2 种结果：
	//   1. 产品信息（可能为 nil，表示产品不存在）
	//   2. 供应链历史（可能为空切片）
	// 所以不需要通用的 productResult，直接用两个独立的结果变量 + Channel 区分即可

	// Channel 传通用的结果：用同一个结构体，靠 type 字段区分是哪种结果
	type queryResult struct {
		kind     string            // "product" 还是 "history"
		product  *Product          // 产品数据（只有 kind=="product" 时有值）
		histories []Supply_History // 历史数据（只有 kind=="history" 时有值）
		err      error
	}

	resultChan := make(chan queryResult, 2) // 缓冲=2，刚好容下两个 goroutine 的结果
	var wg sync.WaitGroup

	// ===== goroutine 1：查产品基本信息 =====
	wg.Add(1)
	go func() {
		defer wg.Done()

		query := "SELECT name, current_holder, status, create_time, update_time FROM product WHERE product_id = ?"

		var p Product
		p.Product_Id = productId

		err := db.QueryRow(query, productId).Scan(
			&p.Name, &p.Current_Holder, &p.Status,
			&p.Create_Time, &p.Update_Time,
		)

		// ErrNoRows 不是"错误"，只是"查不到"，需要特殊处理
		if err == sql.ErrNoRows {
			// 产品不存在 → 传 nil 表示空，err 传 nil 表示"查询本身没出错"
			resultChan <- queryResult{kind: "product", product: nil, err: nil}
			return
		}

		if err != nil {
			resultChan <- queryResult{kind: "product", err: err}
			return
		}

		resultChan <- queryResult{kind: "product", product: &p, err: nil}
	}()

	// ===== goroutine 2：查供应链历史 =====
	wg.Add(1)
	go func() {
		defer wg.Done()

		// 复用已有的查询函数（线程安全，内部各自使用独立的 DB 连接）
		histories, err := GetSupplyHistoryByProductId(productId)
		resultChan <- queryResult{kind: "history", histories: histories, err: err}
	}()

	// ===== 关闭 Channel 的 goroutine =====
	// 和 GetAllProductsConcurrent 一样的模式：等两个 goroutine 都 Done() 了再 close
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// ===== 从 Channel 收集结果 =====
	var productResult *Product // 产品查询结果（nil = 不存在）
	var allHistories []Supply_History
	var firstErr error

	for result := range resultChan {
		if result.err != nil {
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}

		switch result.kind {
		case "product":
			productResult = result.product
		case "history":
			allHistories = result.histories
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	// 产品不存在
	if productResult == nil {
		return nil, nil
	}

	// 组装最终结果：把历史记录贴到产品上
	productResult.Supply_History = allHistories
	return productResult, nil
}

// UpdateProduct 更新产品信息（update_time由数据库自动更新）
func UpdateProduct(product *Product) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "UPDATE product SET name = ?, current_holder = ?, status = ? WHERE product_id = ?"

	result, err := db.Exec(query,
		product.Name,
		product.Current_Holder,
		product.Status,
		product.Product_Id,
	)
	fmt.Println(product)

	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("product not found")
	}

	return nil
}

// UpdateProductStatus 更新产品状态（update_time由数据库自动更新）
func UpdateProductStatus(productId string, status string) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "UPDATE product SET status = ? WHERE product_id = ?"

	result, err := db.Exec(query, status, productId)
	if err != nil {
		return fmt.Errorf("failed to update product status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("product not found")
	}

	return nil
}

// DeleteProduct 删除产品（由于设置了CASCADE，关联的supply_history会自动删除）
func DeleteProduct(productId string) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	// 由于外键设置了ON DELETE CASCADE，只需要删除产品即可
	// 关联的supply_history记录会被数据库自动删除
	query := "DELETE FROM product WHERE product_id = ?"

	result, err := db.Exec(query, productId)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("product not found")
	}

	return nil
}

// BatchCreateSupplyHistory 智能批量创建供应链历史记录
func BatchCreateSupplyHistory(histories []Supply_History) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	if len(histories) == 0 {
		return nil
	}

	// 根据数据量选择最佳策略
	if len(histories) < 50 {
		// 小数据量：使用事务+循环（你原来的方法）
		return batchCreateWithTransaction(histories)
	} else {
		// 大数据量：使用单条SQL批量插入
		return batchCreateWithSingleSQL(histories)
	}
}

// 小数据量使用事务方式
func batchCreateWithTransaction(histories []Supply_History) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var txErr error
	defer func() {
		if txErr != nil {
			tx.Rollback()
		}
	}()

	query := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		txErr = err
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, history := range histories {
		_, err = stmt.Exec(
			history.Product_Id,
			history.Product_Name,
			history.Node_Name,
			history.Location,
			history.Action,
			history.Operation_Role,
			history.Description,
		)
		if err != nil {
			txErr = err
			return fmt.Errorf("failed to create supply history: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		txErr = err
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// 大数据量使用单条SQL
func batchCreateWithSingleSQL(histories []Supply_History) error {
	valueStrings := make([]string, 0, len(histories))
	valueArgs := make([]interface{}, 0, len(histories)*7)

	for _, history := range histories {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			history.Product_Id,
			history.Product_Name,
			history.Node_Name,
			history.Location,
			history.Action,
			history.Operation_Role,
			history.Description,
		)
	}

	query := fmt.Sprintf("INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES %s",
		strings.Join(valueStrings, ","))

	_, err := db.Exec(query, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to batch create supply history: %w", err)
	}

	return nil
}
