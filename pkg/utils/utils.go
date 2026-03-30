// pkg/utils/utils.go
package utils

import (
	"log"

	"main/config"
)

// InitConfig 初始化配置
func InitConfig(configPath string) *config.Config {
	if err := config.InitConfig(configPath); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	cfg := config.GetConfig()
	log.Printf("配置加载成功: %v", cfg)

	return cfg
}
