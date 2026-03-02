package model

import (
	"database/sql"
	"errors"
	"fmt"
	"supply_chain/config"
	"supply_chain/pkg/utils"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	user_name   string `json:"UserName"`
	account     string `json:"Account"`
	password    string `json:"PassWord"`
	role        string `json:"Role"`
	create_time string `json:"Create_Time"`
	update_time string `json:"Update_Time"`
}

type Supply_History struct {
	product_id     string `json:"Product_Id"`
	product_name   string `json:"Product_Name"`
	node_name      string `json:"Node_Name"`
	location       string `json:"Location"`
	action         string `json:"Action"`
	operation_role string `json:"Operation_Role"`
	description    string `json:"Description"`
	create_time    string `json:"Create_Time"`
}

type Product struct {
	product_id     string           `json:"Product_Id"`
	name           string           `json:"Name"`
	current_holder string           `json:"Current_Holder"`
	status         string           `json:"Status"`
	create_time    string           `json:"Create_Time"`
	update_time    string           `json:"Update_Time"`
	supply_history []Supply_History `json:"Supply_History"`
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

	query := "INSERT INTO user (user_name, account, password, role) VALUES (?, ?, ?, ?)"

	_, err := db.Exec(query,
		user.user_name,
		user.account,
		user.password,
		user.role,
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
	query := "SELECT user_name, account, password, role, create_time, update_time FROM user WHERE account = ?"

	err := db.QueryRow(query, account).Scan(
		&user.user_name,
		&user.account,
		&user.password,
		&user.role,
		&user.create_time,
		&user.update_time,
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
	query := "SELECT user_name, account, password, role, create_time, update_time FROM user WHERE user_name = ?"

	err := db.QueryRow(query, userName).Scan(
		&user.user_name,
		&user.account,
		&user.password,
		&user.role,
		&user.create_time,
		&user.update_time,
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

	query := "SELECT user_name, account, password, role, create_time, update_time FROM user ORDER BY create_time DESC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.user_name,
			&user.account,
			&user.password,
			&user.role,
			&user.create_time,
			&user.update_time,
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

	query := "UPDATE user SET user_name = ?, password = ?, role = ? WHERE account = ?"

	result, err := db.Exec(query,
		user.user_name,
		user.password,
		user.role,
		user.account,
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
	// 如果product_id不存在，数据库会返回外键约束错误

	query := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"

	_, err := db.Exec(query,
		history.product_id,
		history.product_name,
		history.node_name,
		history.location,
		history.action,
		history.operation_role,
		history.description,
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

	query := "SELECT product_id, product_name, node_name, location, action, operation_role, description, create_time FROM supply_history WHERE product_id = ? ORDER BY create_time DESC"

	rows, err := db.Query(query, productId)
	if err != nil {
		return nil, fmt.Errorf("failed to get supply history: %w", err)
	}
	defer rows.Close()

	var histories []Supply_History
	for rows.Next() {
		var history Supply_History
		err := rows.Scan(
			&history.product_id,
			&history.product_name,
			&history.node_name,
			&history.location,
			&history.action,
			&history.operation_role,
			&history.description,
			&history.create_time,
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

	query := "SELECT product_id, product_name, node_name, location, action, operation_role, description, create_time FROM supply_history ORDER BY create_time DESC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all supply history: %w", err)
	}
	defer rows.Close()

	var histories []Supply_History
	for rows.Next() {
		var history Supply_History
		err := rows.Scan(
			&history.product_id,
			&history.product_name,
			&history.node_name,
			&history.location,
			&history.action,
			&history.operation_role,
			&history.description,
			&history.create_time,
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

// DeleteSupplyHistory 删除指定的历史记录
func DeleteSupplyHistory(productId string, createTime string) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "DELETE FROM supply_history WHERE product_id = ? AND create_time = ?"
	result, err := db.Exec(query, productId, createTime)
	if err != nil {
		return fmt.Errorf("failed to delete supply history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("supply history not found")
	}

	return nil
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
		product.product_id,
		product.name,
		product.current_holder,
		product.status,
	)

	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	// 如果有历史记录，一并创建
	if len(product.supply_history) > 0 {
		for i := range product.supply_history {
			product.supply_history[i].product_id = product.product_id

			historyQuery := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"

			_, err = tx.Exec(historyQuery,
				product.supply_history[i].product_id,
				product.supply_history[i].product_name,
				product.supply_history[i].node_name,
				product.supply_history[i].location,
				product.supply_history[i].action,
				product.supply_history[i].operation_role,
				product.supply_history[i].description,
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
		product_id: productId,
	}

	// 获取产品基本信息
	query := "SELECT name, current_holder, status, create_time, update_time FROM product WHERE product_id = ?"

	err := db.QueryRow(query, productId).Scan(
		&product.name,
		&product.current_holder,
		&product.status,
		&product.create_time,
		&product.update_time,
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
	product.supply_history = histories

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
			&product.product_id,
			&product.name,
			&product.current_holder,
			&product.status,
			&product.create_time,
			&product.update_time,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}

		// 获取每个产品的历史记录
		histories, err := GetSupplyHistoryByProductId(product.product_id)
		if err != nil {
			return nil, fmt.Errorf("failed to get history for product %s: %w", product.product_id, err)
		}
		product.supply_history = histories

		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating product rows: %w", err)
	}

	return products, nil
}

// UpdateProduct 更新产品信息（update_time由数据库自动更新）
func UpdateProduct(product *Product) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	query := "UPDATE product SET name = ?, current_holder = ?, status = ? WHERE product_id = ?"

	result, err := db.Exec(query,
		product.name,
		product.current_holder,
		product.status,
		product.product_id,
	)

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

// BatchCreateSupplyHistory 批量创建供应链历史记录
func BatchCreateSupplyHistory(histories []Supply_History) error {
	if db == nil {
		return errors.New("database connection not initialized")
	}

	if len(histories) == 0 {
		return nil
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

	// 准备批量插入语句
	query := "INSERT INTO supply_history (product_id, product_name, node_name, location, action, operation_role, description) VALUES (?, ?, ?, ?, ?, ?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// 批量插入
	for _, history := range histories {
		_, err = stmt.Exec(
			history.product_id,
			history.product_name,
			history.node_name,
			history.location,
			history.action,
			history.operation_role,
			history.description,
		)
		if err != nil {
			return fmt.Errorf("failed to batch create supply history: %w", err)
		}
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
