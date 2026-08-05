package services

import (
	"elder-guard-iot/config"
	"elder-guard-iot/models"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQLService MySQL 服务
type MySQLService struct {
	db *gorm.DB
}

// mysqlClient 全局 MySQL 客户端
var mysqlClient *MySQLService

// InitMySQL 初始化 MySQL 连接
func InitMySQL(cfg *config.Config) error {
	dsn := cfg.GetMySQLDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("MySQL 连接失败: %w", err)
	}

	// 获取底层 sql.DB 并设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层 DB 失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	mysqlClient = &MySQLService{db: db}

	// 自动迁移表结构
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Printf("[INFO] MySQL 连接成功: %s:%s/%s", cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	return nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate() error {
	err := mysqlClient.db.AutoMigrate(
		&models.Device{},
		&models.DeviceEvent{},
		&models.DeviceAlert{},
		&models.AlarmOrder{},
		&models.DeviceContact{},
		&models.DeviceSetting{},
		&models.DeviceReminder{},
		&models.Geofence{},
	)
	if err != nil {
		return err
	}

	// 自动修复历史上报但解析前保存为 0 的心率事件
	mysqlClient.db.Exec(`
		UPDATE device_events 
		SET heart_rate = CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(raw_payload, ',', -1), ']', 1) AS UNSIGNED) 
		WHERE (event_type = 'HEART' OR raw_payload LIKE '%*HEART,%') AND heart_rate = 0 AND raw_payload LIKE '%,%'
	`)

	// 刷新 devices 表中的最新心率
	var devices []models.Device
	mysqlClient.db.Find(&devices)
	for _, dev := range devices {
		var lastEvent models.DeviceEvent
		if err := mysqlClient.db.Where("imei = ? AND heart_rate > 0", dev.IMEI).Order("event_time DESC").First(&lastEvent).Error; err == nil {
			mysqlClient.db.Model(&models.Device{}).Where("imei = ?", dev.IMEI).UpdateColumn("last_heart_rate", lastEvent.HeartRate)
		}
	}

	return nil
}

// GetMySQLClient 获取 MySQL 客户端实例
func GetMySQLClient() *MySQLService {
	return mysqlClient
}

// GetDB 获取 gorm.DB 实例
func (m *MySQLService) GetDB() *gorm.DB {
	return m.db
}

// SaveDeviceEvent 保存设备事件
func (m *MySQLService) SaveDeviceEvent(event *models.DeviceEvent) error {
	if err := m.db.Create(event).Error; err != nil {
		return fmt.Errorf("保存设备事件失败: %w", err)
	}
	log.Printf("[DEBUG] 设备事件已保存: IMEI=%s, Type=%s", event.IMEI, event.EventType)
	return nil
}

// SaveDeviceAlert 保存设备告警
func (m *MySQLService) SaveDeviceAlert(alert *models.DeviceAlert) error {
	if err := m.db.Create(alert).Error; err != nil {
		return fmt.Errorf("保存设备告警失败: %w", err)
	}
	log.Printf("[DEBUG] 设备告警已保存: IMEI=%s, Type=%s", alert.IMEI, alert.AlertType)
	return nil
}

// UpdateDeviceStatus 更新设备状态
func (m *MySQLService) UpdateDeviceStatus(imei string, status string, heartbeat time.Time,
	lat, lng float64, heartRate, battery int, bp string, spo2 int, hrv int, steps int) error {

	updates := map[string]interface{}{
		"last_heartbeat": heartbeat,
	}

	if lat != 0 || lng != 0 {
		updates["last_latitude"] = lat
		updates["last_longitude"] = lng
	}

	if heartRate > 0 {
		updates["last_heart_rate"] = heartRate
	}

	if battery > 0 {
		updates["battery"] = battery
	}

	if bp != "" {
		updates["blood_pressure"] = bp
	}

	if spo2 > 0 {
		updates["spo2"] = spo2
		updates["sp_o2"] = spo2
	}

	if hrv > 0 {
		updates["hrv"] = hrv
	}

	if steps > 0 {
		updates["steps"] = steps
	}

	// 根据消息类型更新状态
	if status != "" {
		updates["status"] = status
	}

	if err := m.db.Model(&models.Device{}).Where("imei = ?", imei).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新设备状态失败: %w", err)
	}
	return nil
}

// GetOrCreateDevice 获取或创建设备记录
func (m *MySQLService) GetOrCreateDevice(imei string) (*models.Device, error) {
	var device models.Device
	result := m.db.Where("imei = ?", imei).First(&device)
	if result.Error == gorm.ErrRecordNotFound {
		// 创建设备记录
		device = models.Device{
			IMEI:    imei,
			Status:  "online",
			CreatedAt: time.Now(),
		}
		if err := m.db.Create(&device).Error; err != nil {
			return nil, fmt.Errorf("创建设备记录失败: %w", err)
		}
		log.Printf("[INFO] 新设备注册: IMEI=%s", imei)
		return &device, nil
	} else if result.Error != nil {
		return nil, fmt.Errorf("查询设备失败: %w", result.Error)
	}
	return &device, nil
}

// GetPendingAlerts 获取待处理的告警
func (m *MySQLService) GetPendingAlerts() ([]models.DeviceAlert, error) {
	var alerts []models.DeviceAlert
	if err := m.db.Where("status = ?", "pending").Find(&alerts).Error; err != nil {
		return nil, fmt.Errorf("查询待处理告警失败: %w", err)
	}
	return alerts, nil
}

// SaveAlarmOrder 保存告警工单
func (m *MySQLService) SaveAlarmOrder(order *models.AlarmOrder) error {
	if err := m.db.Create(order).Error; err != nil {
		return fmt.Errorf("保存告警工单失败: %w", err)
	}
	log.Printf("[DEBUG] 告警工单已保存: AlarmID=%s, IMEI=%s, Type=%s, Status=%s",
		order.AlarmID, order.DeviceIMEI, order.AlertType, order.Status)
	return nil
}

// CleanOldDeviceEvents 清理指定天数之前的常规高频设备事件记录（告警工单 AlarmOrder 与设备最新状态不受影响）
func (m *MySQLService) CleanOldDeviceEvents(retentionDays int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	result := m.db.Where("created_at < ?", cutoffTime).Delete(&models.DeviceEvent{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理过期历史事件失败: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		log.Printf("[INFO] [MySQL 归档清理] 已清理 %d 天前的过期历史设备事件记录: 共 %d 条", retentionDays, result.RowsAffected)
	}
	return result.RowsAffected, nil
}

// Close 关闭数据库连接
func (m *MySQLService) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
