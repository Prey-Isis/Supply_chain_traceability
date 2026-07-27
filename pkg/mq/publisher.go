// ============================================================
// pkg/mq/publisher.go — 消息发布器
// ============================================================
// 【这个文件干什么？】
//   把 Go 结构体（Task）序列化成 JSON 字节流，发送到 RabbitMQ。
//   相当于"寄快递"：打包（序列化）→ 贴面单（routing key）→ 交给快递员（Publish）
//
// 【为什么序列化成 JSON？】
//   RabbitMQ 只认识字节流（[]byte），不认识 Go 结构体。
//   JSON 是跨语言通用格式，以后如果有其他语言的消费者也能解析。
//   对比方案：Protobuf 更快但可读性差；Gob 只能 Go 到 Go。
//   教学场景 JSON 最合适——消息内容肉眼可读，管理界面里直接能看到。
//
// 【Published 和 TCP 发送的区别】
//   Publish() 是"发出去就完事"，不管 MQ 收到没有（fire-and-forget）。
//   如果要确认 MQ 收到了，需要开启 Publisher Confirms（事务模式），
//   性能会下降。我们的任务非关键（审计/统计），丢了就丢了，不用确认。
// ============================================================

package mq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ============================================================
// TaskMessage —— 消息在 MQ 中传输的格式
// ============================================================
// 【为什么不直接用 worker.Task？】
//   worker.Task 的 Payload 字段是 any 类型（interface{}），
//   JSON 反序列化时无法自动还原成具体类型。
//   解决方案：发布时把 Payload 先序列化成 json.RawMessage，
//   消费端拿到消息后，由 Handler 自己反序列化自己的 Payload 类型。
//   这叫"延迟解码"——消息格式统一，各 Handler 各解各的。

// TaskMessage 线上传输的消息格式（与 worker.Task 对应，但 Payload 是原始 JSON）
type TaskMessage struct {
	ID         string          `json:"id"`          // 任务唯一ID
	Type       string          `json:"type"`        // 任务类型（audit_log / status_sync / stats_refresh）
	Payload    json.RawMessage `json:"payload"`     // 任务数据（JSON 原始字节，延迟解码）
	CreatedAt  time.Time       `json:"created_at"`  // 任务创建时间
	Retries    int             `json:"retries"`     // 已重试次数
	MaxRetries int             `json:"max_retries"` // 最大重试次数
}

// NewTaskMessage 创建一个待发布的消息
//   taskType - 任务类型字符串
//   payload  - 任意可 JSON 序列化的结构体
//
// 【使用示例】
//   msg, _ := mq.NewTaskMessage("audit_log", worker.AuditLogPayload{...})
//   client.Publish(msg)
func NewTaskMessage(taskType string, payload any) (TaskMessage, error) {
	// 把 Payload 结构体序列化成 JSON 字节
	// json.RawMessage 本质就是 []byte，反序列化时可以再解开
	raw, err := json.Marshal(payload)
	if err != nil {
		return TaskMessage{}, fmt.Errorf("序列化任务 Payload 失败: %w", err)
	}

	return TaskMessage{
		ID:         generateMessageID(),
		Type:       taskType,
		Payload:    raw,
		CreatedAt:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}, nil
}

// ============================================================
// 发布消息
// ============================================================

// Publish 发送一个任务消息到 RabbitMQ（非阻塞，fire-and-forget）
//
// 【发送流程】
//   1. TaskMessage → JSON []byte（序列化）
//   2. 调用 amqp.Channel.Publish() 发送到交换机
//   3. RabbitMQ 根据 routing key 把消息路由到绑定的队列
//
// 【失败场景】
//   - 连接断开 → 返回 error（调用方记日志降级，任务丢弃）
//   - 序列化失败 → 返回 error（代码 bug，不可能发生除非类型不对）
//   - MQ 队列满了 → 不阻塞（非持久化队列内存满时 MQ 会丢旧消息）
func (c *Client) Publish(msg TaskMessage) error {
	// ----- 第 1 步：获取可用的 Channel -----
	ch, err := c.channel()
	if err != nil {
		return fmt.Errorf("发布失败（MQ 连接不可用）: %w", err)
	}

	// ----- 第 2 步：序列化消息为 JSON -----
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// ----- 第 3 步：发布到交换机 -----
	// 参数说明：
	//   exchange   - 交换机名称，消息先到这里
	//   key        - routing key，交换机用它决定消息进哪个队列
	//   mandatory  - false：路由失败时不返回错误（直接丢弃）
	//   immediate  - false：没有消费者时不立即返回（消息排队等待）
	//   msg        - 消息体（AMQP 协议的消息结构）
	err = ch.Publish(
		ExchangeName, // exchange
		RoutingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",  // 告诉消费者这是 JSON
			Body:         body,                // 消息内容（字节流）
			DeliveryMode: amqp.Transient,      // 非持久化（MQ 重启消息消失）
			Timestamp:    time.Now(),          // 发布时间戳
			MessageId:    msg.ID,              // 消息ID（用于追踪）
		},
	)
	if err != nil {
		return fmt.Errorf("发送消息到 RabbitMQ 失败: %w", err)
	}

	log.Printf("[MQ] 📤 任务已发布: id=%s type=%s\n", msg.ID, msg.Type)
	return nil
}

// ============================================================
// 内部工具
// ============================================================

// messageIDCounter 简单自增计数器生成消息ID
var messageIDCounter int64

// generateMessageID 生成简短唯一的消息ID
func generateMessageID() string {
	messageIDCounter++
	return fmt.Sprintf("msg-%d-%08d", time.Now().Unix(), messageIDCounter)
}
