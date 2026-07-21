// ============================================================
// pkg/worker/pool.go — Worker Pool 核心引擎
// ============================================================
// 【这是一个什么东西？】
//   一个"任务分发中心"——你丢任务进去，它分配给闲置的工人（goroutine）去干活。
//
// 【和工作求职平台的类比】
//   - Task        → 工作订单（谁发的、干什么、带什么材料）
//   - chan Task   → 工作公告栏（发单人把订单贴上去）
//   - Worker      → 工人（守在公告栏旁边，有活就抢）
//   - Worker Pool → 工头（管理 3 个工人，开张/关门）
//
// 【核心设计决策】
//   1. 为什么用"带缓冲的 Channel"而不是"无缓冲的"？
//      缓冲 Channel 允许生产者在 Worker 忙的时候把任务"暂存"在管道里，
//      避免 HTTP 请求被阻塞。无缓冲 Channel 会强制生产者等消费者就绪。
//
//   2. 为什么用 select + default 做非阻塞投递？
//      如果队列满了（1000 个任务都在排队），Submit() 不会卡死 HTTP 请求，
//      而是立刻返回错误，让调用方决定降级策略（记录日志/丢弃/返回提示）。
//
//   3. 为什么要优雅关闭？
//      服务重启时，如果队列里还有未处理的任务，直接杀进程会丢数据。
//      优雅关闭 = 不再接受新任务 → 等 Worker 把队列清空 → 再退出。
// ============================================================

package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// ============================================================
// 自定义错误
// ============================================================

var (
	// ErrQueueFull 队列已满，任务被拒绝
	ErrQueueFull = errors.New("任务队列已满，任务被拒绝")

	// ErrPoolClosed Worker Pool 已关闭，不再接受新任务
	ErrPoolClosed = errors.New("Worker Pool 已关闭")
)

// ============================================================
// Worker Pool 结构体
// ============================================================

// TaskHandler 任务处理函数的签名
// 	Worker 拿到 Task 后，根据 Task.Type 路由到不同的 Handler
type TaskHandler func(ctx context.Context, task Task) error

// WorkerPool 工人池
type WorkerPool struct {
	// ----- 任务队列 -----
	// ★ 这就是你的"消息队列"核心 —— 一个带缓冲的 Go Channel
	//   make(chan Task, queueSize)
	//   缓冲区大小 = queueSize，超过这个数，Submit() 会返回 ErrQueueFull
	taskQueue chan Task

	// ----- Worker 管理 -----
	workerCount int             // 工人数量（启动多少个 goroutine）
	wg          sync.WaitGroup  // 等所有 Worker 干完活再关门

	// ----- 生命周期管理 -----
	ctx    context.Context    // 传递给每个 Worker 的上下文（用于超时/取消）
	cancel context.CancelFunc // 主动关闭所有 Worker 的信号
	closed bool               // 是否已经关闭

	// ----- 任务路由 -----
	// 每种 TaskType 对应一个处理函数
	// 相当于：Worker 拿到一个 Task，看一眼 Type，去 map 里找到对的 Handler
	handlers map[TaskType]TaskHandler

	// ----- 统计信息 -----
	mu         sync.Mutex // 保护下面几个统计字段的并发安全
	totalTasks int64      // 总投递任务数
	failedTasks int64     // 失败任务数（超过重试次数的）
}

// ============================================================
// 创建 & 启动 Worker Pool
// ============================================================

// New 创建一个新的 Worker Pool（不立即启动，调用 Start() 才开工）
//
// 参数:
//   workerCount - 工人数量（建议 3~5，太少不够用，太多吃内存）
//   queueSize   - 任务队列缓冲大小（建议 500~2000，取决于任务产生速度）
//
// 返回一个已初始化但未启动的 WorkerPool
func New(workerCount, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		taskQueue:   make(chan Task, queueSize), // ★ 核心: 带缓冲的 Channel = 消息队列
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[TaskType]TaskHandler),
	}
}

// RegisterHandler 注册一个任务处理函数
// 	Worker 从队列取出 Task 后，根据 Task.Type 找到对应的 Handler 执行
func (wp *WorkerPool) RegisterHandler(taskType TaskType, handler TaskHandler) {
	wp.handlers[taskType] = handler
}

// Start 启动所有 Worker（每个 Worker 是一个独立的 goroutine）
// 必须在 RegisterHandler 之后调用
func (wp *WorkerPool) Start() {
	log.Printf("[WorkerPool] 启动 %d 个 Worker，队列容量 %d\n", wp.workerCount, cap(wp.taskQueue))

	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1) // 每启动一个 Worker，计数器 +1
		go wp.workerLoop(i + 1) // 启动 goroutine，编号从 1 开始
	}
}

// ============================================================
// Worker 主循环
// ============================================================

// workerLoop 单个 Worker 的无限循环
// 	"坐在 Channel 旁边，有活就干，没活就等"
//
// 【循环条件】
//   用 for-range 读取 Channel：
//     - Channel 有数据 → 取出来，执行 handleTask
//     - Channel 关闭且为空 → 自动退出循环
//     比 for { select {} } 写法更简洁
func (wp *WorkerPool) workerLoop(workerID int) {
	// defer 保证 Worker 退出时计数器 -1
	defer wp.wg.Done()

	log.Printf("[Worker-%d] 👷 工人已就绪\n", workerID)

	// ★ for-range Channel：有任务就处理，Channel 关闭后自动退出
	for task := range wp.taskQueue {
		log.Printf("[Worker-%d] 📥 领取任务: %s\n", workerID, task.String())

		// 执行任务
		if err := wp.handleTask(task); err != nil {
			log.Printf("[Worker-%d] ❌ 任务失败: %s, 错误: %v\n", workerID, task.String(), err)

			// 失败重试：把任务重新放回队列
			if task.ShouldRetry() {
				task.RetryCount()
				log.Printf("[Worker-%d] 🔄 重试任务: %s\n", workerID, task.String())

				// 用 goroutine 异步放回队列，避免死锁
				// 如果队列满了，说明系统压力太大，丢弃并记录
				go func(t Task) {
					select {
					case wp.taskQueue <- t:
						// 成功放回
					default:
						// 队列满了，任务被丢弃
						wp.mu.Lock()
						wp.failedTasks++
						wp.mu.Unlock()
						log.Printf("[Worker-%d] 💀 任务丢弃（队列已满）: %s\n", workerID, t.String())
					}
				}(task)
			} else {
				// 重试次数用完，标记失败
				wp.mu.Lock()
				wp.failedTasks++
				wp.mu.Unlock()
				log.Printf("[Worker-%d] 💀 任务彻底失败（已达最大重试次数）: %s\n", workerID, task.String())
			}
		} else {
			log.Printf("[Worker-%d] ✅ 任务完成: %s\n", workerID, task.String())
		}
	}

	log.Printf("[Worker-%d] 🛑 工人已下班\n", workerID)
}

// ============================================================
// 任务执行 & 路由
// ============================================================

// handleTask 执行单个任务
// 	1. 根据 Task.Type 找到对应的 Handler
// 	2. 如果没有注册 Handler，返回错误
// 	3. 调用 Handler 执行
func (wp *WorkerPool) handleTask(task Task) error {
	handler, ok := wp.handlers[task.Type]
	if !ok {
		return fmt.Errorf("未注册的任务类型: %s", task.Type)
	}

	// 执行任务，带上下文的超时保护（10 秒）
	ctx, cancel := context.WithTimeout(wp.ctx, 10*time.Second)
	defer cancel()

	return handler(ctx, task)
}

// ============================================================
// 任务投递（生产者调用）
// ============================================================

// Submit 投递一个任务到队列（非阻塞）
//
// 【为什么非阻塞？】
//   HTTP Controller 调 Submit() 时，如果 Worker 忙不过来、队列满了，
//   不能让 HTTP 请求卡在这里等。用 select + default 实现：
//     case 放入成功 → 返回 nil（任务已入队）
//     default       → 返回 ErrQueueFull（调用方自行降级）
//
// 【调用示例】
//   err := workerPool.Submit(NewTask(TaskAuditLog, payload))
//   if err != nil {
//       log.Println("任务投递失败，降级处理:", err)
//   }
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return ErrPoolClosed
	}

	select {
	case wp.taskQueue <- task:
		// ★ 成功放入 Channel —— 任务已入队，Worker 会处理
		wp.totalTasks++
		return nil
	default:
		// ★ 队列满了 —— 立即返回，不阻塞
		return ErrQueueFull
	}
}

// ============================================================
// 优雅关闭
// ============================================================

// Shutdown 优雅关闭 Worker Pool
// 	1. 标记为已关闭（拒绝新任务）
// 	2. close(taskQueue) —— 通知所有 Worker "没新活了，干完手头的就下班"
// 	3. wg.Wait() —— 等所有 Worker 处理完队列里的剩余任务
func (wp *WorkerPool) Shutdown() {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return
	}
	wp.closed = true
	wp.mu.Unlock()

	log.Println("[WorkerPool] 🛑 正在优雅关闭...")
	log.Printf("[WorkerPool]   统计: 总任务 %d, 失败 %d, 队列剩余 %d\n",
		wp.totalTasks, wp.failedTasks, len(wp.taskQueue))

	// 关闭 Channel → Worker 的 for-range 会退出
	close(wp.taskQueue)

	// 等所有 Worker 处理完剩余任务
	wp.wg.Wait()

	log.Println("[WorkerPool] ✅ 关闭完成")
}

// Stats 返回 Worker Pool 的运行统计
func (wp *WorkerPool) Stats() map[string]int64 {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	return map[string]int64{
		"total_tasks":   wp.totalTasks,
		"failed_tasks":  wp.failedTasks,
		"queue_pending": int64(len(wp.taskQueue)),
		"queue_capacity": int64(cap(wp.taskQueue)),
		"active_workers": int64(wp.workerCount),
	}
}

// ============================================================
// 全局便捷函数
// ============================================================

// 全局 Worker Pool 单例
var DefaultPool *WorkerPool

// SubmitTask 向全局 Worker Pool 投递一个任务（非阻塞）
// 供外部包（如 router）调用，无需直接持有 WorkerPool 引用
func SubmitTask(task Task) error {
	if DefaultPool == nil {
		return errors.New("Worker Pool 未初始化")
	}
	return DefaultPool.Submit(task)
}
