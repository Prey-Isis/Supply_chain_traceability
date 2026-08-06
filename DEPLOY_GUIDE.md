# 🚀 供应链溯源系统 · 从拉取到部署到运行

> 本手册将引导你完成：**拉取代码 → 环境检查 → 配置环境变量 → Docker 构建 → 启动 → 验证 → 运维**。
>
> 技术栈：**Go + Gin 后端 / Vue3 前端 / MySQL / RabbitMQ / Nginx / Docker Compose**

---

## 📋 使用说明

1. 在服务器上以 **root 或具有 docker 权限的用户** 执行以下命令
2. 从上到下**按顺序执行**，不要跳过
3. 每段命令都是独立的，失败会直接报错，便于排查
4. 命令均为**幂等**设计，重复执行不会破坏环境

> ⚠️ **前置要求**：需要已安装 Docker 和 Docker Compose（V2）。

---

## 0️⃣ 环境检查

先确认服务器是否满足部署条件：Docker、Docker Compose 是否可用。

```bash
# 检查 Docker 版本
echo "===== Docker 版本 ====="
docker --version || { echo "❌ 未安装 Docker，请先安装！"; exit 1; }

# 检查 Docker Compose（新版是 docker compose 子命令）
echo "===== Compose 版本 ====="
docker compose version || docker-compose --version || { echo "❌ 未安装 Docker Compose！"; exit 1; }

# 检查 Docker 守护进程是否运行
echo "===== Docker 状态 ====="
docker info > /dev/null 2>&1 && echo "✅ Docker 守护进程运行中" || { echo "❌ Docker 未运行，请执行: systemctl start docker"; exit 1; }

echo ""
echo "✅ 环境检查通过，可以开始部署！"
```

---

## 1️⃣ 拉取项目代码

从 GitHub 克隆项目到当前目录。

> 💡 **提示**：如果目录已存在会执行 `git pull` 更新（相当于更新代码）。

```bash
# 如果目录不存在则克隆，存在则拉取最新代码
if [ ! -d "Supply_chain_traceability/.git" ]; then
    echo "📥 首次拉取，正在克隆仓库..."
    git clone https://github.com/Prey-Isis/Supply_chain_traceability.git
else
    echo "🔄 目录已存在，正在拉取最新代码..."
    cd Supply_chain_traceability && git pull
fi

# 进入项目目录
cd Supply_chain_traceability
echo "📍 当前目录: $(pwd)"
echo "📁 项目文件:"
ls -la
```

---

## 2️⃣ 配置环境变量

项目通过 `.env` 文件注入敏感配置（数据库密码、JWT 密钥、RabbitMQ 密码）。

> 🔒 **安全提醒**：`.env` 已被 `.gitignore` 排除，**不会**被提交到 GitHub。

```bash
cd Supply_chain_traceability

# 复制模板（如果 .env 已存在则不覆盖，避免丢失已有配置）
if [ -f ".env" ]; then
    echo "⚠️  .env 已存在，保留现有配置"
    echo "📋 当前 .env 内容:"
    cat .env
else
    echo "📝 首次配置，从模板创建 .env..."
    cp .env.example .env
    echo "✅ 已创建 .env，请下一步修改密码！"
fi
```

### 2.1 自动生成强密码并写入 .env

> ⚠️ **注意**：以下命令会**覆盖** `.env` 中的密码字段，生成随机强密码。
> 如果你已手动配置过，可跳过此节。

```bash
cd Supply_chain_traceability

# 生成强随机密码（去掉特殊字符，避免 sed 转义问题）
MYSQL_PASS=$(openssl rand -base64 12 | tr -d '/+=' )
MQ_PASS=$(openssl rand -base64 12 | tr -d '/+=' )
JWT_SECRET=$(openssl rand -base64 32 | tr -d '/+=')

# 用 sed 替换 .env 中的占位值
sed -i "s/^MYSQL_ROOT_PASSWORD=.*/MYSQL_ROOT_PASSWORD=${MYSQL_PASS}/" .env
sed -i "s/^RABBITMQ_PASSWORD=.*/RABBITMQ_PASSWORD=${MQ_PASS}/" .env
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${JWT_SECRET}/" .env

echo "✅ 已生成随机密码并写入 .env"
echo ""
echo "🔑 重要！请立即记下以下密码（仅显示一次）："
echo "   MySQL:     ${MYSQL_PASS}"
echo "   RabbitMQ:  ${MQ_PASS}"
echo "   JWT:       ${JWT_SECRET}"
echo ""
echo "⚠️  如果丢失，需删除 mysql 数据卷重新初始化！"
```

---

## 3️⃣ 首次构建 & 启动

> 🔨 **首次构建说明**：
> - 拉取 Node / Go / Nginx / MySQL / RabbitMQ 基础镜像（1-3 分钟，取决于网络）
> - 多阶段构建编译前端 + 后端
> - MySQL 首次启动自动执行 `mysql_sql/init.sql` 建表 + 导入种子数据
>
> ⏱️ 整体耗时约 3-10 分钟，耐心等待。

```bash
cd Supply_chain_traceability

echo "🚀 开始构建并启动容器（首次构建较慢）..."
docker compose up -d --build

echo ""
echo "✅ 构建启动命令已执行，查看状态:"
docker compose ps
```

---

## 4️⃣ 验证部署结果

依次检查：容器状态 → 健康检查 → 登录测试。

```bash
cd Supply_chain_traceability

echo "===== 1. 容器状态 ====="
docker compose ps

echo ""
echo "===== 2. 健康检查（应返回 ok）====="
# 从 app 容器内探测 Go 后端
docker exec supply-chain-app curl -s http://localhost:8080/health || echo "❌ 健康检查失败"

echo ""
echo "===== 3. 前端页面（80 端口）====="
curl -s -o /dev/null -w "HTTP 状态码: %{http_code}\n" http://localhost/ 2>/dev/null || echo "（端口未放行，见排查章节）"
```

```bash
cd Supply_chain_traceability

echo "===== 登录测试（种子数据管理员账号）====="
curl -s -X POST http://localhost/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"Account":"11111111","PassWord":"123456"}' | head -c 300
echo ""
```

---

## 5️⃣ 运维常用命令

部署成功后，日常维护会用到以下命令，按需执行。

### 查看实时日志

```bash
cd Supply_chain_traceability
echo "===== 查看实时日志（Ctrl+C 退出）====="
docker compose logs -f app
```

### 代码更新后重新部署

```bash
cd Supply_chain_traceability

echo "===== 代码更新后重新部署 ====="
git pull
docker compose up -d --build
echo "✅ 已更新到最新版本"
```

### 停止服务 / 完全清理

```bash
cd Supply_chain_traceability

echo "===== 停止服务（保留数据）====="
docker compose down

echo "===== 完全清理（删除数据卷！不可恢复）====="
echo "⚠️ 如需彻底清理，执行：docker compose down -v"
```

---

## 6️⃣ RabbitMQ 管理界面（可选）

RabbitMQ 自带 Web 管理界面，但**不要**对公网开放端口，用 SSH 隧道访问最安全。

```bash
echo "===== 方法：SSH 隧道访问 RabbitMQ 管理界面 ====="
echo "在本地电脑执行（不是服务器）："
echo "  ssh -L 15672:localhost:15672 root@<服务器IP>"
echo "然后浏览器打开: http://localhost:15672"
echo "账号密码: 见 .env 中的 RABBITMQ_USER / RABBITMQ_PASSWORD"
echo ""
echo "===== 或查看容器内 RabbitMQ 是否健康 ====="
docker ps --filter name=rabbitmq --format "{{.Status}}"
```

---

## 7️⃣ 常见问题排查

| 现象 | 可能原因 | 解决 |
|------|---------|------|
| `docker: command not found` | 未安装 Docker | `apt install docker.io` 或参考官方文档 |
| 构建报 `unknown instruction: MOUNT` | Docker 版本过旧，不支持 BuildKit | `DOCKER_BUILDKIT=0 docker compose up -d --build` 降级 |
| MySQL 连接失败 | 密码不匹配 | 检查 `.env` 的 `MYSQL_ROOT_PASSWORD`，与容器初始化一致 |
| 前端打不开 | 安全组未放行 80 端口 | 阿里云控制台 → 安全组 → 入方向放行 TCP 80 |
| RabbitMQ 连接失败 | MQ 未就绪或密码不对 | `docker compose logs rabbitmq` 查看日志 |
| 数据初始化失败 | init.sql 未执行 | 删除 mysql 数据卷重新初始化（`docker compose down -v`） |

---

## 🎉 恭喜！

如果以上步骤全部通过，你的供应链溯源系统已成功部署！

- 前端访问：`http://<服务器公网IP>`
- 管理员账号：`11111111` / `123456`（种子数据）
- API 文档：见项目 README.md

> 📖 更多技术细节（并发优化、Worker Pool、RabbitMQ 集成）请阅读项目源码及注释。
>
> Copyright © 2026 Prey-Isis. All rights reserved.
