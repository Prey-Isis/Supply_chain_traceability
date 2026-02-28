// config/config.go
package config

import (
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`

	// 连接池配置（可选，可根据需要添加）
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxLifetime  time.Duration `mapstructure:"max_lifetime"`
}

// Config 总配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

// 全局配置实例
var GlobalConfig *Config

// InitConfig 初始化配置
func InitConfig(configPath string) error {
	v := viper.New()

	// 设置配置文件
	if configPath != "" {
		v.SetConfigFile(configPath) // 使用指定的配置文件
	} else {
		// 默认配置
		v.SetConfigName("config")   // 配置文件名称(无扩展名)
		v.SetConfigType("yaml")     // 如果配置文件的名称没有扩展名，则需要设置类型
		v.AddConfigPath(".")        // 查找配置文件的路径
		v.AddConfigPath("./config") // 也可以查找 config 目录
	}

	// 读取环境变量（可选，支持通过环境变量覆盖配置）
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 将配置解析到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置默认值（如果配置文件中没有指定）
	setDefaults(&config)

	GlobalConfig = &config

	// 可选：监听配置文件变化
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("配置文件已变更:", e.Name)
		if err := v.Unmarshal(&config); err != nil {
			log.Printf("重新加载配置失败: %v\n", err)
		} else {
			setDefaults(&config)
			GlobalConfig = &config
			log.Println("配置已重新加载")
		}
	})

	return nil
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	// 服务器默认配置
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.Mode == "" {
		config.Server.Mode = "debug"
	}

	// 数据库默认配置
	if config.Database.Driver == "" {
		config.Database.Driver = "mysql"
	}
	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}
	if config.Database.Port == 0 {
		config.Database.Port = 3306
	}
	// 数据库连接池默认值
	if config.Database.MaxIdleConns == 0 {
		config.Database.MaxIdleConns = 10
	}
	if config.Database.MaxOpenConns == 0 {
		config.Database.MaxOpenConns = 100
	}
	if config.Database.MaxLifetime == 0 {
		config.Database.MaxLifetime = time.Hour
	}
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return GlobalConfig
}

// GetDSN 获取数据库连接字符串
func (db *DatabaseConfig) GetDSN() string {
	switch db.Driver {
	case "mysql":
		// MySQL DSN 格式: username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			db.Username, db.Password, db.Host, db.Port, db.Database)
	case "postgres":
		// PostgreSQL DSN 格式
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			db.Host, db.Port, db.Username, db.Password, db.Database)
	default:
		return ""
	}
}

// String 实现 Stringer 接口，用于打印配置（注意隐藏密码）
func (c *Config) String() string {
	return fmt.Sprintf("Server: %+v, Database: {Driver:%s Host:%s Port:%d Username:%s Password:****** Database:%s}",
		c.Server, c.Database.Driver, c.Database.Host, c.Database.Port,
		c.Database.Username, c.Database.Database)
}
