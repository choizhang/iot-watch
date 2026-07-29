package services

import (
	"context"
	"elder-guard-iot/config"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// InfluxDBService InfluxDB 时序数据库服务
type InfluxDBService struct {
	client    influxdb2.Client
	writeAPI  api.WriteAPI
	queryAPI  api.QueryAPI
	org       string
	bucket    string
}

// influxDBClient 全局 InfluxDB 客户端
var influxDBClient *InfluxDBService

// InitInfluxDB 初始化 InfluxDB 连接
func InitInfluxDB(cfg *config.Config) error {
	// 创建 InfluxDB 客户端
	client := influxdb2.NewClientWithOptions(
		cfg.InfluxDB.URL,
		cfg.InfluxDB.Token,
		influxdb2.DefaultOptions().SetBatchSize(100).SetFlushInterval(5000),
	)

	writeAPI := client.WriteAPI(cfg.InfluxDB.Org, cfg.InfluxDB.Bucket)
	queryAPI := client.QueryAPI(cfg.InfluxDB.Org)

	// 测试连接 - 查询 buckets 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := queryAPI.Query(ctx, "buckets()")
	if err != nil {
		return fmt.Errorf("InfluxDB 连接失败: %w", err)
	}

	influxDBClient = &InfluxDBService{
		client:   client,
		writeAPI: writeAPI,
		queryAPI: queryAPI,
		org:      cfg.InfluxDB.Org,
		bucket:   cfg.InfluxDB.Bucket,
	}

	log.Printf("[INFO] InfluxDB 连接成功: %s, Org: %s, Bucket: %s",
		cfg.InfluxDB.URL, cfg.InfluxDB.Org, cfg.InfluxDB.Bucket)

	// 启动异步错误处理
	go influxDBClient.handleErrors()

	return nil
}

// handleErrors 处理异步写入错误
func (s *InfluxDBService) handleErrors() {
	for err := range s.writeAPI.Errors() {
		log.Printf("[ERROR] InfluxDB 写入错误: %v", err)
	}
}

// GetInfluxDBClient 获取 InfluxDB 客户端实例
func GetInfluxDBClient() *InfluxDBService {
	return influxDBClient
}

// TelemetryData 设备遥测数据
type TelemetryData struct {
	IMEI       string
	EventType  string
	Latitude   float64
	Longitude  float64
	HeartRate  int
	Battery    int
	SOSFlag    bool
	RawPayload string
}

// WriteTelemetry 异步写入设备遥测数据到 InfluxDB
func (s *InfluxDBService) WriteTelemetry(data *TelemetryData) error {
	if s == nil || s.writeAPI == nil {
		return fmt.Errorf("InfluxDB 客户端未初始化")
	}

	// 创建数据点
	point := influxdb2.NewPoint(
		"device_telemetry", // Measurement
		map[string]string{
			"imei":       data.IMEI,
			"event_type": data.EventType,
		},
		map[string]interface{}{
			"lat":        data.Latitude,
			"lon":        data.Longitude,
			"heart_rate": data.HeartRate,
			"battery":    data.Battery,
			"sos_flag":   data.SOSFlag,
		},
		time.Now(),
	)

	// 异步写入
	s.writeAPI.WritePoint(point)

	log.Printf("[DEBUG] InfluxDB 写入点: IMEI=%s, Type=%s, Lat=%.6f, Lon=%.6f",
		data.IMEI, data.EventType, data.Latitude, data.Longitude)

	return nil
}

// WriteGPSData 写入 GPS 定位数据（专用方法）
func (s *InfluxDBService) WriteGPSData(imei string, lat, lon float64, timestamp time.Time) error {
	if s == nil || s.writeAPI == nil {
		return fmt.Errorf("InfluxDB 客户端未初始化")
	}

	point := influxdb2.NewPoint(
		"gps_data",
		map[string]string{
			"imei": imei,
		},
		map[string]interface{}{
			"lat": lat,
			"lon": lon,
		},
		timestamp,
	)

	s.writeAPI.WritePoint(point)
	return nil
}

// WriteHealthData 写入健康数据（心率等）
func (s *InfluxDBService) WriteHealthData(imei string, heartRate, battery int, timestamp time.Time) error {
	if s == nil || s.writeAPI == nil {
		return fmt.Errorf("InfluxDB 客户端未初始化")
	}

	point := influxdb2.NewPoint(
		"health_data",
		map[string]string{
			"imei": imei,
		},
		map[string]interface{}{
			"heart_rate": heartRate,
			"battery":    battery,
		},
		timestamp,
	)

	s.writeAPI.WritePoint(point)
	return nil
}

// Flush 强制刷新缓冲区
func (s *InfluxDBService) Flush() {
	if s != nil && s.writeAPI != nil {
		s.writeAPI.Flush()
	}
}

// HeartRatePoint 单条心率时序数据
type HeartRatePoint struct {
	Time      time.Time `json:"time"`
	HeartRate int       `json:"heart_rate"`
	Battery   int       `json:"battery"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
}

// QueryHeartRateHistory 查询指定设备的心率历史数据（最近 N 小时）
func (s *InfluxDBService) QueryHeartRateHistory(imei string, hours int) ([]HeartRatePoint, error) {
	if s == nil || s.queryAPI == nil {
		return nil, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	// 计算时间范围
	end := time.Now()
	start := end.Add(-time.Duration(hours) * time.Hour)

	// Flux 查询
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "heart_rate")
		|> aggregateWindow(every: 5m, fn: mean, createEmpty: false)
		|> sort(columns: ["_time"])
	`, s.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer result.Close()

	var points []HeartRatePoint
	for result.Next() {
		record := result.Record()
		hr := int(record.Value().(float64))
		points = append(points, HeartRatePoint{
			Time:      record.Time(),
			HeartRate: hr,
		})
	}

	return points, nil
}

// Close 关闭 InfluxDB 连接
func (s *InfluxDBService) Close() {
	if s != nil && s.client != nil {
		s.Flush()
		s.client.Close()
		log.Printf("[INFO] InfluxDB 连接已关闭")
	}
}
