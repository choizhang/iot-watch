package services

import (
	"context"
	"elder-guard-iot/models"
	"fmt"
	"log"
	"time"
)

// OfflineDetector 设备离线巡检服务
type OfflineDetector struct {
	mysqlSvc       *MySQLService
	redisSvc       *RedisService
	checkInterval  time.Duration
	offlineTimeout time.Duration
}

// NewOfflineDetector 创建离线检测服务
func NewOfflineDetector(mysqlSvc *MySQLService, redisSvc *RedisService, checkInterval, offlineTimeout time.Duration) *OfflineDetector {
	return &OfflineDetector{
		mysqlSvc:       mysqlSvc,
		redisSvc:       redisSvc,
		checkInterval:  checkInterval,
		offlineTimeout: offlineTimeout,
	}
}

// Start 启动后台巡检循环
func (d *OfflineDetector) Start(ctx context.Context) {
	log.Printf("[INFO] 设备离线检测服务已启动 (巡检间隔: %v, 判定超时: %v)", d.checkInterval, d.offlineTimeout)
	ticker := time.NewTicker(d.checkInterval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Printf("[INFO] 设备离线检测服务已停止")
				ticker.Stop()
				return
			case <-ticker.C:
				d.checkOfflineDevices()
			}
		}
	}()
}

// checkOfflineDevices 扫描超时未上报的设备并更正状态
func (d *OfflineDetector) checkOfflineDevices() {
	db := d.mysqlSvc.GetDB()
	if db == nil {
		return
	}

	thresholdTime := time.Now().Add(-d.offlineTimeout)

	// 查询 status != 'offline' 且 last_heartbeat < thresholdTime 的设备
	var offlineDevices []models.Device
	err := db.Where("status != ? AND last_heartbeat < ?", "offline", thresholdTime).Find(&offlineDevices).Error
	if err != nil {
		log.Printf("[ERROR] [离线巡检] 查询超时设备失败: %v", err)
		return
	}

	if len(offlineDevices) == 0 {
		return
	}

	bgCtx := context.Background()
	for _, dev := range offlineDevices {
		// 跳过从未使用过的未初始化设备 (last_heartbeat 为零值)
		if dev.LastHeartbeat.IsZero() {
			continue
		}

		log.Printf("[WARN] [离线巡检] 设备 %s 超过 %v 未发送心跳，标记为离线 (最后心跳: %s)",
			dev.IMEI, d.offlineTimeout, dev.LastHeartbeat.Format("2006-01-02 15:04:05"))

		// 1. 更新 MySQL 状态为 offline
		db.Model(&models.Device{}).Where("imei = ?", dev.IMEI).Update("status", "offline")

		// 2. 同步更新 Redis 缓存为 offline
		key := fmt.Sprintf("device:status:%s", dev.IMEI)
		d.redisSvc.GetClient().HSet(bgCtx, key, map[string]interface{}{
			"status":     "offline",
			"updated_at": time.Now().Unix(),
		})

		// 记录设备事件
		d.mysqlSvc.SaveDeviceEvent(&models.DeviceEvent{
			IMEI:       dev.IMEI,
			EventType:  "OFFLINE",
			RawPayload: "SYSTEM_OFFLINE_DETECTOR_TRIGGERED",
			EventTime:  time.Now(),
			CreatedAt:  time.Now(),
		})
	}
}
