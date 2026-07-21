// ============================================================
// pkg/worker/handlers.go — 任务处理函数
// ============================================================
// 【这是什么？】
//   当 Worker 从队列取出 Task 后，根据 Task.Type 找到对应的 Handler 执行。
//   每个 Handler 是一个独立的业务逻辑块，负责处理一种类型的任务。
//
// 【为什么放在独立文件？】
//   和 Controller / Model 分层一样：pool.go 是"框架层"，handlers.go 是"业务层"。
//   新增任务类型只需要在这里加一个 Handler 然后 Register 即可，
//   不用动 Worker Pool 的核心代码。
//
// 【关于 import 循环依赖】
//   worker 包不应该 import internal/model，否则会形成循环依赖
//   (model 可能引用 worker，worker 再引用 model)。
//   解决方法：这里用 fmt.Println/log 模拟"写审计日志"的动作，
//   真实落地时，通过 RegisterHandler 把 model 层的函数注入进来。
//   详见 cmd/api/main.go 的 InitTaskWorker()。
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
	// ★ 类型断言：把 any 类型的 Payload 还原成具体类型
	payload, ok := task.Payload.(AuditLogPayload)
	if !ok {
		return fmt.Errorf("审计日志任务 Payload 类型错误: 期望 AuditLogPayload, 实际 %T", task.Payload)
	}

	// 模拟写审计日志（实际项目应该写 MySQL audit_log 表）
	// 这里用 log.Println 输出到标准输出，Docker 日志/ELK 可以收集
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
// 【业务逻辑】
//   当产品的供应链历史记录增加时，检查是否需要自动流转产品状态：
//   例如：产品刚被"装车发运" → 自动把产品状态从 "in_warehouse" 改为 "in_transit"
//
// 【为什么异步做？】
//   状态流转涉及：查产品当前状态 → 查最新历史操作 → 匹配状态机规则 → 更新产品状态
//   这一套要 3~4 次 DB 查询，丢到后台做，HTTP 不用等。
func HandleStatusSync(ctx context.Context, task Task) error {
	payload, ok := task.Payload.(StatusSyncPayload)
	if !ok {
		return fmt.Errorf("状态同步任务 Payload 类型错误: 期望 StatusSyncPayload, 实际 %T", task.Payload)
	}

	// 模拟状态同步逻辑
	// 实际应该：查 product 表 → 查最新 supply_history → 匹配流转规则 → update product.status
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
// 【业务逻辑】
//   更新产品的聚合统计数据，比如"该产品有多少条供应链历史"。
//   如果每次 GET /products 都实时 COUNT，数据多了会很慢。
//   改为创建/修改时异步刷新一个缓存字段。
func HandleStatsRefresh(ctx context.Context, task Task) error {
	payload, ok := task.Payload.(StatsRefreshPayload)
	if !ok {
		return fmt.Errorf("统计刷新任务 Payload 类型错误: 期望 StatsRefreshPayload, 实际 %T", task.Payload)
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
