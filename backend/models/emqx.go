package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// EMQXWebhookPayload EMQX 5.x Webhook 报文结构
// EMQX 会通过 HTTP POST 向我们的服务发送此格式的数据
type EMQXWebhookPayload struct {
	// ClientID 设备客户端 ID（通常为 IMEI）
	ClientID string `json:"clientid"`
	// Username 设备用户名
	Username string `json:"username"`
	// Payload 原始报文内容（ASCII/HEX 字符串）
	Payload string `json:"payload"`
	// Topic EMQX 主题，如 "device/telemetry"
	Topic string `json:"topic"`
	// Ts 时间戳（毫秒）
	Ts int64 `json:"ts"`
	// TsStr 时间戳字符串（方便调试）
	TsStr string `json:"ts_str"`
	// Peerhost 设备 IP 地址
	Peerhost string `json:"peerhost"`
	// QoS 服务质量等级 (0, 1, 2)
	QoS int `json:"qos"`
}

// String 返回 JSON 格式的字符串表示
func (e *EMQXWebhookPayload) String() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// GetTime 获取时间对象
func (e *EMQXWebhookPayload) GetTime() time.Time {
	return time.UnixMilli(e.Ts)
}

// DevicePayload 手环设备解析后的数据模型
type DevicePayload struct {
	// IMEI 设备号（国际移动设备识别码）
	IMEI string
	// MsgType 消息类型：HEARTBEAT(心跳)、LOCATION(定位)、SOS(紧急求救)、FALL(跌倒检测)、HEART_RATE(心率)
	MsgType string
	// Latitude 纬度
	Latitude float64
	// Longitude 经度
	Longitude float64
	// HeartRate 心率值（bpm）
	HeartRate int
	// Battery 电量（0-100%）
	Battery int
	// RawPayload 原始报文
	RawPayload string
	// Timestamp 解析时间
	Timestamp time.Time
}

// MsgType 常量定义
const (
	MsgTypeHeartbeat = "HEARTBEAT"
	MsgTypeLocation  = "LOCATION"
	MsgTypeSOS       = "SOS"
	MsgTypeFall      = "FALL"
	MsgTypeHeartRate = "HEART_RATE"
)

// IsAlert 判断是否为告警类型消息
func (p *DevicePayload) IsAlert() bool {
	return p.MsgType == MsgTypeSOS || p.MsgType == MsgTypeFall
}

// String 返回格式化的字符串表示
func (p *DevicePayload) String() string {
	return "[DEVICE PAYLOAD] IMEI:" + p.IMEI +
		" Type:" + p.MsgType +
		" Lat:" + fmt.Sprintf("%.6f", p.Latitude) +
		" Lng:" + fmt.Sprintf("%.6f", p.Longitude) +
		" HR:" + fmt.Sprintf("%d", p.HeartRate) +
		" Batt:" + fmt.Sprintf("%d%%", p.Battery)
}
