
# 项目初始化代码，用于一键生成对应目录文件，请勿重复执行！！！！！！！！！！

# init-project.ps1
Write-Host "Creating Gin project structure..." -ForegroundColor Green

# 定义目录结构
$directories = @(
    "cmd\api",
    "internal\handler",
    "internal\service", 
    "internal\repository",
    "internal\model",
    "internal\router",
    "pkg\utils",
    "config",
    "middleware",
    "scripts"
)

# 创建目录
foreach ($dir in $directories) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Write-Host "  Created: $dir" -ForegroundColor Gray
}

# 创建 main.go
@"
package main

import "fmt"

func main() {
    fmt.Println("Gin server starting...")
}
"@ | Out-File -FilePath "cmd\api\main.go" -Encoding UTF8
Write-Host "  Created: cmd/api/main.go" -ForegroundColor Gray

# 创建 config.yaml
@"
server:
  port: 8080
  mode: debug

database:
  driver: mysql
  host: localhost
  port: 3306
"@ | Out-File -FilePath "config\config.yaml" -Encoding UTF8
Write-Host "  Created: config/config.yaml" -ForegroundColor Gray

# 创建 README.md
@"
# My Gin Project

## 目录结构
- `cmd/api/` - 主程序入口
- `internal/` - 私有代码
  - `handler/` - HTTP处理器
  - `service/` - 业务逻辑
  - `repository/` - 数据访问
  - `model/` - 数据模型
  - `router/` - 路由注册
- `pkg/` - 可导出的公共代码
- `config/` - 配置文件
- `middleware/` - Gin中间件
- `scripts/` - 辅助脚本
"@ | Out-File -FilePath "README.md" -Encoding UTF8
Write-Host "  Created: README.md" -ForegroundColor Gray

# 创建 .gitkeep 文件（可选）
New-Item -ItemType File -Path "cmd\.gitkeep" -Force | Out-Null
New-Item -ItemType File -Path "internal\.gitkeep" -Force | Out-Null
New-Item -ItemType File -Path "pkg\.gitkeep" -Force | Out-Null

# 初始化 go module
$moduleName = Read-Host "Enter module name (e.g., github.com/yourusername/project)"
if ([string]::IsNullOrWhiteSpace($moduleName)) {
    $moduleName = (Get-Item .).Name
    Write-Host "Using directory name as module name: $moduleName" -ForegroundColor Yellow
}

go mod init $moduleName

Write-Host "`n" 
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "✅ 项目初始化完成！" -ForegroundColor Green
Write-Host "📁 目录结构已创建" -ForegroundColor Green
Write-Host "`n🚀 下一步：" -ForegroundColor Yellow
Write-Host "   go mod tidy" -ForegroundColor White
Write-Host "========================================" -ForegroundColor Cyan