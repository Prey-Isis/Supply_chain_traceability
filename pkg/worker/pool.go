// ============================================================
// pkg/worker/pool.go — Worker Pool（MQ 消费版）
// ============================================================
// 【这是一个什么东西？】
//   一组 goroutine（Worker），每个都从 RabbitMQ 队列里抢任务来执行。
//
// 【★ 和内存版的区别】
//   内存版：任务来源是进程内的 chan Task（服务重启任务全丢）
//   MQ 版： 任务来源是 RabbitMQ 队列（独立进程，服务重启任务还在）
//
//   Worker 的角色不变：抢到任务 → 调 Handler → 成功/失败。
//   变化的是"任务从哪来"和"成功/失败怎么汇报"：
//     内存版：成功=继续下一条，失败=手动塞回 channel
//     MQ 版：  成功=ACK（MQ 删除消息），失败=NACK（MQ 重投或丢弃）
//
// 【架构示意】
//   RabbitMQ ──delivery──→ Worker-1 ──handler──→ 写日志/更新状态
//              ──delivery──→ Worker-2 ──handler──→ ...
//              ──delivery──→ Worker-3 ──handler──→ ...
//   MQ 的 QoS(prefetch=1) 保证：哪个 Worker 闲了，消息就给谁。
// ============================================================

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"main/pkg/mq"
)

// ============================================================
// 自定义错误
// ============================================================

var (
	// ErrPoolClosed Worker Pool 已关闭
	ErrPoolClosed = errors.New("Worker Pool 已关闭")

	// ErrMQNotConnected RabbitMQ 客户端未连接
	ErrMQNotConnected = errors.New("RabbitMQ 客户端未初始化")
)

// ============================================================
// Worker Pool 结构体
// ============================================================

// TaskHandler 任务处理函数的签名
//   ctx  - 上下文（超时控制）
//   task - 从 MQ 反序列化出来的任务（Payload 是 JSON 字节，需要自己解码）
type TaskHandler func(ctx context.Context, task Task) error

// WorkerPool 工人池（MQ 版）
type WorkerPool struct {
	// ----- MQ 连接 -----
	mqClient *mq.Client // RabbitMQ 客户端（任务来源）

	// ----- Worker 管理 -----
	workerCount int            // 工人数量
	wg          sync.WaitGroup // 等所有 Worker 干完活再关门

	// ----- 生命周期管理 -----
	ctx    context.Context    // 传递给每个 Worker 的上下文
	cancel context.CancelFunc // 主动关闭所有 Worker 的信号
	closed bool               // 是否已经关闭

	// ----- 任务路由 -----
	// 每种 TaskType 对应一个处理函数
	handlers map[TaskType]TaskHandler

	// ----- 统计信息 -----
	mu          sync.Mutex
	totalTasks  int64 // 总处理任务数
	failedTasks int64 // 失败任务数
}

// ============================================================
// 创建 & 启动 Worker Pool
// ============================================================

// New 创建一个新的 Worker Pool
//
// 【参数变化】
//   内存版：New(workerCount, queueSize) — queueSize 是 channel 缓冲大小
//   MQ 版：  New(workerCount, mqClient) — mqClient 是 RabbitMQ 客户端
//            不需要 queueSize 了，因为缓冲在 RabbitMQ 服务器上
func New(workerCount int, mqClient *mq.Client) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		mqClient:    mqClient,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[TaskType]TaskHandler),
	}
}

// RegisterHandler 注册一个任务处理函数
func (wp *WorkerPool) RegisterHandler(taskType TaskType, handler TaskHandler) {
	wp.handlers[taskType] = handler
}

// Start 启动所有 Worker（每个 Worker 是一个独立 goroutine，从 MQ 消费）
func (wp *WorkerPool) Start() {
	log.Printf("[WorkerPool] 启动 %d 个 Worker（MQ 消费模式）\n", wp.workerCount)

	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.workerLoop(i + 1)
	}
}

// ============================================================
// Worker 主循环
// ============================================================

// workerLoop 单个 Worker 的主循环
//
// 【★ 核心变化：从 MQ 消费而不是从内存 Channel】
//   内存版：for task := range wp.taskQueue（Go channel）
//   MQ 版：  wp.mqClient.Consume(workerID, handler)（RabbitMQ delivery）
//
//   mq.Consume() 内部也是一个 for-range channel 循环，
//   每条消息来了就调我们传入的 handler 回调。
//   Consume() 返回 error 说明连接断了 → 等重连后重新启动消费。
func (wp *WorkerPool) workerLoop(workerID int) {
	defer wp.wg.Done()

	log.Printf("[Worker-%d] 👷 工人已就绪（等待 MQ 任务）\n", workerID)

	// ★ 从 RabbitMQ 消费消息
	// Consume() 是阻塞式的：内部 for-range delivery channel
	// 每来一条消息 → 调 handleMessage → 返回 error → 连接断开时退出
	for {
		wp.mu.Lock()
		closed := wp.closed
		wp.mu.Unlock()
		if closed {
			break
		}

		err := wp.mqClient.Consume(workerID, func(msg mq.TaskMessage) error {
			return wp.handleMessage(msg)
		})

		// Consume 返回 = 连接断开或出错
		wp.mu.Lock()
		closed = wp.closed
		wp.mu.Unlock()
		if closed {
			break
		}

		log.Printf("[Worker-%d] ⚠️ MQ 消费中断: %v，5 秒后重试...\n", workerID, err)
		time.Sleep(5 * time.Second)
	}

	log.Printf("[Worker-%d] 🛑 工人已下班\n", workerID)
}

// ============================================================
// 消息处理 & 路由
// ============================================================

// handleMessage 处理一条从 MQ 收到的消息
//   1. 把 mq.TaskMessage 转成 worker.Task（类型适配）
//   2. 根据 Task.Type 找到对应的 Handler
//   3. 调用 Handler 执行
//   4. 返回 nil（成功，触发 ACK）或 error（失败，触发重试逻辑）
func (wp *WorkerPool) handleMessage(msg mq.TaskMessage) error {
	// ----- 第 1 步：类型转换 -----
	// mq.TaskMessage 和 worker.Task 字段一致，只是类型不同
	// mq 包不应该 import worker（会循环依赖），所以这里做一层转换
	task := Task{
		ID:         msg.ID,
		Type:       TaskType(msg.Type),
		Payload:    json.RawMessage(msg.Payload),
		CreatedAt:  msg.CreatedAt,
		Retries:    msg.Retries,
		MaxRetries: msg.MaxRetries,
	}

	// ----- 第 2 步：路由到对应 Handler -----
	handler, ok := wp.handlers[task.Type]
	if !ok {
		return fmt.Errorf("未注册的任务类型: %s", task.Type)
	}

	// ----- 第 3 步：执行任务（带 10 秒超时保护）-----
	ctx, cancel := context.WithTimeout(wp.ctx, 10*time.Second)
	defer cancel()

	wp.mu.Lock()
	wp.totalTasks++
	wp.mu.Unlock()

	err := handler(ctx, task)
	if err != nil {
		// 如果已经超过最大重试次数，标记为彻底失败
		if task.Retries >= task.MaxRetries {
			wp.mu.Lock()
			wp.failedTasks++
			wp.mu.Unlock()
		}
		return err
	}

	return nil
}

// ============================================================
// 任务投递（生产者调用）
// ============================================================

// Submit 投递一个任务（发布到 RabbitMQ）
//
// 【和内存版的区别】
//   内存版：task → chan Task（进程内传递）
//   MQ 版：  task → JSON → RabbitMQ Publish（网络传输）
//
// 【失败处理】
//   MQ 连接断开时返回 error，调用方（Controller）记日志降级。
//   任务非关键（审计/统计），丢了不影响业务主流程。
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.Lock()
	closed := wp.closed
	wp.mu.Unlock()

	if closed {
		return ErrPoolClosed
	}

	if wp.mqClient == nil {
		return ErrMQNotConnected
	}

	// 把 worker.Task 转成 mq.TaskMessage
	msg := mq.TaskMessage{
		ID:         task.ID,
		Type:       string(task.Type),
		Payload:    task.Payload,
		CreatedAt:  task.CreatedAt,
		Retries:    task.Retries,
		MaxRetries: task.MaxRetries,
	}

	return wp.mqClient.Publish(msg)
}

// ============================================================
// 优雅关闭
// ============================================================

// Shutdown 优雅关闭 Worker Pool
//   1. 标记为已关闭（拒绝新任务）
//   2. 关闭 MQ 连接（Consumer 循环退出）
//   3. wg.Wait() 等所有 Worker 退出
func (wp *WorkerPool) Shutdown() {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return
	}
	wp.closed = true
	wp.mu.Unlock()

	log.Println("[WorkerPool] 🛑 正在优雅关闭...")
	log.Printf("[WorkerPool]   统计: 总任务 %d, 失败 %d\n", wp.totalTasks, wp.failedTasks)

	// 关闭 MQ 连接 → Consume 循环退出 → Worker 退出
	if wp.mqClient != nil {
		wp.mqClient.Close()
	}

	// 取消所有 Worker 的上下文
	wp.cancel()

	// 等所有 Worker 退出
	wp.wg.Wait()

	log.Println("[WorkerPool] ✅ 关闭完成")
}

// ============================================================
// 统计查询
// ============================================================

// Stats 返回 Worker Pool 的运行统计
func (wp *WorkerPool) Stats() map[string]int64 {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	return map[string]int64{
		"total_tasks":    wp.totalTasks,
		"failed_tasks":   wp.failedTasks,
		"active_workers": int64(wp.workerCount),
	}
}

// ============================================================
// 全局便捷函数
// ============================================================

// DefaultPool 全局 Worker Pool 单例
var DefaultPool *WorkerPool

// SubmitTask 向全局 Worker Pool 投递一个任务
// 供外部包（如 router）调用，无需直接持有 WorkerPool 引用
func SubmitTask(task Task) error {
	if DefaultPool == nil {
		return errors.New("Worker Pool 未初始化")
	}
	return DefaultPool.Submit(task)
}
