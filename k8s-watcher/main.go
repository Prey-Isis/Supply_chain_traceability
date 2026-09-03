// ============================================================================
// 供应链溯源系统 · K8s Pod 重启告警监视器（supply-chain-watcher）
// ----------------------------------------------------------------------------
// 这是什么？
//   一个运行在 Kubernetes（K8s）集群里的"哨兵"程序。它盯着集群里的 Pod
//   （Pod = K8s 中运行容器的最小单元），一旦某个 Pod 的容器累计重启次数
//   超过阈值（默认 3 次，说明大概率在反复崩溃，即 CrashLoopBackOff 状态），
//   就调用钉钉群机器人发送告警消息。
//
// 工作流程（从启动到告警）：
//   1. 读取环境变量配置（main → loadConfig）
//   2. 向 K8s API 服务器认证并建立连接（newClientset）
//   3. 通过 Informer 建立 Pod 的持续监听：
//      - 第一次会"全量拉取"集群所有 Pod（List）
//      - 之后保持长连接，任何 Pod 的变化实时推送过来（Watch）
//      - 每 60 秒做一次本地缓存全量重放（resync），兜底补漏
//   4. 每次 Pod 新建/更新都会回调 check()：
//      算重启次数 → 判断是否超阈值 → 判断是否需要去重 → 发钉钉
//   5. Pod 被删除时回调 remove()，清理该 Pod 的告警状态
//
// 代码地图：
//   main.go  —— 配置、认证、监听、去重判断（本文件）
//   alert.go —— 钉钉消息的实际发送（HTTP 细节）
//
// 部署与配置说明见仓库 README.md 的「K8s Pod 监视器」章节。
// ============================================================================
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	// 内嵌时区数据：运行镜像用的是 scratch（空镜像，不带任何系统文件），
	// 没有这个包的话 Go 只能显示 UTC 时间；配合镜像里的 TZ=Asia/Shanghai
	// 环境变量，日志和告警里才能显示北京时间。
	_ "time/tzdata"

	corev1 "k8s.io/api/core/v1"               // K8s 官方类型定义（Pod、PodPhase 等）
	"k8s.io/client-go/informers"              // Informer 机制：List-Watch + 本地缓存
	"k8s.io/client-go/kubernetes"             // K8s API 客户端（相当于"遥控器"）
	"k8s.io/client-go/rest"                   // REST 连接配置（含集群内认证）
	"k8s.io/client-go/tools/cache"            // 缓存工具（事件回调、key 计算）
	"k8s.io/client-go/tools/clientcmd"        // kubeconfig 文件加载（集群外调试用）
)

// resync 周期 60 秒，它有两个作用：
//  1. 周期性把缓存里所有 Pod 重新回调一遍 check()，弥补错过的实时事件
//  2. 兼作"发送失败后的重试定时器"——发送失败不记录去重状态，
//     下个 resync 周期同一 Pod 再次进 check() 时会重新尝试发送
const resyncInterval = time.Minute

// config 保存程序的全部配置，来源全部是环境变量（见 loadConfig）
type config struct {
	WebhookURL string // 钉钉机器人 Webhook 地址；为空 = DRY-RUN 只打日志不发送
	Secret     string // 钉钉加签密钥（SEC 开头）；机器人开了"加签"才需要
	Threshold  int32  // 重启次数阈值，超过它才告警（默认 3）
	Namespace  string // 监听的命名空间；空字符串 = 监听全部命名空间
}

// loadConfig 从环境变量读配置。整个程序不依赖配置文件，方便容器化部署。
func loadConfig() config {
	c := config{
		WebhookURL: os.Getenv("WEBHOOK_URL"),
		Secret:     os.Getenv("DINGTALK_SECRET"),
		Namespace:  os.Getenv("NAMESPACE"),
		Threshold:  3,
	}
	if v := os.Getenv("RESTART_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			c.Threshold = int32(n)
		} else {
			log.Printf("[config] RESTART_THRESHOLD=%q 无效，回退默认 3", v)
		}
	}
	return c
}

// newClientset 创建 K8s API 客户端，分两种场景：
//
//  1. 集群内运行（部署后的正常情况）：
//     K8s 会自动给每个 Pod 注入认证信息（环境变量 + ServiceAccount 令牌
//     文件），rest.InClusterConfig() 一行就能拿到。这就是为什么部署清单
//     （deploy.yaml）里要指定 serviceAccountName——没身份会被拒绝（403）。
//
//  2. 集群外运行（本地 go run 调试）：
//     InClusterConfig 会报错，此时回退读取 kubeconfig 文件
//     （kubectl 用的同一份配置，默认 ~/.kube/config，也可用 $KUBECONFIG
//     环境变量指定路径）。这样在开发机上就能直接连 Docker Desktop 的集群调试。
func newClientset() (*kubernetes.Clientset, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("[auth] 不在集群内(%v)，回退 kubeconfig（本地调试模式）", err)
		restCfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(), nil).ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(restCfg)
}

// handler 承载核心业务：收到 Pod 事件后判断要不要告警。
//
// state 是去重状态表：
//   - key   = "命名空间/Pod名"
//   - value = 上次【成功发出告警】时的重启总次数
//
// 为什么需要它？Informer 每 60 秒 resync 会把所有 Pod 重新回调一遍，
// 如果不去重，同一个崩溃的 Pod 每分钟都会刷一条告警。
//
// 为什么"成功才写入"？如果钉钉发送失败（网络抖动等）也写入状态，
// 这个 Pod 就永远不会再告警了。只记成功 → 失败后下个 resync 自动重试。
//
// mu 读写锁：Informer 的事件回调本身是串行的，理论上不加锁也安全，
// 但加一把锁的成本几乎为零，能避免以后加新 goroutine 时踩数据竞争的坑。
type handler struct {
	cfg   config
	mu    sync.Mutex
	state map[string]int32
}

// check 是 Add（新建 Pod）和 Update（Pod 状态变化 + resync 重放）事件的
// 统一入口，也是整个监视器的核心判断逻辑。
func (h *handler) check(obj interface{}) {
	// 类型断言：Informer 回调给的是 interface{}，断言成 *corev1.Pod。
	// 必须用带 ok 的写法——万一断言失败 panic 会带崩整个进程。
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}
	// Job/批处理跑完的 Pod（Succeeded/Failed 且不再重启）不告警，纯噪声
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return
	}
	key := pod.Namespace + "/" + pod.Name
	total := sumRestarts(pod)

	h.mu.Lock()
	// 防御性检查：正常情况下重启计数只增不减；如果 state 里记录的值比
	// 当前还大，说明"同名 Pod 被删后重建、但删除事件丢了"，此时清掉旧
	// 记录把它当新 Pod 对待，避免计数永久错乱。
	if last, ok := h.state[key]; ok && last > total {
		delete(h.state, key)
	}
	last, alerted := h.state[key]
	h.mu.Unlock()

	// 未超阈值：什么都不做（判断是"超过"即 total > 3 才告警）
	if total <= h.cfg.Threshold {
		return
	}
	// resync 防抖的关键判断：已经告过警，且从上次告警到现在重启数没有
	// 再涨满一个阈值（比如上次报 4 次，现在 5 次，5-4=1 < 3），就保持
	// 静默。这样持续崩溃的 Pod 每多崩 3 次才报一次（4、7、10…），
	// 既不会每分钟刷屏，也不会让长期故障只报一声就永远沉默。
	if alerted && total-last < h.cfg.Threshold {
		return
	}

	// 组装消息并发送。发送失败（返回 err）时故意不写 state，
	// 下个 resync 周期条件重新满足会自动重试，无需额外的重试代码。
	if err := sendDingTalk(h.cfg, buildMessage(pod, total, h.cfg.Threshold)); err != nil {
		log.Printf("[alert] 发送失败 %s: %v（下个 resync 周期自动重试）", key, err)
		return
	}
	h.mu.Lock()
	h.state[key] = total
	h.mu.Unlock()
	log.Printf("[watcher] 已告警 %s restarts=%d", key, total)
}

// remove 处理 Pod 删除事件：把它的告警状态从表里清掉。
// 不清理的话 map 会随时间无限膨胀（内存泄漏）。
func (h *handler) remove(obj interface{}) {
	// Watch 断线重连期间可能漏掉删除事件，Informer 会补发一个
	// "墓碑对象"（DeletedFinalStateUnknown）而不是 Pod 本身。
	// DeletionHandlingMetaNamespaceKeyFunc 两种情况都能正确取到 key。
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	h.mu.Lock()
	delete(h.state, key)
	h.mu.Unlock()
}

// sumRestarts 统计 Pod 的总重启次数。
// 只对 ContainerStatuses（业务容器）求和，与 kubectl get pods 显示的
// RESTARTS 列口径完全一致；init 容器的重试属于"启动过程"，不计入，
// 避免数字虚高导致误告警。Pod 刚创建时该列表可能为空，range 天然安全。
func sumRestarts(pod *corev1.Pod) int32 {
	var n int32
	for _, cs := range pod.Status.ContainerStatuses {
		n += cs.RestartCount
	}
	return n
}

// buildMessage 组装钉钉消息文本，把排障需要的关键信息都带上：
// Pod 名（含命名空间）、重启次数、当前阶段、所在节点、每个容器
// 上次崩溃的原因和退出码（Error=崩溃退出，OOMKilled=内存爆了）。
func buildMessage(pod *corev1.Pod, total, threshold int32) string {
	msg := fmt.Sprintf("【K8s 告警】Pod %s/%s 已重启 %d 次（阈值 %d）\n阶段: %s  节点: %s",
		pod.Namespace, pod.Name, total, threshold, pod.Status.Phase, pod.Spec.NodeName)
	for _, cs := range pod.Status.ContainerStatuses {
		if lt := cs.LastTerminationState.Terminated; lt != nil {
			msg += fmt.Sprintf("\n容器 %s: 上次退出原因=%s 退出码=%d", cs.Name, lt.Reason, lt.ExitCode)
		}
	}
	return msg
}

func main() {
	cfg := loadConfig()
	clientset, err := newClientset()
	if err != nil {
		log.Fatalf("[auth] 无法创建 kubernetes client: %v", err)
	}

	h := &handler{cfg: cfg, state: map[string]int32{}}

	// NewSharedInformerFactory 是 client-go 的"监听工厂"：
	//   参数1 clientset    —— 用哪个 API 客户端
	//   参数2 resync 周期  —— 每 60s 全量重放一遍缓存（见文件头说明）
	//   WithNamespace      —— 只监听指定命名空间；传空串 = 全部
	// 相比每次都调 API 轮询，Informer 用本地缓存 + 长连接推送，
// 既实时又几乎不给 API 服务器造成压力，是官方推荐的标准姿势。
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, resyncInterval,
		informers.WithNamespace(cfg.Namespace))

	// 取出 Pod 专用的 Informer 并注册三类事件的回调：
	//   AddFunc    新建 Pod（含启动时的全量 List，每个存量 Pod 都会回调一次，
	//              所以程序一启动就能对"已经在崩溃"的 Pod 立即告警）
	//   UpdateFunc Pod 状态变化（重启数 +1、阶段切换等）+ resync 重放
	//   DeleteFunc Pod 被删除 → 清理状态
	podInformer := factory.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    h.check,
		UpdateFunc: func(_, cur interface{}) { h.check(cur) },
		DeleteFunc: h.remove,
	})

	// Start 是非阻塞的：启动后台 goroutine 去 List + Watch。
	// WaitForCacheSync 等第一次全量同步完成，之后 check() 看到的数据才完整。
	stop := make(chan struct{})
	factory.Start(stop)
	if !cache.WaitForCacheSync(stop, podInformer.HasSynced) {
		log.Fatal("[watcher] cache sync 失败")
	}
	webhookState := "未配置(DRY-RUN)"
	if cfg.WebhookURL != "" {
		webhookState = "已配置"
	}
	log.Printf("[watcher] 已开始监视 pods namespace=%q threshold=%d resync=%v webhook=%s",
		cfg.Namespace, cfg.Threshold, resyncInterval, webhookState)

	// 阻塞等待 SIGINT（Ctrl+C）/ SIGTERM（kubectl delete 等发出的终止信号），
	// 收到后关闭 stop 通道，让所有后台监听优雅退出。
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	close(stop)
	log.Println("[watcher] 收到退出信号，已停止")
}
