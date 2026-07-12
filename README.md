<div align="center">

<img src="supply-chain-frontend/public/favicon.svg" width="80" alt="logo" />

# 供应链溯源系统

### Supply Chain Traceability System

</div>

---

## 📖 目录

- [项目简介](#-项目简介)
- [技术栈](#-技术栈)
- [目录结构](#-目录结构)
- [功能特性](#-功能特性)
- [API 文档](#-api-文档)
- [快速部署](#-快速部署)
- [本地开发](#-本地开发)
- [环境变量](#-环境变量)

---

## 📌 项目简介

供应链溯源系统是一套面向供应链全链路监管的 Web 应用，支持**产品注册、流转记录录入、多节点溯源查询**，并提供完整的三级权限体系（管理员 / 经理 / 普通用户）。前端采用 Vue 3 构建，后端基于 Go + Gin 框架，数据库使用 MySQL，通过 Docker Compose 一键部署。

---

## 🛠 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | Vue 3 · Vite · Axios · Element Plus |
| 后端 | Go 1.26 · Gin · GORM · Viper |
| 数据库 | MySQL 8.0 |
| 中间件 | JWT 认证 · CORS · 限流 · 请求超时 |
| 部署 | Docker · Docker Compose · Nginx |

---

## 📁 目录结构

```
Supply_chain_traceability/
│
├── cmd/
│   └── api/
│       └── main.go                  # 应用入口
│
├── config/
│   ├── config.go                    # 配置加载（Viper）
│   └── config.yaml                  # 默认配置
│
├── internal/
│   ├── jwt/
│   │   └── jwt.go                   # JWT 签发 & 解析
│   ├── model/
│   │   └── Model.go                 # 数据模型 & 数据库操作
│   └── router/
│       └── router.go                # 路由处理器（Handler）
│
├── middleware/
│   └── middleware.go                # 认证 · CORS · 限流 · 超时
│
├── pkg/
│   └── utils/
│       └── utils.go                 # 工具函数
│
├── mysql_sql/
│   ├── init.sql                     # 合并建表 + 种子数据
│   ├── user.sql                     # 用户表 DDL
│   ├── product.sql                  # 产品表 DDL
│   └── supply_history.sql           # 供应链历史表 DDL
│
├── supply-chain-frontend/           # Vue 3 前端
│   ├── src/
│   │   ├── api.js                   # API 封装
│   │   ├── main.js                  # 入口
│   │   ├── store.js                 # Pinia 状态管理
│   │   ├── App.vue                  # 根组件
│   │   ├── style.css                # 全局样式
│   │   ├── views/
│   │   │   ├── Login.vue            # 登录页
│   │   │   ├── Layout.vue           # 主布局（导航框架）
│   │   │   ├── Products.vue         # 产品管理
│   │   │   └── Users.vue            # 用户管理（管理员）
│   │   ├── components/
│   │   │   ├── ProductDetail.vue    # 产品溯源详情
│   │   │   ├── ProductDialog.vue    # 产品编辑弹窗
│   │   │   └── UserDialog.vue       # 用户编辑弹窗
│   │   └── assets/                  # 静态资源
│   ├── vite.config.js               # Vite 配置（含 API 代理）
│   └── package.json
│
├── dockerfile                       # 多阶段构建（Node → Go → Nginx）
├── docker-compose.yml               # 容器编排（MySQL + App）
├── nginx.conf                       # Nginx 反向代理
├── start.sh                         # 容器启动脚本
├── .env.example                     # 环境变量模板
├── .dockerignore
├── go.mod
└── README.md
```

---

## ✨ 功能特性

- 🔐 **JWT 认证**：登录签发 Token，支持刷新，Cookie + Authorization Header 双通道
- 👥 **三级角色**：`admin`（管理员）> `manager`（经理）> `user`（普通用户）
- 📦 **产品注册**：创建产品并可选附带初始供应链历史记录（事务写入）
- 🔗 **溯源链**：按时间线展示产品从生产到交付的完整流转链路
- 📊 **批量导入**：支持一次性导入多条供应链历史记录
- 🌐 **CORS 跨域**：开发环境 Vite 代理 + 生产环境 Nginx 反向代理
- ⚡ **限流保护**：单 IP 每秒 100 请求上限
- 🛡 **安全防护**：参数校验、SQL 注入防护（预编译）、角色权限拦截

---

## 📡 API 文档

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/login` | 用户登录 |
| POST | `/api/v1/refresh-token` | 刷新 Token |
| GET | `/api/v1/products` | 获取所有产品（可选认证） |
| GET | `/api/v1/products/:product_id` | 获取产品详情（可选认证） |
| GET | `/api/v1/products/:product_id/history` | 获取产品溯源历史 |
| GET | `/api/v1/supply-history` | 获取所有历史记录 |

### 需认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/user/current` | 获取当前用户信息 |
| POST | `/api/v1/logout` | 登出 |
| POST | `/api/v1/products` | 创建产品 |
| PUT | `/api/v1/products/:product_id` | 更新产品 |
| PATCH | `/api/v1/products/:product_id/status` | 更新产品状态 |
| POST | `/api/v1/supply-history` | 创建供应链历史 |
| POST | `/api/v1/supply-history/batch` | 批量创建历史记录 |

### 管理员接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/admin/users` | 创建用户 |
| GET | `/api/v1/admin/users` | 获取所有用户 |
| GET | `/api/v1/admin/users/:account` | 获取指定用户 |
| PUT | `/api/v1/admin/users/:account` | 更新用户 |
| DELETE | `/api/v1/admin/users/:account` | 删除用户 |
| DELETE | `/api/v1/admin/products/:product_id` | 删除产品 |

### 通用

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |

---

## 🚀 快速部署

### 前置要求

- 服务器：Ubuntu 20.04+（或其他 Linux 发行版）
- 已安装 [Docker](https://docs.docker.com/engine/install/) & [Docker Compose](https://docs.docker.com/compose/install/)
- 阿里云安全组入方向放行 **TCP 80** 端口

### 1. 克隆项目

```bash
git clone git@github.com:Prey-Isis/Supply_chain_traceability.git
cd Supply_chain_traceability
```

### 2. 配置环境变量

```bash
cp .env.example .env
vim .env
```

必须修改的项：

```env
MYSQL_ROOT_PASSWORD=<你的 MySQL 密码>
JWT_SECRET=<随机生成的密钥>
```

> 生成随机密钥：`openssl rand -base64 32`

### 3. 启动

```bash
docker compose up -d
```

首次启动会自动完成：拉取镜像 → 编译前端 → 编译 Go 后端 → 初始化数据库。

### 4. 验证

```bash
# 检查容器状态
docker compose ps

# 测试 API
docker exec supply-chain-app curl -s http://localhost:8080/health

# 测试前端
curl -s -o /dev/null -w "%{http_code}" http://localhost:80/
```

浏览器访问 `http://<服务器公网IP>`，使用预设管理员账号登录。

---

## 💻 本地开发

### 后端

```bash
# 安装依赖
go mod download

# 启动（监听 :8080）
go run ./cmd/api/
```

配置文件：`config/config.yaml`

### 前端

```bash
cd supply-chain-frontend
npm install
npm run dev            # 监听 :3000，API 自动代理到 :8080
```

### 数据库

本地安装 MySQL 8.0，执行 `mysql_sql/init.sql` 初始化表结构和种子数据。

---

## 🔧 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYSQL_ROOT_PASSWORD` | MySQL root 密码 | `SupplyChain@2024` |
| `APP_PORT` | 应用对外端口 | `80` |
| `GIN_MODE` | Gin 运行模式 | `release` |
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key` |
| `DB_HOST` | 数据库地址 | `localhost` |
| `DB_PORT` | 数据库端口 | `3306` |
| `DB_USER` | 数据库用户 | `root` |
| `DB_PASSWORD` | 数据库密码 | 同 `MYSQL_ROOT_PASSWORD` |
| `DB_NAME` | 数据库名 | `supply_chain` |

---

## 📄 许可

本项目代码开源于 GitHub，供个人学习、研究、非商业用途免费使用。

**商业使用**（包括但不限于：直接销售、SaaS 化运营、集成到付费产品、为企业提供部署服务并收费）**须事先获得作者的书面授权**。

如需商业授权，请联系--微信：**2445756144yb**

> Copyright © 2026 Prey-Isis. All rights reserved.
