# ============================================================================
# supply-chain-watcher 镜像构建文件（多阶段构建）
# ----------------------------------------------------------------------------
# 构建命令（在项目根目录执行）：
#   docker build -t supply-chain-watcher:latest ./k8s-watcher
#
# 设计目标：镜像极小、启动极快、攻击面极小。
#   最终镜像基于 scratch（空镜像，连 shell 都没有），里面只有两个文件：
#   静态编译的 Go 二进制 + CA 根证书。总大小约 37MB，几乎全是二进制本身
#   （client-go 依赖较多），进程启动毫秒级。
#
# 国内加速：基础镜像走 daocloud 代理、Go 依赖走 goproxy.cn、apk 走阿里云，
#   全程无海外直连。
# ============================================================================

# ---- 构建阶段：编译 Go 程序 ----
FROM docker.m.daocloud.io/library/golang:1.26-alpine AS builder
# alpine 基础镜像不带 CA 证书，装一份，后面复制到运行镜像里
# （钉钉 Webhook 是 https，没有根证书握手会直接失败）
RUN sed -i 's#https://dl-cdn.alpinelinux.org#https://mirrors.aliyun.com#g' /etc/apk/repositories \
 && apk add --no-cache ca-certificates
WORKDIR /build
# CGO_ENABLED=0：纯静态编译，产物不依赖任何系统库，scratch 里也能跑
# GOOS=linux：容器里是 Linux 内核（Windows 宿主机构建也无需额外配置）
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux
# 先只拷贝 go.mod/go.sum 再下载依赖：依赖不变时这一层直接命中缓存，
# 增量构建秒级完成
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# -trimpath 去掉本机路径信息；-w -s 去掉调试符号表，二进制缩小约 30%
RUN --mount=type=cache,target=/root/.cache/go-build go build -trimpath -ldflags="-w -s" -o /out/watcher .

# ---- 运行阶段：scratch = 空镜像 ----
# 没有包管理器、没有 shell、没有任何系统工具：
#   ✔ 攻击面最小（就算被入侵也没有可执行的工具）
#   ✔ 体积最小（除二进制和证书外零额外文件）
#   ✔ 启动最快（毫秒级）
# 代价是没有调试 shell——排查问题只能看 kubectl logs，对守护进程足够。
FROM scratch
# 时区数据不用装：main.go 里 import 了 time/tzdata（Go 1.15+ 特性，
# 时区表直接编进二进制），配合下面的 TZ 环境变量显示北京时间。
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/watcher /watcher
ENV TZ=Asia/Shanghai
ENTRYPOINT ["/watcher"]
