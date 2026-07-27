// ============================================================
// pkg/mq/connection.go — RabbitMQ 连接管理
// ============================================================
// 【这个文件干什么？】
//   管理和 RabbitMQ 服务器之间的 TCP 连接。
//
// 【AMQP 协议的两个层级概念】
//   Connection（连接）: 一条 TCP 长连接，相当于"打电话的电话线"
//   Channel（通道）:   连接内的逻辑通道，相当于"电话线里的分机号"
//                      一个 Connection 可以开多个 Channel，
//                      发布消息用 Channel A，消费消息用 Channel B，互不干扰。
//
// 【为什么要自动重连？】
//   RabbitMQ 服务器可能重启、网络可能抖动，连接会断。
//   如果断了不重连，服务就成"哑炮"了——任务发布全部失败。
//   这里的策略：连接断开 → 后台 goroutine 每 3 秒尝试重连，直到成功。
//
// 【和 HTTP 短连接的区别】
//   HTTP 是"一问一答"短连接，用完就关。
//   AMQP 是长连接，建立一次用一辈子，性能高得多（没有 TCP 握手开销）。
// ============================================================

package mq

import (
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ============================================================
// 交换机与队列的常量定义
// ============================================================

const (
	// ExchangeName 交换机名称
	// direct 类型：消息按 routing key 精确路由到绑定的队列
	ExchangeName = "supply.tasks"

	// QueueName 任务队列名称
	QueueName = "supply.task_queue"

	// RoutingKey 路由键
	// direct 交换机下，routing key 决定消息进哪个队列
	// 我们所有任务共用一个 key，任务类型靠消息体里的 type 字段区分
	RoutingKey = "task"
)

// ============================================================
// Client 结构体 —— RabbitMQ 客户端
// ============================================================

// Client 封装了与 RabbitMQ 的连接，对外提供 Publish / Consume 能力
type Client struct {
	// ----- 连接信息 -----
	url string // AMQP 连接字符串: amqp://user:pass@host:port/vhost

	// ----- AMQP 连接对象 -----
	conn *amqp.Connection // TCP 连接（电话线）
	ch   *amqp.Channel    // 逻辑通道（分机号）

	// ----- 并发保护 -----
	mu sync.RWMutex // 保护 conn/ch 的并发读写（发布和重连可能同时进行）

	// ----- 生命周期 -----
	closed    bool        // 是否已主动关闭（主动关闭后不再重连）
	closeChan chan struct{} // 通知后台 goroutine 退出的信号
}

// ============================================================
// 创建客户端
// ============================================================

// NewClient 创建一个 RabbitMQ 客户端（创建后需要调用 Connect() 建立连接）
func NewClient(url string) *Client {
	return &Client{
		url:       url,
		closeChan: make(chan struct{}),
	}
}

// ============================================================
// 建立连接
// ============================================================

// Connect 连接到 RabbitMQ 服务器，并声明交换机和队列
//
// 【声明（Declare）是什么意思？】
//   RabbitMQ 要求交换机和队列"先声明再使用"。
//   声明是幂等的：存在就不管，不存在就创建。
//   相当于打地基 —— 往 MQ 里注册："我要用这个名字的交换机和队列"
//
// 【这个函数做了什么？】
//   1. 建立 TCP 连接 → 2. 打开 Channel → 3. 声明 direct 交换机 → 4. 声明队列 → 5. 绑定队列到交换机
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ----- 第 1 步：建立 TCP 连接 -----
	// amqp.Dial 会解析 amqp://user:pass@host:port/vhost 格式的 URL
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	c.conn = conn

	// ----- 第 2 步：打开逻辑通道 -----
	// Channel 是 AMQP 的核心抽象，所有操作（发布/消费）都在 Channel 上进行
	// 一个 Connection 可以开很多 Channel，轻量级，不用担心资源
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("打开 AMQP Channel 失败: %w", err)
	}
	c.ch = ch

	// ----- 第 3 步：声明 direct 交换机 -----
	// 参数说明：
	//   name         - 交换机名称
	//   kind         - "direct" 直连型：routing key 精确匹配
	//   durable      - false：非持久化（MQ 重启后交换机消失，符合"不加持久化"要求）
	//   autoDelete   - false：不自动删除（没有绑定队列时也保留）
	//   internal     - false：允许外部发布者发送消息
	//   noWait       - false：等待服务器确认声明成功
	//   args         - nil：无额外参数
	err = ch.ExchangeDeclare(
		ExchangeName, // name
		"direct",     // kind
		false,        // durable（不持久化）
		false,        // autoDelete
		false,        // internal
		false,        // noWait
		nil,          // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明交换机失败: %w", err)
	}

	// ----- 第 4 步：声明任务队列 -----
	//   name         - 队列名称
	//   durable      - false：非持久化（MQ 重启后队列和消息消失）
	//   autoDelete   - false：最后一个消费者断开时不自动删除
	//   exclusive    - false：不排他（多个连接可以共享这个队列）
	//   noWait       - false：等待服务器确认
	//   args         - nil：无额外参数（不设置 TTL、死信等高级特性）
	_, err = ch.QueueDeclare(
		QueueName, // name
		false,     // durable
		false,     // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明队列失败: %w", err)
	}

	// ----- 第 5 步：绑定队列到交换机 -----
	// 绑定 = 告诉交换机："凡是 routing key = 'task' 的消息，都给我放到这个队列里"
	//   queue      - 队列名
	//   key        - 绑定的 routing key
	//   exchange   - 交换机名
	//   noWait     - false：等待确认
	//   args       - nil
	err = ch.QueueBind(
		QueueName,    // queue
		RoutingKey,   // key
		ExchangeName, // exchange
		false,        // noWait
		nil,          // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("绑定队列失败: %w", err)
	}

	log.Println("[MQ] ✅ RabbitMQ 连接成功，交换机/队列已声明")

	// ----- 第 6 步：启动断线重连监听 -----
	// 在后台 goroutine 中监听连接断开事件，断了就自动重连
	go c.watchConnection()

	return nil
}

// ============================================================
// 断线自动重连
// ============================================================

// watchConnection 监听连接断开事件，自动重连
//
// 【怎么知道连接断了？】
//   amqp.Connection 提供了一个 NotifyClose() 方法，
//   返回一个 channel，连接断开时会往里发一个 error。
//   我们只需要 select 监听这个 channel 就行。
//
// 【重连策略】
//   简单固定间隔：每 3 秒试一次，直到成功。
//   生产环境更讲究的做法是指数退避（1s → 2s → 4s → 8s...），
//   防止 MQ 宕机时疯狂重连打爆服务器。
func (c *Client) watchConnection() {
	for {
		c.mu.RLock()
		conn := c.conn
		closed := c.closed
		c.mu.RUnlock()

		if closed || conn == nil {
			return // 已主动关闭或从未连接
		}

		// NotifyClose 返回的 channel 在连接断开时收到 error
		closeErr := <-conn.NotifyClose(make(chan *amqp.Error))

		// 检查是否主动关闭
		c.mu.RLock()
		closed = c.closed
		c.mu.RUnlock()
		if closed {
			return
		}

		log.Printf("[MQ] ⚠️ 连接断开: %v，3 秒后尝试重连...\n", closeErr)

		// 等待 3 秒（可以被 closeChan 打断）
		select {
		case <-time.After(3 * time.Second):
		case <-c.closeChan:
			return
		}

		// 尝试重连
		for {
			log.Println("[MQ] 🔄 正在重连...")
			if err := c.Connect(); err != nil {
				log.Printf("[MQ] ❌ 重连失败: %v，3 秒后重试\n", err)
				select {
				case <-time.After(3 * time.Second):
					continue
				case <-c.closeChan:
					return
				}
			}
			log.Println("[MQ] ✅ 重连成功")
			break
		}
	}
}

// ============================================================
// 关闭连接
// ============================================================

// Close 关闭连接（主动关闭，不会触发重连）
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	c.closed = true
	close(c.closeChan) // 通知后台 goroutine 退出

	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("[MQ] 🔌 连接已关闭")
}

// ============================================================
// 内部工具：获取当前 Channel（线程安全）
// ============================================================

// channel 返回当前的 AMQP Channel
// 发布和消费都需要 Channel，通过这个方法获取保证拿到的是最新连接上的
func (c *Client) channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ch == nil || c.ch.IsClosed() {
		return nil, fmt.Errorf("AMQP Channel 不可用（连接可能断开，正在重连）")
	}
	return c.ch, nil
}
