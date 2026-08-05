package models

import (
	"time"
)

// Device 设备表 - MySQL 模型
type Device struct {
	// ID 主键自增
	ID uint `gorm:"primaryKey" json:"id"`
	// IMEI 设备唯一标识
	IMEI string `gorm:"uniqueIndex;size:20;not null" json:"imei"`
	// DeviceName 设备名称
	DeviceName string `gorm:"size:100" json:"device_name"`
	// DeviceModel 设备型号
	DeviceModel string `gorm:"size:50" json:"device_model"`
	// OwnerName 所属老人姓名
	OwnerName string `gorm:"size:50" json:"owner_name"`
	// OwnerPhone 紧急联系人电话
	OwnerPhone string `gorm:"size:20" json:"owner_phone"`
	// Status 设备状态：online/offline/alert
	Status string `gorm:"size:20;default:offline" json:"status"`
	// LastHeartbeat 最后心跳时间
	LastHeartbeat time.Time `json:"last_heartbeat"`
	// LastLatitude 最后纬度
	LastLatitude float64 `json:"last_latitude"`
	// LastLongitude 最后经度
	LastLongitude float64 `json:"last_longitude"`
	// LastHeartRate 最后心率
	LastHeartRate int `json:"last_heart_rate"`
	// Battery 电量
	Battery int `json:"battery"`
	// Health metrics
	BloodPressure string `gorm:"size:20" json:"bp"`
	SpO2          int    `gorm:"column:spo2" json:"spo2"`
	HRV           int    `json:"hrv"`
	Steps         int    `json:"steps"`
	// Accuracy 第三方定位 API 真实解算返回的精度米数
	Accuracy float64 `gorm:"default:18.5" json:"accuracy"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}

// DeviceEvent 设备事件表 - 记录设备上报事件
type DeviceEvent struct {
	// ID 主键自增
	ID uint `gorm:"primaryKey" json:"id"`
	// IMEI 设备号
	IMEI string `gorm:"index;size:20;not null" json:"imei"`
	// EventType 事件类型：heartbeat/location/sos/fall/heart_rate
	EventType string `gorm:"size:20;not null" json:"event_type"`
	// LocationType 定位方式：GPS/WIFI/LBS
	LocationType string `gorm:"size:20;default:GPS" json:"location_type"`
	// Latitude 纬度
	Latitude float64 `json:"latitude"`
	// Longitude 经度
	Longitude float64 `json:"longitude"`
	// HeartRate 心率值
	HeartRate int `json:"heart_rate"`
	// Battery 电量
	Battery int `json:"battery"`
	// Health metrics
	BloodPressure string `gorm:"size:20" json:"bp"`
	SpO2          int    `gorm:"column:spo2" json:"spo2"`
	HRV           int    `json:"hrv"`
	Steps         int    `json:"steps"`
	// RawPayload 原始报文
	RawPayload string `gorm:"type:text" json:"raw_payload"`
	// EventTime 事件发生时间
	EventTime time.Time `json:"event_time"`
	// CreatedAt 记录创建时间
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (DeviceEvent) TableName() string {
	return "device_events"
}

// DeviceAlert 设备告警表 - 记录紧急告警
type DeviceAlert struct {
	// ID 主键自增
	ID uint `gorm:"primaryKey" json:"id"`
	// IMEI 设备号
	IMEI string `gorm:"index;size:20;not null" json:"imei"`
	// AlertType 告警类型：SOS/FALL
	AlertType string `gorm:"size:10;not null" json:"alert_type"`
	// Latitude 告警时纬度
	Latitude float64 `json:"latitude"`
	// Longitude 告警时经度
	Longitude float64 `json:"longitude"`
	// HeartRate 告警时心率
	HeartRate int `json:"heart_rate"`
	// Status 告警状态：pending/acknowledged/resolved
	Status string `gorm:"size:20;default:pending" json:"status"`
	// AcknowledgedAt 确认时间
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	// ResolvedAt 解决时间
	ResolvedAt *time.Time `json:"resolved_at"`
	// CreatedAt 记录创建时间
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (DeviceAlert) TableName() string {
	return "device_alerts"
}

// AlarmOrder 告警工单表 - SOS/FALL 告警业务工单
type AlarmOrder struct {
	// ID 主键自增
	ID uint `gorm:"primaryKey" json:"id"`
	// AlarmID 告警工单号（格式：SOS-YYYYMMDD-HHMMSS-IMEI后4位）
	AlarmID string `gorm:"uniqueIndex;size:50;not null" json:"alarm_id"`
	// ElderID 长者ID（关联长者资料表）
	ElderID uint `gorm:"index" json:"elder_id"`
	// DeviceIMEI 设备IMEI
	DeviceIMEI string `gorm:"index;size:20;not null" json:"device_imei"`
	// AlertType 告警类型：SOS/FALL
	AlertType string `gorm:"size:10;not null" json:"alert_type"`
	// TriggerTime 触发时间
	TriggerTime time.Time `json:"trigger_time"`
	// Latitude 告警时纬度
	Latitude float64 `json:"latitude"`
	// Longitude 告警时经度
	Longitude float64 `json:"longitude"`
	// HeartRate 告警时心率
	HeartRate int `json:"heart_rate"`
	// Status 工单状态：UNHANDLED（未处理）/HANDLING（处理中）/COMPLETED（已处理）
	Status string `gorm:"size:20;default:UNHANDLED;index" json:"status"`
	// HandlerID 处理人ID
	HandlerID *uint `json:"handler_id"`
	// HandlerName 处理人姓名
	HandlerName string `gorm:"size:50" json:"handler_name"`
	// HandlerNotes 处理备注
	HandlerNotes string `gorm:"type:text" json:"handler_notes"`
	// CreatedAt 记录创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (AlarmOrder) TableName() string {
	return "alarm_orders"
}

// AlarmStatus 常量
const (
	AlarmStatusUnhandeld = "UNHANDLED"
	AlarmStatusHandling   = "HANDLING"
	AlarmStatusCompleted = "COMPLETED"
)

// DeviceContact 设备联系人表 - MySQL 模型
type DeviceContact struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IMEI      string    `gorm:"index;size:20;not null" json:"imei"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Phone     string    `gorm:"size:20;not null" json:"phone"`
	Relation  string    `gorm:"size:20" json:"relation"` // 家属, 医疗, 社区
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DeviceContact) TableName() string {
	return "device_contacts"
}

// DeviceSetting 设备设置表 - MySQL 模型
type DeviceSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IMEI      string    `gorm:"uniqueIndex;size:20;not null" json:"imei"`
	Interval  int       `gorm:"default:300" json:"interval"` // 上报间隔，单位秒
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DeviceSetting) TableName() string {
	return "device_settings"
}

// DeviceReminder 设备提醒表 - MySQL 模型
type DeviceReminder struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IMEI      string    `gorm:"index;size:20;not null" json:"imei"`
	Time      string    `gorm:"size:5;not null" json:"time"` // 格式 "HH:MM"
	Label     string    `gorm:"size:100;not null" json:"label"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (DeviceReminder) TableName() string {
	return "device_reminders"
}

// Geofence 电子围栏表 - MySQL 模型
type Geofence struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IMEI      string    `gorm:"index;size:20;not null" json:"imei"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Latitude  float64   `gorm:"not null" json:"latitude"`
	Longitude float64   `gorm:"not null" json:"longitude"`
	Radius    int       `gorm:"not null;default:500" json:"radius"` // 围栏半径（米）
	FenceType string    `gorm:"size:20;default:IN" json:"fence_type"` // IN (出界告警) / OUT (入界告警)
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Geofence) TableName() string {
	return "geofences"
}

