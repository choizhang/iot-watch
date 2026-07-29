package models

import (
	"encoding/json"
	"time"
)

// DeviceState Redis 设备状态模型
// 用于实时存储设备最新状态信息
type DeviceState struct {
	// IMEI 设备号
	IMEI string `json:"imei"`
	// Status 设备状态：online/offline/alert
	Status string `json:"status"`
	// HeartRate 心率值
	HeartRate int `json:"heart_rate"`
	// Battery 电量
	Battery int `json:"battery"`
	// Latitude 纬度
	Latitude float64 `json:"latitude"`
	// Longitude 经度
	Longitude float64 `json:"longitude"`
	// LastUpdate 最后更新时间
	LastUpdate time.Time `json:"last_update"`
	// MsgType 最新消息类型
	MsgType string `json:"msg_type"`
}

// RedisKeyDeviceState 获取设备状态 Key
func RedisKeyDeviceState(imei string) string {
	return "device:state:" + imei
}

// RedisKeyAlertSOS 获取 SOS 告警 Key
func RedisKeyAlertSOS(imei string) string {
	return "alert:sos:" + imei
}

// RedisKeyAlertFall 获取跌倒告警 Key
func RedisKeyAlertFall(imei string) string {
	return "alert:fall:" + imei
}

// ToJSON 序列化为 JSON 字符串
func (s *DeviceState) ToJSON() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串反序列化
func (s *DeviceState) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), s)
}

// IsOnline 判断设备是否在线（5分钟内有心跳）
func (s *DeviceState) IsOnline() bool {
	return time.Since(s.LastUpdate) < 5*time.Minute
}

// IsAlert 判断是否有未处理的告警
func (s *DeviceState) IsAlert() bool {
	return s.Status == "alert"
}
