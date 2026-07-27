// ============================================================
// pkg/worker/handlers.go — 任务处理函数
// ============================================================
// 【★ MQ 版的关键变化：Payload 反序列化】
//   内存版：payload, ok := task.Payload.(AuditLogPayload)  ← Go 类型断言
//   MQ 版：  err := task.DecodePayload(&payload)            ← JSON 反序列化
//
//   为什么改？Task 经过 RabbitMQ 传输后，Payload 已经变成了 JSON 字节
//   （json.RawMessage），不再是内存中的 Go 结构体。
//   必须用 json.Unmarshal 还原，不能再用类型断言。
//
// 【为什么 Handler 的业务逻辑不变？】
//   好的分层设计：Worker Pool 负责"任务从哪来"，Handler 只关心"拿到任务做什么"。
//   切换消息中间件（内存 Channel → RabbitMQ → 以后换 Kafka），
//   Handler 层的业务代码一行都不用改。
// ============================================================

package worker

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ============================================================
// 审计日志处理器
// ============================================================
// 【业务逻辑】
//   供应链溯源系统中，每次"创建产品"或"新增供应链历史"时，
//   需要记录一条审计日志：谁、在什么时候、对哪个产品、做了什么操作。
//
// 【为什么异步做？】
//   审计日志和业务主流程无关 —— 用户只关心"创建成功"，不关心日志写没写完。
//   如果审计日志写库要 20ms，同步写就是给每个请求加了 20ms 延迟。
//   丢到后台 Worker 处理，HTTP 响应不感知。
func HandleAuditLog(ctx context.Context, task Task) error {
	// ★ 反序列化：把 JSON 字节还原成 AuditLogPayload 结构体
	// 和内存版的 task.Payload.(AuditLogPayload) 不同，
	// MQ 版必须经过 json.Unmarshal，因为数据是从网络来的
	var payload AuditLogPayload
	if err := task.DecodePayload(&payload); err != nil {
		return fmt.Errorf("审计日志任务 Payload 反序列化失败: %w", err)
	}

	// 模拟写审计日志（实际项目应该写 MySQL audit_log 表）
	log.Printf("[审计日志] 产品=%s(%s) 操作=%s 操作人=%s 详情=%s 时间=%s",
		payload.ProductID,
		payload.ProductName,
		payload.Action,
		payload.Operator,
		payload.Detail,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// 模拟耗时操作：审计日志写库平均 15ms
	select {
	case <-time.After(15 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// ============================================================
// 状态同步处理器
// ============================================================
func HandleStatusSync(ctx context.Context, task Task) error {
	var payload StatusSyncPayload
	if err := task.DecodePayload(&payload); err != nil {
		return fmt.Errorf("状态同步任务 Payload 反序列化失败: %w", err)
	}

	log.Printf("[状态同步] 产品=%s → 自动同步状态（建议新状态=%s）", payload.ProductID, payload.NewStatus)

	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// ============================================================
// 统计刷新处理器
// ============================================================
func HandleStatsRefresh(ctx context.Context, task Task) error {
	var payload StatsRefreshPayload
	if err := task.DecodePayload(&payload); err != nil {
		return fmt.Errorf("统计刷新任务 Payload 反序列化失败: %w", err)
	}

	scope := "全量"
	if payload.ProductID != "" {
		scope = "产品=" + payload.ProductID
	}
	log.Printf("[统计刷新] %s → 刷新供应链历史计数", scope)

	select {
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
