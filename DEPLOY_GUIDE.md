# 🚀 供应链溯源系统 · 从拉取到部署到运行

> 本手册将引导你完成：**拉取代码 → 环境检查 → 配置环境变量 → Docker 构建 → 启动 → 验证 → 运维**。
>
> 技术栈：**Go + Gin 后端 / Vue3 前端 / MySQL / RabbitMQ / Nginx / Docker Compose / K8s Watcher（可选）**

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
>
> ⚠️ **分两步构建**，原因：
> - 先构建镜像、再启动容器，出错时便于定位（是构建失败还是启动失败）
> - ❌ 不要用 `docker compose up -d --build`（构建 + 启动混一起，不好排查）
> - 注：Compose V2 的 `docker compose build` **没有** `--no-parallel` 参数（那是 V1 的）
> - 本项目只有 app 一个服务需要构建，不存在多服务并行问题
> - 2G 内存紧张时可用：`BUILDKIT_MAX_PARALLELISM=1 docker compose build`（限制 BuildKit 一次只构建一个 stage）

```bash
cd Supply_chain_traceability

echo "🚀 第 1 步：构建镜像..."
docker compose build

echo ""
echo "🚀 第 2 步：启动容器..."
docker compose up -d

echo ""
echo "✅ 构建启动完成，查看状态:"
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
echo "===== 3. 前端页面（80 端口，带前缀）====="
curl -s -o /dev/null -w "HTTP 状态码: %{http_code}\n" http://localhost/supply_chain/ 2>/dev/null || echo "（端口未放行，见排查章节）"
```

> 📌 **路径前缀**：系统访问路径为 `http://IP/supply_chain`（根路径 `/` 会自动 301 跳转过去），
> 这是为了一个服务器部署多个项目时通过路径区分。

```bash
cd Supply_chain_traceability

echo "===== 登录测试（种子数据管理员账号，带前缀）====="
curl -s -X POST http://localhost/supply_chain/api/v1/login \
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

> ⚠️ 同样分两步（先构建后启动，便于排查）

```bash
cd Supply_chain_traceability

echo "===== 代码更新后重新部署 ====="
git pull
docker compose build                 # 先构建镜像（增量，秒级）
docker compose up -d                 # 再启动新容器
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

## 🆕 7️⃣ 内存紧张时的操作（2G 及以下服务器必看）

> 2G 内存构建多阶段 Docker 镜像容易 OOM（vite 打包 + Go 编译峰值高）。
> 按以下步骤递进处理，从"加 swap"到"本地构建"共 5 招。

### 7.1 先诊断内存状态

```bash
free -h                            # 内存 + swap 使用情况
dmesg | grep -i "oom\|killed"      # 看谁被 OOM killer 杀过
swapon --show                      # swap 是否生效
```

### 7.2 建立 Swap 交换分区（第一招）

> Swap 用磁盘当内存，系统瞬时内存峰值时不会直接崩。2G 内存建议配 2G swap。

```bash
# 创建 2G swap 文件（fallocate 秒建）
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 开机自动挂载
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# 验证
free -h
```

> `fallocate` 不支持时用：`sudo dd if=/dev/zero of=/swapfile bs=1M count=2048`

### 7.3 构建前释放内存（第二招）

> 构建是内存峰值时刻。先停容器、清缓存，把内存让给构建。

```bash
cd Supply_chain_traceability

docker compose down          # 停 MySQL/RabbitMQ（各占 ~512M）
docker system prune -f       # 清构建缓存（安全）
free -h                      # 确认内存空出来

docker compose build         # 再构建
docker compose up -d
```

### 7.4 让系统更积极用 Swap（第三招）

```bash
# 立即生效（默认 60，小内存调到 100）
echo 100 > /proc/sys/vm/swappiness

# 永久生效
echo 'vm.swappiness = 100' | sudo tee -a /etc/sysctl.conf
```

### 7.5 限制 BuildKit 串行构建（第四招）

> 多阶段构建默认**并行**执行各 stage，内存峰值 = 各阶段之和。
> 强制一次只构建一个 stage，峰值降到最大的那个（约 1G）。

```bash
BUILDKIT_MAX_PARALLELISM=1 docker compose build
docker compose up -d
```

### 7.6 终极：本地构建，服务器只加载（第五招）

> 服务器内存实在挤不出时，在本地（内存大）构建，打包传过去。

```bash
# ① 本地电脑（Windows + Docker Desktop）项目根目录：
docker build -t supply-chain-app:latest .
docker save supply-chain-app:latest -o supply-app.tar

# ② 传到服务器
scp supply-app.tar developer@服务器IP:/home/developer/

# ③ 服务器加载（不构建，秒完成）
docker load -i /home/developer/supply-app.tar

# ④ docker-compose.yml 的 app 服务加一行：
#    image: supply-chain-app:latest
#    然后 docker compose up -d
```

---

## 8️⃣ 常见问题排查

| 现象 | 可能原因 | 解决 |
|------|---------|------|
| `docker: command not found` | 未安装 Docker | `apt install docker.io` 或参考官方文档 |
| 构建报 `unknown instruction: MOUNT` | Docker 版本过旧，不支持 BuildKit | `DOCKER_BUILDKIT=0 docker compose build` 降级 |
| MySQL 连接失败 | 密码不匹配 | 检查 `.env` 的 `MYSQL_ROOT_PASSWORD`，与容器初始化一致 |
| 前端打不开 | 安全组未放行 80 端口 | 阿里云控制台 → 安全组 → 入方向放行 TCP 80 |
| RabbitMQ 连接失败（容器 Restarting） | 设置了已弃用的环境变量 `RABBITMQ_VM_MEMORY_HIGH_WATERMARK` | **移除该环境变量**，改用配置文件 `rabbitmq/rabbitmq.conf` 挂载设置内存水位 |
| RabbitMQ 连接失败（日志报 system_memory_high_watermark 告警） | 2G 机器默认内存水位 40%(800M)，系统空闲跌破即拒接连接 | `rabbitmq/rabbitmq.conf` 已设 `vm_memory_high_watermark.relative = 0.5` 放宽；加 swap + swappiness=100 |
| 数据初始化失败 | init.sql 未执行 | 删除 mysql 数据卷重新初始化（`docker compose down -v`） |
| watcher 日志持续报 Forbidden / 403 | `k8s-watcher/rbac.yaml` 未部署或 ServiceAccount 名不匹配 | `kubectl apply -f k8s-watcher/rbac.yaml` |
| watcher Pod ImagePullBackOff | 本地镜像未构建 | 先执行 `docker build -t supply-chain-watcher:latest ./k8s-watcher` |
| Pod 超阈值但钉钉收不到 | 关键词不匹配 / 加签错误 / Webhook 留空（DRY-RUN） | 日志搜 `errcode`（310000 = 安全校验失败）；确认 deploy.yaml 已填 URL；机器人关键词需包含「告警」 |

---

## 🆕 9️⃣ K8s Pod 重启告警监视器（可选 · k8s-watcher）

> ⚠️ **与主服务的 Docker Compose 部署相互独立**。主服务跑在单机 Docker 上时本节可完全跳过；
> 只有把项目部署到 Kubernetes（如 Docker Desktop 自带集群）时才有意义。
>
> 功能：实时监听集群内所有 Pod，某个 Pod 的容器累计重启**超过 3 次**（反复崩溃）时，
> 自动调用钉钉群机器人推送告警。配置项、告警行为、详细说明见 README.md「K8s Pod 监视器」章节。

### 9.1 创建钉钉机器人（一次即可）

群设置 → 机器人 → 添加 → **自定义（通过 Webhook 接入）**；
安全设置同时勾选 **自定义关键词**（填 `告警`）和 **加签**（记下 SEC 开头的密钥），
完成后拿到 Webhook 地址。

### 9.2 配置 → 构建 → 部署

```bash
# ① 把 Webhook 和加签密钥填进 k8s-watcher/deploy.yaml：
#    WEBHOOK_URL / DINGTALK_SECRET（留空则是 DRY-RUN，只打日志不发送）

# ② 构建镜像（项目根目录）
docker build -t supply-chain-watcher:latest ./k8s-watcher

# ③ 部署（先授权、再部署，顺序不能反）
kubectl apply -f k8s-watcher/rbac.yaml
kubectl apply -f k8s-watcher/deploy.yaml

# ④ 确认日志出现「已开始监视 pods」
kubectl logs -f deploy/supply-chain-watcher
```

### 9.3 验证与卸载

```bash
# 构造一个故意崩溃的测试 Pod，2~3 分钟后重启数超 3，钉钉群应收到告警
kubectl run crash-test --image=docker.m.daocloud.io/library/busybox:1.36 \
  --restart=Always -- sh -c "echo boom; exit 1"

kubectl get pod crash-test -w          # 盯 RESTARTS 列（Ctrl+C 退出）
kubectl delete pod crash-test --now    # 验证完清理

# 卸载监视器
kubectl delete -f k8s-watcher/deploy.yaml -f k8s-watcher/rbac.yaml
```

> 🔎 遇到问题查 8️⃣ 常见问题排查表末尾新增的 watcher 三行。

---

## 🎉 恭喜！

如果以上步骤全部通过，你的供应链溯源系统已成功部署！

- 前端访问：`http://<服务器公网IP>/supply_chain`
- 管理员账号：`11111111` / `123456`（种子数据）
- API 文档：见项目 README.md

> 📖 更多技术细节（并发优化、Worker Pool、RabbitMQ 集成）请阅读项目源码及注释。
>
> Copyright © 2026 Prey-Isis. All rights reserved.
