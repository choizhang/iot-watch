package services

import (
	"elder-guard-iot/models"
	"fmt"
	"log"
	"time"
)

// AlertService 告警服务
type AlertService struct {
	redisSvc *RedisService
	mysqlSvc *MySQLService
}

// alertService 全局告警服务实例
var alertService *AlertService

// InitAlertService 初始化告警服务
func InitAlertService(redisSvc *RedisService, mysqlSvc *MySQLService) {
	alertService = &AlertService{
		redisSvc: redisSvc,
		mysqlSvc: mysqlSvc,
	}
}

// GetAlertService 获取告警服务实例
func GetAlertService() *AlertService {
	return alertService
}

// HandleSOSAlert 处理 SOS 告警
func (s *AlertService) HandleSOSAlert(payload *models.DevicePayload) error {
	log.Printf("\n"+
		"╔══════════════════════════════════════════════════════════════╗\n"+
		"║  [CRITICAL ALERT] 收到长者 SOS 救命告警！设备号：%-15s      ║\n"+
		"╠══════════════════════════════════════════════════════════════╣\n"+
		"║  经度：%.6f                                           ║\n"+
		"║  纬度：%.6f                                           ║\n"+
		"║  心率：%-3d bpm                                        ║\n"+
		"║  时间：%s                                           ║\n"+
		"╚══════════════════════════════════════════════════════════════╝\n",
		payload.IMEI, payload.Longitude, payload.Latitude, payload.HeartRate,
		payload.Timestamp.Format("2006-01-02 15:04:05"))

	// 1. 在 Redis 写入高优先级告警标识
	if err := s.redisSvc.SetSOSAlert(payload.IMEI); err != nil {
		log.Printf("[ERROR] 设置 SOS 告警标识失败: %v", err)
	}

	// 2. 在 MySQL 记录告警
	alert := &models.DeviceAlert{
		IMEI:       payload.IMEI,
		AlertType:  models.MsgTypeSOS,
		Latitude:   payload.Latitude,
		Longitude:  payload.Longitude,
		HeartRate:  payload.HeartRate,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := s.mysqlSvc.SaveDeviceAlert(alert); err != nil {
		log.Printf("[ERROR] 保存 SOS 告警到数据库失败: %v", err)
	}

	// 3. 预留调用 APNs/FCM 推送接口
	// TODO: 调用推送服务发送告警通知给紧急联系人
	s.sendPushNotification(payload, "SOS")

	return nil
}

// HandleFallAlert 处理跌倒告警
func (s *AlertService) HandleFallAlert(payload *models.DevicePayload) error {
	log.Printf("\n"+
		"╔══════════════════════════════════════════════════════════════╗\n"+
		"║  [CRITICAL ALERT] 收到长者跌倒告警！设备号：%-15s     ║\n"+
		"╠══════════════════════════════════════════════════════════════╣\n"+
		"║  经度：%.6f                                           ║\n"+
		"║  纬度：%.6f                                           ║\n"+
		"║  心率：%-3d bpm                                        ║\n"+
		"║  时间：%s                                           ║\n"+
		"╚══════════════════════════════════════════════════════════════╝\n",
		payload.IMEI, payload.Longitude, payload.Latitude, payload.HeartRate,
		payload.Timestamp.Format("2006-01-02 15:04:05"))

	// 1. 在 Redis 写入高优先级告警标识
	if err := s.redisSvc.SetFallAlert(payload.IMEI); err != nil {
		log.Printf("[ERROR] 设置跌倒告警标识失败: %v", err)
	}

	// 2. 在 MySQL 记录告警
	alert := &models.DeviceAlert{
		IMEI:       payload.IMEI,
		AlertType:  models.MsgTypeFall,
		Latitude:   payload.Latitude,
		Longitude:  payload.Longitude,
		HeartRate:  payload.HeartRate,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := s.mysqlSvc.SaveDeviceAlert(alert); err != nil {
		log.Printf("[ERROR] 保存跌倒告警到数据库失败: %v", err)
	}

	// 3. 预留调用 APNs/FCM 推送接口
	// TODO: 调用推送服务发送告警通知给紧急联系人
	s.sendPushNotification(payload, "FALL")

	return nil
}

// sendPushNotification 发送推送通知（预留接口）
// TODO: 实现 APNs (iOS) 和 FCM (Android) 推送
func (s *AlertService) sendPushNotification(payload *models.DevicePayload, alertType string) {
	// 预留：构建推送消息体
	pushMessage := map[string]interface{}{
		"title": fmt.Sprintf("长者%s告警", map[string]string{"SOS": "紧急求救", "FALL": "跌倒检测"}[alertType]),
		"body": fmt.Sprintf("设备 %s 发出%s告警，请立即确认！", payload.IMEI, alertType),
		"data": map[string]interface{}{
			"imei":      payload.IMEI,
			"latitude":  payload.Latitude,
			"longitude": payload.Longitude,
			"heartRate": payload.HeartRate,
			"alertType": alertType,
		},
	}

	log.Printf("[INFO] 推送通知已构建（待发送）: %+v", pushMessage)

	// TODO: 调用实际推送服务
	// if payload.DeviceType == "iOS" {
	//     s.sendAPNs(pushMessage)
	// } else {
	//     s.sendFCM(pushMessage)
	// }
}

// ProcessPayload 处理设备报文，根据消息类型执行相应逻辑
func (s *AlertService) ProcessPayload(payload *models.DevicePayload) error {
	// 更新设备状态到 Redis
	state := &models.DeviceState{
		IMEI:        payload.IMEI,
		HeartRate:   payload.HeartRate,
		Battery:     payload.Battery,
		Latitude:    payload.Latitude,
		Longitude:   payload.Longitude,
		LastUpdate:  payload.Timestamp,
		MsgType:     payload.MsgType,
	}

	// 根据消息类型设置状态
	switch payload.MsgType {
	case models.MsgTypeSOS:
		state.Status = "alert"
	case models.MsgTypeFall:
		state.Status = "alert"
	default:
		state.Status = "online"
	}

	// 保存到 Redis
	if err := s.redisSvc.SetDeviceState(state); err != nil {
		log.Printf("[ERROR] 更新设备状态失败: %v", err)
	}

	// 在 MySQL 记录事件
	event := &models.DeviceEvent{
		IMEI:        payload.IMEI,
		EventType:   payload.MsgType,
		Latitude:    payload.Latitude,
		Longitude:   payload.Longitude,
		HeartRate:   payload.HeartRate,
		Battery:     payload.Battery,
		RawPayload:  payload.RawPayload,
		EventTime:   payload.Timestamp,
		CreatedAt:   time.Now(),
	}
	if err := s.mysqlSvc.SaveDeviceEvent(event); err != nil {
		log.Printf("[ERROR] 保存设备事件失败: %v", err)
	}

	// 确保 MySQL 中存在设备记录，然后更新设备最后状态
	s.mysqlSvc.GetOrCreateDevice(payload.IMEI)
	s.mysqlSvc.UpdateDeviceStatus(payload.IMEI, state.Status, payload.Timestamp,
		payload.Latitude, payload.Longitude, payload.HeartRate, payload.Battery)

	// 处理告警逻辑
	if payload.IsAlert() {
		switch payload.MsgType {
		case models.MsgTypeSOS:
			return s.HandleSOSAlert(payload)
		case models.MsgTypeFall:
			return s.HandleFallAlert(payload)
		}
	}

	return nil
}
