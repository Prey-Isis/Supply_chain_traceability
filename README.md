# My Gin Project

## 目录结构
- cmd/api/ - 主程序入口
- internal/ - 私有代码
  - handler/ - HTTP处理器
  - service/ - 业务逻辑
  - epository/ - 数据访问
  - model/ - 数据模型
  - outer/ - 路由注册
- pkg/ - 可导出的公共代码
- config/ - 配置文件
- middleware/ - Gin中间件
- scripts/ - 辅助脚本
