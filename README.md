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
- [部署手册](DEPLOY_GUIDE.md)
- [快速部署](#-快速部署)
- [低内存服务器部署指南](#-低内存服务器部署指南2g-及以下)
- [K8s Pod 监视器](#-k8s-pod-监视器可选组件)
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
| 后端 | Go 1.26 · Gin · Viper |
| 数据库 | MySQL 8.0 |
| 消息队列 | RabbitMQ（异步任务：审计日志 / 状态同步 / 统计刷新） |
| 中间件 | JWT 认证 · CORS · 限流 · 请求超时 · Worker Pool |
| 并发优化 | Goroutine + Channel（产品列表 / 产品详情并发查询） |
| 部署 | Docker · Docker Compose · Nginx（多阶段构建 + BuildKit 缓存） |
| 监控告警 | K8s Watcher（client-go Informer）· 钉钉机器人 Webhook |

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
│   ├── utils/
│   │   └── utils.go                 # 工具函数
│   ├── mq/
│   │   ├── connection.go            # RabbitMQ 连接管理 + 自动重连
│   │   ├── publisher.go             # 任务发布（JSON 序列化 → Publish）
│   │   └── consumer.go              # 消息消费（ACK/NACK + 重试）
│   └── worker/
│       ├── task.go                  # 任务类型定义（json.RawMessage Payload）
│       ├── pool.go                  # Worker Pool 核心（MQ 消费模式）
│       └── handlers.go              # 业务处理：审计日志 / 状态同步 / 统计刷新
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
├── k8s-watcher/                     # K8s Pod 重启告警监视器（可选组件）
│   ├── main.go                      # 入口：配置加载 / 集群认证 / Informer 监听 / 去重判断
│   ├── alert.go                     # 钉钉机器人消息发送（text 消息 + HMAC 加签）
│   ├── Dockerfile                   # 多阶段构建 → scratch 极简镜像（约 37MB，秒级启动）
│   ├── rbac.yaml                    # 最小权限授权（ServiceAccount + 只读 pods）
│   ├── deploy.yaml                  # K8s Deployment（Webhook 等配置在此填写）
│   └── go.mod                       # 独立 Go 模块（client-go v0.36，与主服务零耦合）
│
├── dockerfile                       # 多阶段构建（Node → Go → Nginx，BuildKit 缓存）
├── docker-compose.yml               # 容器编排（MySQL + RabbitMQ + App）
├── nginx.conf                       # Nginx 反向代理
├── start.sh                         # 容器启动脚本
├── rabbitmq/
│   └── rabbitmq.conf                # RabbitMQ 配置（内存水位等）
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
- ⚡ **并发查询**：产品列表 / 产品详情接口使用 Goroutine + Channel 并发查库（N+1 → 并行）
- 📨 **异步任务队列**：基于 RabbitMQ 的 Worker Pool，审计日志 / 状态同步 / 统计刷新异步执行，不阻塞 HTTP 响应
- ♻️ **任务可靠投递**：手动 ACK/NACK + 自动重试 + 断线重连，服务重启任务不丢失
- 🌐 **CORS 跨域**：开发环境 Vite 代理 + 生产环境 Nginx 反向代理
- 🛡 **安全防护**：参数校验、SQL 注入防护（预编译）、角色权限拦截、单 IP 限流（100 req/s）
- 🔭 **Pod 重启告警**：内置 client-go 监视器（可选组件），Pod 反复崩溃超阈值自动推送钉钉告警，详见「K8s Pod 监视器」章节

---

## 📡 API 文档

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/login` | 用户登录 |
| POST | `/api/v1/refresh-token` | 刷新 Token |
| GET | `/api/v1/products` | 获取所有产品（可选认证） |
| GET | `/api/v1/products/concurrent` | 获取所有产品（并发查询版） |
| GET | `/api/v1/products/concurrent/:product_id` | 获取产品详情（并发查询版） |
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
| GET | `/health` | 健康检查（含 Worker 统计） |
| GET | `/worker/stats` | Worker Pool 运行统计（任务总数/失败数/队列状态） |

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
RABBITMQ_PASSWORD=<你的 RabbitMQ 密码>
```

> 生成随机密钥：`openssl rand -base64 32`

### 3. 构建 & 启动

```bash
# ✅ 分两步：先构建镜像，再启动容器
# 【为什么不写 --no-parallel？】
#   Compose V2 的 docker compose build 没有 --no-parallel 参数（那是 V1 的）。
#   本项目只有 app 一个服务需要构建（mysql/rabbitmq 是直接拉镜像），
#   不存在多服务并行问题，直接 build 即可。
docker compose build
docker compose up -d
```

> ❌ 不要用 `docker compose up -d --build`（构建 + 启动混一起，出错时不好排查）。

首次启动会自动完成：拉取镜像 → 编译前端 → 编译 Go 后端 → 初始化数据库 → 启动 RabbitMQ。
由于 Dockerfile 使用 BuildKit 分层缓存，**增量构建仅需数秒**（只重编译变化的层）。

> 💡 **小内存服务器（2G）额外建议**：
> 1. 加 Swap 交换分区：`fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile`
>    （并写入 `/etc/fstab` 开机自动挂载）
> 2. 构建前先停掉占用大的其他容器（`docker stop <容器名>`），构建完再启动
> 3. 内存还紧张可用：`BUILDKIT_MAX_PARALLELISM=1 docker compose build`（限制 BuildKit 一次只构建一个 stage）
> 4. Dockerfile 已内置内存限制：前端 Node 堆内存 1G、Go 编译 `-p=1` 串行

### 4. 验证

```bash
# 检查容器状态（应看到 3 个容器：mysql / rabbitmq / app）
docker compose ps

# 测试 API（容器内直连 Go 后端 8080，不走 Nginx 前缀）
docker exec supply-chain-app curl -s http://localhost:8080/health

# 查看 Worker Pool 统计（需带前缀）
curl -s http://localhost/supply_chain/api/v1/products | head -c 200

# 测试前端（带前缀返回 200）
curl -s -o /dev/null -w "%{http_code}" http://localhost:80/supply_chain/
```

浏览器访问 `http://<服务器公网IP>/supply_chain`，使用预设管理员账号登录。
访问根路径 `http://<服务器公网IP>/` 会自动 301 重定向到 `/supply_chain/`。

> 📌 **路径前缀说明**：系统部署在 `/supply_chain` 子路径下，这是为了**一个服务器部署多个项目**时通过路径区分。
> 如需修改前缀，改三处即可：`vite.config.js` 的 `base`、`src/api.js` 的 `baseURL`、`nginx.conf` 的两个 `location`。
> 注意：修改前缀后前端必须重新构建（`npm run build`）。

### 5. RabbitMQ 管理界面（可选）

```bash
# 通过 SSH 隧道访问（不要对公网开放 15672 端口！）
ssh -L 15672:localhost:15672 root@服务器IP
# 浏览器打开 http://localhost:15672  (账号密码见 .env)
```

---

## 🐌 低内存服务器部署指南（2G 及以下）

> 云服务器常见配置是 2G 内存（还跑着其他服务时可用内存更少），
> Docker 多阶段构建（vite 打包 + Go 编译）容易 OOM。本指南按"内存紧张等级"递进。

### 0. 先诊断：确认内存到底够不够

```bash
# 内存 + swap 使用情况
free -h

# 看谁被杀过（OOM killer 记录）
dmesg | grep -i "oom\|killed"

# swap 是否生效
swapon --show
```

### 1. 建立 Swap 交换分区（内存不足的"第二层保险"）

> Swap 用磁盘充当内存，**让系统不因瞬时内存峰值直接崩掉**。2G 内存建议配 2G swap。

```bash
# 创建 2G swap 文件（fallocate 秒建，比 dd 快）
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile              # 只允许 root 读写（安全）
sudo mkswap /swapfile                 # 格式化为 swap
sudo swapon /swapfile                 # 启用

# 开机自动挂载（防止重启后失效）
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# 验证
free -h                               # Swap 一栏应显示 2G
```

> 若 `fallocate` 不支持（个别文件系统），用传统方式：
> `sudo dd if=/dev/zero of=/swapfile bs=1M count=2048`

### 2. 构建前释放内存（关键！）

> 构建是内存峰值最高的时刻。先停掉不用的容器，把内存让给构建。

```bash
# ① 停掉本项目的运行容器（MySQL/RabbitMQ 各占 ~512M，很能抢）
cd Supply_chain_traceability
docker compose down

# ② 清理构建缓存垃圾（安全，只删无用缓存）
docker system prune -f

# ③ 确认内存空出来了
free -h

# ④ 构建 + 启动
docker compose build
docker compose up -d
```

### 3. 让系统更积极使用 Swap（小内存机器强烈建议）

> Linux 默认 `swappiness=60`（60% 内存才用 swap）。小内存机器调到 100，
> 让系统**尽早把不常用数据挪到 swap**，给构建留出物理内存。

```bash
# 立即生效
echo 100 > /proc/sys/vm/swappiness

# 永久生效（写入 sysctl 配置）
echo 'vm.swappiness = 100' | sudo tee -a /etc/sysctl.conf
```

### 4. 限制 BuildKit 并行构建（内存峰值减半）

> 本 Dockerfile 是**多阶段构建**（frontend / backend / 最终镜像 3 个 stage）。
> BuildKit 默认并行执行这些 stage，内存峰值 = 各阶段之和（vite 1G + go 编译 500M + ...）。
> 用环境变量强制**一次只构建一个 stage**，峰值降到最大的那个（约 1G）。

```bash
# ★ 串行构建：一次只编译一个阶段
BUILDKIT_MAX_PARALLELISM=1 docker compose build
docker compose up -d
```

### 5. 终极方案：本地构建，服务器只加载（服务器零构建压力）

> 如果服务器内存实在挤不出（还有其他重要服务），
> 就在**本地电脑**（内存大）构建镜像，打包传过去，服务器只 `docker load`。

```bash
# ① 本地 Windows（需装 Docker Desktop），在项目根目录执行：
docker build -t supply-chain-app:latest .
docker save supply-chain-app:latest -o supply-app.tar

# ② 传到服务器（scp 走 SSH，比镜像仓库稳）
scp supply-app.tar developer@服务器IP:/home/developer/

# ③ 服务器上加载（秒完成，不构建！）
docker load -i /home/developer/supply-app.tar

# ④ docker-compose.yml 的 app 服务加一行，改用本地镜像
#    image: supply-chain-app:latest
#    然后 docker compose up -d 直接启动
```

### 6. Docker 已内置的内存保护（了解即可）

| 限制 | 位置 | 作用 |
|------|------|------|
| `NODE_OPTIONS=--max-old-space-size=1024` | Dockerfile 前端阶段 | vite 打包堆内存封顶 1G |
| `GOMAXPROCS=1` + `GOGC=100` | Dockerfile 后端阶段 | Go 编译器单核运行 |
| `go build -p=1` | Dockerfile 后端阶段 | 一次只编译 1 个包 |
| `mem_limit`（384m/768m/512m） | docker-compose | MySQL/RabbitMQ/App 运行期内存上限 |
| `rabbitmq/rabbitmq.conf` | 配置文件挂载 | RabbitMQ 内存水位 0.5（2G 机器放宽，避免误告警拒接连接） |

> ⚠️ **RabbitMQ 配置注意**：新版 RabbitMQ 已**弃用** `RABBITMQ_VM_MEMORY_HIGH_WATERMARK` 环境变量，
> 设置它会直接启动失败（`deprecated environment variables detected`）。
> 内存水位等高级配置必须用配置文件（本项目的 `rabbitmq/rabbitmq.conf`）挂载。

---

## 🔍 K8s Pod 监视器（可选组件）

`k8s-watcher/` 是一个可选的独立组件：部署在 Kubernetes 集群里的告警哨兵。它通过 client-go 的 Informer 机制实时监听集群内所有 Pod，当某个 Pod 的容器累计重启次数**超过阈值（默认 3 次）**——即进入 CrashLoopBackOff 反复崩溃——时，自动调用钉钉群机器人推送告警。

> 特点：独立 Go 模块（不增加主服务任何依赖）· scratch 极简镜像（约 37MB，毫秒级启动）· 资源占用极低（内存 16Mi 起步）· Webhook 未配置时自动进入 DRY-RUN 模式（告警只打日志、不发送），方便先验证再接入钉钉。

### 前置要求

- 一个可用的 Kubernetes 集群（本地推荐 **Docker Desktop**：Settings → Kubernetes → Enable Kubernetes）
- 已安装 kubectl（Docker Desktop 自带）
- 一个钉钉群（接收告警用）

### 1. 创建钉钉机器人

1. 打开钉钉群 → 群设置 → 机器人 → 添加机器人 → **自定义（通过 Webhook 接入）**
2. 安全设置建议两项都勾：
   - **自定义关键词**：填 `告警`（本程序的消息固定以【K8s 告警】开头，可直接命中）
   - **加签**：记录生成的密钥（`SEC` 开头那串）
3. 完成后会得到 **Webhook 地址**（形如 `https://oapi.dingtalk.com/robot/send?access_token=xxxx`），注意妥善保管，泄漏后任何人都能向群里发消息

### 2. 配置并部署

```bash
# ① 编辑 k8s-watcher/deploy.yaml，填入两个环境变量：
#    WEBHOOK_URL      ← 第 1 步拿到的 Webhook 地址
#    DINGTALK_SECRET  ← 加签密钥（SEC 开头；未开启加签则留空）

# ② 构建镜像（在项目根目录执行，国内网络已内置加速）
docker build -t supply-chain-watcher:latest ./k8s-watcher

# ③ 部署（先授权、再部署，顺序不能反）
kubectl apply -f k8s-watcher/rbac.yaml
kubectl apply -f k8s-watcher/deploy.yaml

# ④ 观察日志，出现「已开始监视 pods」即部署成功
#    若持续刷 Forbidden / 403，说明 rbac.yaml 未生效
kubectl logs -f deploy/supply-chain-watcher
```

### 3. 验证告警

部署一个故意崩溃的测试 Pod，等它反复重启超过阈值：

```bash
kubectl run crash-test --image=docker.m.daocloud.io/library/busybox:1.36 \
  --restart=Always -- sh -c "echo boom; exit 1"

# 盯 RESTARTS 列，2~3 分钟内涨到 4，钉钉群收到告警
kubectl get pod crash-test -w

# 验证完清理
kubectl delete pod crash-test --now
```

### 配置项（k8s-watcher/deploy.yaml 的 env）

| 环境变量 | 默认值 | 说明 |
|------|------|------|
| `WEBHOOK_URL` | 空 | 钉钉机器人 Webhook；**留空 = DRY-RUN 只打日志不发送** |
| `DINGTALK_SECRET` | 空 | 加签密钥（SEC 开头）；机器人未开启加签则留空 |
| `RESTART_THRESHOLD` | `3` | 重启次数阈值，**超过**该值才告警 |
| `NAMESPACE` | 空（全部） | 只监视指定命名空间时填写 |

### 告警行为说明

- **去重防抖**：同一 Pod 不会反复刷屏——每再多崩满一个阈值次数（4 → 7 → 10…）才再报一次；Pod 被删除后状态自动清理
- **失败自动重试**：钉钉发送失败（网络抖动等）会在下个同步周期（60 秒）自动重试，无需干预
- **已完结 Pod 不告警**：Job 类正常跑完的 Pod（Succeeded/Failed）自动跳过
- **重启口径**：与 `kubectl get pods` 的 RESTARTS 列一致（不含 init 容器），方便核对

### 卸载

```bash
kubectl delete -f k8s-watcher/deploy.yaml
kubectl delete -f k8s-watcher/rbac.yaml
```

---

## 💻 本地开发

### 后端

```bash
# 安装依赖
go mod download

# 需要先启动本地 RabbitMQ（Docker 方式）
# ⚠️ 注意：不要设 RABBITMQ_VM_MEMORY_HIGH_WATERMARK 环境变量（新版已弃用，会启动失败）
# 需要自定义内存水位等配置时，用 -v 挂载 rabbitmq/rabbitmq.conf
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=supply_mq -e RABBITMQ_DEFAULT_PASS=SupplyMQ@2024 \
  -v "$(pwd)/rabbitmq/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro" \
  docker.m.daocloud.io/library/rabbitmq:3-management

# 启动（监听 :8080）
go run ./cmd/api/
```

配置文件：`config/config.yaml`

### 前端

```bash
cd supply-chain-frontend
npm install
npm run dev            # 监听 :3000，访问 http://localhost:3000/supply_chain/
```

> 💡 开发环境也带 `/supply_chain` 前缀（与生产一致），API 由 Vite proxy 剥离前缀后转发到 `:8080`。

### 数据库

本地安装 MySQL 8.0，执行 `mysql_sql/init.sql` 初始化表结构和种子数据。

---

## 🔧 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MYSQL_ROOT_PASSWORD` | MySQL root 密码 | `SupplyChain@2024` |
| `RABBITMQ_USER` | RabbitMQ 账号 | `supply_mq` |
| `RABBITMQ_PASSWORD` | RabbitMQ 密码 | `SupplyMQ@2024` |
| `APP_PORT` | 应用对外端口 | `80` |
| `GIN_MODE` | Gin 运行模式 | `release` |
| `JWT_SECRET` | JWT 签名密钥 | `your-secret-key` |
| `DB_HOST` | 数据库地址 | `localhost` |
| `DB_PORT` | 数据库端口 | `3306` |
| `DB_USER` | 数据库用户 | `root` |
| `DB_PASSWORD` | 数据库密码 | 同 `MYSQL_ROOT_PASSWORD` |
| `DB_NAME` | 数据库名 | `supply_chain` |
| `MQ_HOST` | RabbitMQ 地址 | `localhost` |
| `MQ_PORT` | RabbitMQ 端口 | `5672` |

---

## 📄 许可

本项目代码开源于 GitHub，供个人学习、研究、非商业用途免费使用。

**商业使用**（包括但不限于：直接销售、SaaS 化运营、集成到付费产品、为企业提供部署服务并收费）**须事先获得作者的书面授权**。

如需商业授权，请联系--微信：**2445756144yb**

> Copyright © 2026 Prey-Isis. All rights reserved.
