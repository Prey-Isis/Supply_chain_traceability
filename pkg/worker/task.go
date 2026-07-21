// ============================================================
// pkg/worker/task.go — 任务类型定义
// ============================================================
// 【什么是 Task？】
//   一个"待办事项"的数据结构。生产者（HTTP Controller）填好字段
//   丢进 Channel，消费者（Worker）从 Channel 取出后执行。
//
// 【为什么用 interface{} 做 Payload？】
//   不同类型的任务携带的数据不同：
//     审计日志 → 需要 Product_Id、操作人、变更前后的值
//     状态同步 → 只需要 Product_Id + 新状态
//     统计更新 → 什么都不需要（自己去数据库统计）
//   用 interface{}（Go 1.18+ 可用 any 代替）可以统一塞进同一个 Channel。
// ============================================================

package worker

import (
	"fmt"
	"time"
)

// TaskType 任务类型常量 —— 避免魔法字符串
type TaskType string

const (
	// TaskAuditLog 写审计日志：记录谁在什么时候对哪个产品做了什么操作
	TaskAuditLog TaskType = "audit_log"

	// TaskStatusSync 状态同步：当产品增加历史记录后，检查是否需要自动流转状态
	TaskStatusSync TaskType = "status_sync"

	// TaskStatsRefresh 统计刷新：更新产品的"历史记录条数"等聚合数据
	TaskStatsRefresh TaskType = "stats_refresh"
)

// Task 一个待执行的任务
type Task struct {
	ID        string    // 任务唯一ID（UUID，用于追踪和日志）
	Type      TaskType  // 任务类型：audit_log / status_sync / stats_refresh
	Payload   any       // 任务携带的数据，具体结构取决于 Type
	CreatedAt time.Time // 任务创建时间
	Retries   int       // 已重试次数（初始为 0）
	MaxRetries int      // 最大重试次数（默认 3）
}

// NewTask 创建任务的工厂函数
// 	Type   - 任务类型
// 	payload - 任务数据
// 返回一个初始化好的 Task
func NewTask(taskType TaskType, payload any) Task {
	return Task{
		ID:         generateTaskID(), // 生成简短唯一ID
		Type:       taskType,
		Payload:    payload,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}
}

// ShouldRetry 判断任务是否应该重试
// 	retries < maxRetries → 可以重试
func (t *Task) ShouldRetry() bool {
	return t.Retries < t.MaxRetries
}

// RetryCount 标记一次重试，Retries + 1
func (t *Task) RetryCount() {
	t.Retries++
}

// String 格式化输出（调试用）
func (t *Task) String() string {
	return fmt.Sprintf("[Task %s] type=%s retries=%d/%d", t.ID, t.Type, t.Retries, t.MaxRetries)
}

// ============================================================
// 不同任务类型的 Payload 结构体
// ============================================================

// AuditLogPayload 审计日志任务的数据
type AuditLogPayload struct {
	ProductID   string // 产品ID
	ProductName string // 产品名称
	Action      string // 操作类型（如 "node_updated"）
	Operator    string // 操作人账号（从 JWT 中提取）
	Detail      string // 操作详情描述
}

// StatusSyncPayload 状态同步任务的数据
type StatusSyncPayload struct {
	ProductID string // 产品ID
	NewStatus string // 建议的新状态（Worker 会判断是否需要）
}

// StatsRefreshPayload 统计刷新任务的数据
// 这个类型不需要额外字段 —— Worker 会全量扫描
type StatsRefreshPayload struct {
	ProductID string // 可选：只刷新某个产品；空字符串 = 全量刷新
}

// ============================================================
// 内部工具函数
// ============================================================

// taskIDCounter 简单自增计数器，用来生成短任务ID
var taskIDCounter int64

// generateTaskID 生成一个简短的唯一任务ID
// 格式: "task-00000001"
// 生产环境建议用 uuid.New().String()
func generateTaskID() string {
	taskIDCounter++
	return fmt.Sprintf("task-%08d", taskIDCounter)
}
