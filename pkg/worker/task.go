// ============================================================
// pkg/worker/task.go — 任务类型定义
// ============================================================
// 【什么是 Task？】
//   一个"待办事项"的数据结构。生产者（HTTP Controller）填好字段
//   发布到 RabbitMQ，消费者（Worker）从 MQ 取出后执行。
//
// 【★ MQ 版的关键变化：Payload 从 any 变成 json.RawMessage】
//   内存版：Task 在 Go 进程内传递，Payload 可以是任何 Go 类型（any）
//   MQ 版： Task 要序列化成 JSON 经过网络传输，Payload 必须是 JSON 字节
//
//   解决方案：发布时把具体 Payload（如 AuditLogPayload）序列化成
//   json.RawMessage（本质就是 []byte），消费端 Handler 自己解开。
//   这叫"延迟解码"——传输格式统一，各 Handler 各解各的。
//
// 【类型对应关系】
//   发布:  worker.NewTask(TaskAuditLog, AuditLogPayload{...})
//          → Task.Payload = {"ProductID":"P001","Action":"created",...}（JSON 字节）
//   消费:  HandleAuditLog 里 json.Unmarshal(Task.Payload, &AuditLogPayload{})
//          → 还原成 AuditLogPayload 结构体使用
// ============================================================

package worker

import (
	"encoding/json"
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

// Task 一个待执行的任务（可 JSON 序列化，通过 RabbitMQ 传输）
type Task struct {
	ID         string          `json:"id"`          // 任务唯一ID
	Type       TaskType        `json:"type"`        // 任务类型
	Payload    json.RawMessage `json:"payload"`     // ★ JSON 原始字节（延迟解码）
	CreatedAt  time.Time       `json:"created_at"`  // 任务创建时间
	Retries    int             `json:"retries"`     // 已重试次数
	MaxRetries int             `json:"max_retries"` // 最大重试次数
}

// NewTask 创建任务的工厂函数
//   taskType - 任务类型
//   payload  - 任务数据（任意可 JSON 序列化的结构体）
//
// 【内部做了什么？】
//   把 payload 结构体序列化成 JSON 字节，存到 Task.Payload 里。
//   之后 Task 整体再序列化一次发到 MQ（双层序列化）。
//   为什么不直接把 payload 嵌进去？因为反序列化时 Go 不知道 Payload 的具体类型。
func NewTask(taskType TaskType, payload any) Task {
	// 序列化具体 Payload 为 JSON 字节
	raw, _ := json.Marshal(payload)

	return Task{
		ID:         generateTaskID(),
		Type:       taskType,
		Payload:    raw,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}
}

// DecodePayload 把 Payload 反序列化成具体类型
//   target - 目标结构体的指针（如 &AuditLogPayload{}）
//
// 【使用示例】
//   var p AuditLogPayload
//   err := task.DecodePayload(&p)
func (t *Task) DecodePayload(target any) error {
	return json.Unmarshal(t.Payload, target)
}

// ShouldRetry 判断任务是否应该重试
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
	ProductID   string `json:"product_id"`   // 产品ID
	ProductName string `json:"product_name"` // 产品名称
	Action      string `json:"action"`       // 操作类型
	Operator    string `json:"operator"`     // 操作人账号
	Detail      string `json:"detail"`       // 操作详情描述
}

// StatusSyncPayload 状态同步任务的数据
type StatusSyncPayload struct {
	ProductID string `json:"product_id"` // 产品ID
	NewStatus string `json:"new_status"` // 建议的新状态
}

// StatsRefreshPayload 统计刷新任务的数据
type StatsRefreshPayload struct {
	ProductID string `json:"product_id"` // 可选：只刷新某个产品
}

// ============================================================
// 内部工具函数
// ============================================================

// taskIDCounter 简单自增计数器，用来生成短任务ID
var taskIDCounter int64

// generateTaskID 生成一个简短的唯一任务ID
func generateTaskID() string {
	taskIDCounter++
	return fmt.Sprintf("task-%d-%08d", time.Now().Unix(), taskIDCounter)
}
