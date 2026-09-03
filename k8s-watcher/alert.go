// ============================================================================
// 钉钉告警发送模块
// ----------------------------------------------------------------------------
// 钉钉群机器人协议（"自定义机器人" Webhook）：
//   向机器人 Webhook 地址 POST 一段 JSON，群里就会收到一条消息。
//   本文件只实现最简单的 text（纯文本）消息类型，够用且最稳。
//
// 机器人的两种安全设置（创建机器人时在钉钉群里配置）：
//   1. 自定义关键词：消息文本必须包含该关键词（如"告警"）才会被接收。
//      本程序的消息固定以【K8s 告警】开头，关键词设为"告警"即可命中。
//   2. 加签：请求时用密钥对时间戳算一个 HMAC-SHA256 签名拼到 URL 上，
//      防止拿到 URL 的陌生人乱发消息。算法在下面 signURL 函数里。
//
// 发送失败怎么办？（本文件不重试）
//   本函数只"尽力发一次"，失败返回 error。调用方（main.go 的 check）
//   收到 error 不会记录去重状态，下个 resync 周期会自动重发——
//   相当于用 informer 的周期性重放当重试定时器，零额外重试代码。
// ============================================================================
package main

import (
	"bytes"
	"crypto/hmac" // HMAC：带密钥的哈希，加签算法的核心
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// 包级共享的 HTTP 客户端。
// 必须显式设置超时——Go 默认的 http.Client 没有超时，钉钉服务器一旦
// 无响应，程序会永远卡死在这条请求上。10s 对一次告警请求绰绰有余。
var httpClient = &http.Client{Timeout: 10 * time.Second}

// ---- 以下结构体和钉钉 API 的 JSON 格式一一对应 ----

type dingTalkContent struct {
	Content string `json:"content"` // 纯文本内容
}

type dingTalkAt struct {
	IsAtAll bool `json:"isAtAll"` // 是否 @所有人；告警场景下 false 即可
}

// 发送 text 消息的完整请求体，形如：
// {"msgtype":"text","text":{"content":"..."},"at":{"isAtAll":false}}
type dingTalkText struct {
	MsgType string          `json:"msgtype"`
	Text    dingTalkContent `json:"text"`
	At      dingTalkAt      `json:"at"`
}

// 钉钉的统一响应格式：{"errcode":0,"errmsg":"ok"}
type dingTalkResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// sendDingTalk 发送一条告警。
// 返回 nil  = "已处理"：发送成功，或 DRY-RUN 已记日志（调用方会记录去重状态）
// 返回 err = "未送达"：调用方不记录去重状态，下个 resync 自动重试
func sendDingTalk(cfg config, content string) error {
	// DRY-RUN：Webhook 未配置时只打印告警内容到日志并视为"已处理"。
	// 注意这里故意返回 nil 而不是 error——如果返回 error，每个 resync
	// 周期（60 秒）都会对同一个 Pod 重复打日志，把日志刷爆。
	// 该模式用于还没申请钉钉机器人时的功能验证。
	if cfg.WebhookURL == "" {
		log.Printf("[DRY-RUN] WEBHOOK_URL 未配置，仅记录告警内容：\n%s", content)
		return nil
	}
	target := cfg.WebhookURL
	if cfg.Secret != "" {
		target = signURL(target, time.Now().UnixMilli(), cfg.Secret)
	}

	body, err := json.Marshal(dingTalkText{
		MsgType: "text",
		Text:    dingTalkContent{Content: content},
	})
	if err != nil {
		return fmt.Errorf("构造消息失败: %w", err)
	}
	resp, err := httpClient.Post(target, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("请求钉钉失败: %w", err)
	}
	defer resp.Body.Close()

	// 最多读 4KB：钉钉响应很短，限制读取量防止异常响应吃掉内存
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉 HTTP %d: %s", resp.StatusCode, data)
	}
	var r dingTalkResp
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	// 关键：HTTP 200 不代表发送成功！关键词不匹配、加签错误等情况
	// 钉钉都返回 200，靠 errcode 区分（0=成功，310000=安全校验失败等）。
	// 只看状态码会把失败误判成成功，导致漏告警。
	if r.ErrCode != 0 {
		return fmt.Errorf("钉钉 errcode=%d errmsg=%s", r.ErrCode, r.ErrMsg)
	}
	return nil
}

// signURL 实现钉钉的"加签"安全设置：
//   时间戳 + "\n" + 密钥 拼成的字符串，用密钥做 HMAC-SHA256，
//   结果 Base64 编码再 URL 转义，最终拼到 Webhook 末尾：
//   ...&timestamp=毫秒时间戳&sign=签名
// 这是钉钉官方文档定义的算法，密钥就是创建机器人时 SEC 开头的那串。
func signURL(webhook string, ts int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d\n%s", ts, secret)))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	return fmt.Sprintf("%s&timestamp=%d&sign=%s", webhook, ts, sign)
}
