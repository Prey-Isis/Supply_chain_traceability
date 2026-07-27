// ============================================================
// pkg/mq/consumer.go — 消息消费者
// ============================================================
// 【这个文件干什么？】
//   从 RabbitMQ 队列里取消息，反序列化成 TaskMessage，交给回调函数处理。
//   相当于"取快递"：去快递柜（Queue）→ 取包裹（Delivery）→ 拆包（反序列化）→ 签收（ACK）
//
// 【核心概念：ACK / NACK —— 消息确认机制】
//   RabbitMQ 把消息发给消费者后，不会立即删除，而是等消费者"确认"：
//
//   ACK（Acknowledgement）= "我处理完了，可以删了"
//     → RabbitMQ 把消息从队列中删除
//
//   NACK（Negative ACK）= "我处理失败了"
//     → requeue=true:  消息放回队列，等下一个消费者重试
//     → requeue=false: 消息直接丢弃（或进死信队列，如果配置了的话）
//
//   如果消费者一直没 ACK 也没 NACK（比如服务崩溃了）：
//     → RabbitMQ 检测到连接断开，自动把消息 requeue，不丢！
//     → 这就是 MQ 比内存 Channel 可靠的核心原因
//
// 【为什么用手动 ACK 而不是自动 ACK？】
//   自动 ACK（autoAck=true）：MQ 发出消息就当成功，马上删除。
//     风险：消费者拿到消息还没处理就崩了 → 消息永远丢失。
//   手动 ACK（autoAck=false）：处理完再 ACK。
//     安全：处理失败可以 NACK 重试，崩溃时消息自动回队列。
//   结论：生产环境必须用手动 ACK！
// ============================================================

package mq

import (
	"encoding/json"
	"fmt"
	"log"
)

// ============================================================
// 消息处理回调
// ============================================================

// MessageHandler 消费到一条消息后的处理函数
//   返回值:
//     nil    → 处理成功，发送 ACK
//     error  → 处理失败，根据重试策略发送 NACK
type MessageHandler func(msg TaskMessage) error

// ============================================================
// 开始消费
// ============================================================

// Consume 开始从队列消费消息（阻塞式，建议在 goroutine 中调用）
//
// 【参数】
//   workerID - 消费者编号（用于日志区分，比如 "Worker-1"）
//   handler  - 每条消息的处理函数
//
// 【工作流程】
//   1. 向 RabbitMQ 注册为队列消费者
//   2. RabbitMQ 把队列里的消息推送到 deliveries channel
//   3. for-range 循环：每来一条消息 → 反序列化 → 调 handler → ACK/NACK
//   4. deliveries channel 关闭 → 说明连接断了 → 退出循环（重连逻辑会重建）
//
// 【QoS（Quality of Service）—— 流量控制】
//   prefetchCount=1 表示 RabbitMQ 一次只给这个消费者发 1 条消息，
//   处理完 ACK 之后再发下一条。
//   如果 prefetchCount=10，MQ 会一口气发 10 条过来等着慢慢处理。
//   设 1 的好处：多个 Worker 之间负载均衡（谁闲谁拿），不会一个 Worker 囤积。
func (c *Client) Consume(workerID int, handler MessageHandler) error {
	ch, err := c.channel()
	if err != nil {
		return fmt.Errorf("消费启动失败（MQ 连接不可用）: %w", err)
	}

	// ----- 设置 QoS：每次只预取 1 条消息 -----
	// 这是 Worker Pool 负载均衡的关键！
	// 3 个 Worker 都设 prefetch=1，RabbitMQ 会轮询分发：
	//   Worker-1 拿到 task-1 → Worker-2 拿到 task-2 → Worker-3 拿到 task-3 → Worker-1 拿到 task-4 ...
	err = ch.Qos(
		1,     // prefetchCount: 最多预取 1 条
		0,     // prefetchSize:  不限制消息大小（0 = 无限制）
		false, // global:        false = 只对当前 Channel 生效
	)
	if err != nil {
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	// ----- 注册为消费者，开始接收消息 -----
	// 参数说明：
	//   queue     - 队列名
	//   consumer  - 消费者标签（调试用，空字符串让 MQ 自动生成）
	//   autoAck   - false：手动 ACK（关键！处理完才确认）
	//   exclusive - false：不排他（多个 Worker 共享队列）
	//   noLocal   - false：允许接收自己发布的消息（无所谓）
	//   noWait    - false：等待服务器确认
	//   args      - nil
	deliveries, err := ch.Consume(
		QueueName, // queue
		"",        // consumer tag
		false,     // autoAck = false → 手动确认！
		false,     // exclusive
		false,     // noLocal
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("注册消费者失败: %w", err)
	}

	log.Printf("[MQ] 👂 Worker-%d 开始监听队列: %s\n", workerID, QueueName)

	// ----- 消费循环：从 deliveries channel 读消息 -----
	// deliveries 是一个 Go Channel，RabbitMQ 来一条消息就往里面塞一条
	// 连接断开时这个 channel 会被关闭，for-range 自动退出
	for delivery := range deliveries {
		log.Printf("[MQ] 📥 Worker-%d 收到消息: %s\n", workerID, delivery.MessageId)

		// ----- 反序列化：JSON 字节 → TaskMessage 结构体 -----
		var msg TaskMessage
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			// 消息格式不对，是脏数据，直接拒绝不重试（requeue=false 丢弃）
			// 这种消息重试一万次也是失败的，没必要放回队列
			log.Printf("[MQ] 💀 Worker-%d 消息反序列化失败: %v，丢弃\n", workerID, err)
			delivery.Nack(false, false) // multiple=false, requeue=false
			continue
		}

		// ----- 调用业务处理函数 -----
		handleErr := handler(msg)

		if handleErr == nil {
			// ★ 处理成功 → ACK：告诉 RabbitMQ "删了吧"
			//   multiple=false：只确认这一条消息
			//   multiple=true：确认这一条及之前所有未确认的消息（批量确认，我们不用）
			if err := delivery.Ack(false); err != nil {
				log.Printf("[MQ] ⚠️ Worker-%d ACK 失败: %v\n", workerID, err)
			}
			log.Printf("[MQ] ✅ Worker-%d 任务完成: id=%s type=%s\n", workerID, msg.ID, msg.Type)
		} else {
			// ★ 处理失败 → 根据重试次数决定 NACK 策略
			if msg.Retries < msg.MaxRetries {
				// 还没到最大重试次数 → 重新发布（Retries+1），原消息 ACK 掉
				//
				// 【为什么不直接 NACK requeue？】
				//   NACK requeue=true 会让消息回到队列头部，立即被重新消费。
				//   如果失败原因是"数据库挂了"，立即重试还是会失败，形成死循环。
				//   改成"重新发布一条 Retries+1 的新消息"：
				//     - 新消息排到队列尾部，给其他消息让路
				//     - 重试次数在消息体里可见，达到上限就停止
				//     - 原消息 ACK 掉，不会形成重复
				msg.Retries++
				log.Printf("[MQ] 🔄 Worker-%d 任务失败（%v），重新发布（第 %d/%d 次重试）\n",
					workerID, handleErr, msg.Retries, msg.MaxRetries)

				if pubErr := c.Publish(msg); pubErr != nil {
					log.Printf("[MQ] ❌ 重新发布失败: %v\n", pubErr)
				}
				delivery.Ack(false) // 原消息确认删除
			} else {
				// 超过最大重试次数 → NACK requeue=false，放弃这条消息
				// 生产环境这里应该进死信队列（DLX），我们简化：记日志丢弃
				log.Printf("[MQ] 💀 Worker-%d 任务彻底失败（已达 %d 次重试）: id=%s type=%s err=%v\n",
					workerID, msg.MaxRetries, msg.ID, msg.Type, handleErr)
				delivery.Nack(false, false) // multiple=false, requeue=false
			}
		}
	}

	// deliveries channel 关闭 = MQ 连接断开
	// 返回 error，让调用方知道消费者已退出（重连后会重新启动消费）
	return fmt.Errorf("MQ 连接断开，消费者退出")
}
