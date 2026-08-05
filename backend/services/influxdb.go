package services

import (
	"context"
	"elder-guard-iot/config"
	"fmt"
	"log"
	"strings"
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
	IMEI          string
	EventType     string
	Latitude      float64
	Longitude     float64
	HeartRate     int
	Battery       int
	BloodPressure string
	SpO2          int
	HRV           int
	Steps         int
	SOSFlag       bool
	RawPayload    string
}

// WriteTelemetry 异步写入设备遥测数据到 InfluxDB
func (s *InfluxDBService) WriteTelemetry(data *TelemetryData) error {
	if s == nil || s.writeAPI == nil {
		return fmt.Errorf("InfluxDB 客户端未初始化")
	}

	fields := make(map[string]interface{})

	if data.Latitude != 0 || data.Longitude != 0 {
		fields["lat"] = data.Latitude
		fields["lon"] = data.Longitude
	}
	if data.Battery > 0 {
		fields["battery"] = data.Battery
	}
	if data.HeartRate > 0 {
		fields["heart_rate"] = data.HeartRate
	}
	if data.BloodPressure != "" {
		fields["bp"] = data.BloodPressure
	}
	if data.SpO2 > 0 {
		fields["spo2"] = data.SpO2
	}
	if data.HRV > 0 {
		fields["hrv"] = data.HRV
	}
	if data.Steps > 0 {
		fields["steps"] = data.Steps
	}
	if data.SOSFlag {
		fields["sos_flag"] = true
	}

	// 若没有有效指标字段，避免写入无数据的空数据点
	if len(fields) == 0 {
		return nil
	}

	// 创建数据点
	point := influxdb2.NewPoint(
		"device_telemetry", // Measurement
		map[string]string{
			"imei":       data.IMEI,
			"event_type": data.EventType,
		},
		fields,
		time.Now(),
	)

	// 异步写入并强制刷盘
	s.writeAPI.WritePoint(point)
	s.writeAPI.Flush()

	log.Printf("[DEBUG] InfluxDB 写入点: IMEI=%s, Type=%s, FieldsCount=%d",
		data.IMEI, data.EventType, len(fields))

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

// VitalsHistoryPointInDB InfluxDB 查到的单条体征历史数据点
type VitalsHistoryPointInDB struct {
	Time          time.Time
	HeartRate     int
	BloodPressure string
	SpO2          int
	HRV           int
	Steps         int
}

// QueryVitalsHistory 查询指定设备在 InfluxDB 中的体征历史数据（根据 24 小时标准时间槽分配真实实测点）
func (s *InfluxDBService) QueryVitalsHistory(imei string, hours int) ([]VitalsHistoryPointInDB, error) {
	if s == nil || s.queryAPI == nil {
		return nil, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	if hours <= 0 {
		hours = 24
	}

	now := time.Now()
	start := now.Add(-time.Duration(hours) * time.Hour)

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "heart_rate" or r["_field"] == "bp" or r["_field"] == "spo2" or r["_field"] == "hrv" or r["_field"] == "steps")
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> sort(columns: ["_time"])
	`, s.bucket, start.Format(time.RFC3339), now.Format(time.RFC3339), imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("InfluxDB 查询体征历史失败: %w", err)
	}
	defer result.Close()

	parseInt := func(val interface{}) int {
		if val == nil {
			return 0
		}
		switch v := val.(type) {
		case float64:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		default:
			return 0
		}
	}

	var rawPoints []VitalsHistoryPointInDB
	for result.Next() {
		rec := result.Record()

		hr := parseInt(rec.ValueByKey("heart_rate"))
		spo2 := parseInt(rec.ValueByKey("spo2"))
		hrv := parseInt(rec.ValueByKey("hrv"))
		steps := parseInt(rec.ValueByKey("steps"))

		bpStr := ""
		if bpVal, ok := rec.ValueByKey("bp").(string); ok {
			bpStr = bpVal
		}

		if hr <= 0 && bpStr == "" && spo2 <= 0 && hrv <= 0 && steps <= 0 {
			continue
		}

		rawPoints = append(rawPoints, VitalsHistoryPointInDB{
			Time:          rec.Time().Local(),
			HeartRate:     hr,
			BloodPressure: bpStr,
			SpO2:          spo2,
			HRV:           hrv,
			Steps:         steps,
		})
	}

	// 构建 hours 个标准时间槽（索引 0 代表 (hours-1) 小时前，索引 hours-1 代表当前时间槽）
	slots := make([]VitalsHistoryPointInDB, hours)
	for i := 0; i < hours; i++ {
		tSlot := now.Add(time.Duration(-(hours - 1 - i)) * time.Hour)
		slotPoint := VitalsHistoryPointInDB{
			Time: tSlot,
		}

		var latestTime time.Time
		for _, pt := range rawPoints {
			diff := tSlot.Sub(pt.Time)
			if diff < 0 {
				diff = -diff
			}
			if diff <= 30*time.Minute {
				if pt.HeartRate > 0 {
					slotPoint.HeartRate = pt.HeartRate
				}
				if pt.BloodPressure != "" {
					slotPoint.BloodPressure = pt.BloodPressure
				}
				if pt.SpO2 > 0 {
					slotPoint.SpO2 = pt.SpO2
				}
				if pt.HRV > 0 {
					slotPoint.HRV = pt.HRV
				}
				if pt.Steps > 0 {
					slotPoint.Steps = pt.Steps
				}
				if pt.Time.After(latestTime) {
					latestTime = pt.Time
				}
			}
		}

		if !latestTime.IsZero() {
			slotPoint.Time = latestTime
		}

		slots[i] = slotPoint
	}

	return slots, nil
}

// QueryTodaySteps 查询设备今天（00:00:00 至当前时刻）的真实计步上报点（按上报精准秒级时间升序）
func (s *InfluxDBService) QueryTodaySteps(imei string) ([]VitalsHistoryPointInDB, error) {
	if s == nil || s.queryAPI == nil {
		return nil, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "steps")
		|> sort(columns: ["_time"])
	`, s.bucket, todayStart.Format(time.RFC3339), now.Format(time.RFC3339), imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("InfluxDB 查询今日步数失败: %w", err)
	}
	defer result.Close()

	parseInt := func(val interface{}) int {
		if val == nil {
			return 0
		}
		switch v := val.(type) {
		case float64:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		default:
			return 0
		}
	}

	var points []VitalsHistoryPointInDB
	for result.Next() {
		rec := result.Record()
		steps := parseInt(rec.Value())
		if steps > 0 {
			points = append(points, VitalsHistoryPointInDB{
				Time:  rec.Time().Local(),
				Steps: steps,
			})
		}
	}

	return points, nil
}

// TrajectoryPointInDB InfluxDB 轨迹点结构
type TrajectoryPointInDB struct {
	Time         time.Time
	Latitude     float64
	Longitude    float64
	EventType    string
	LocationType string
}

// QueryTrajectoryHistory 查询指定设备在 InfluxDB 中的历史轨迹定位点
func (s *InfluxDBService) QueryTrajectoryHistory(imei string, period string) ([]TrajectoryPointInDB, error) {
	if s == nil || s.queryAPI == nil {
		return nil, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	now := time.Now()
	var start, end time.Time
	if period == "yesterday" {
		start = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else if period == "3days" {
		start = now.AddDate(0, 0, -3)
		end = now
	} else { // "today"
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = now
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "lat" or r["_field"] == "lon")
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
		|> filter(fn: (r) => exists r["lat"] and exists r["lon"] and r["lat"] != 0.0 and r["lon"] != 0.0)
		|> sort(columns: ["_time"])
	`, s.bucket, start.Format(time.RFC3339), end.Format(time.RFC3339), imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("Flux 查询轨迹失败: %w", err)
	}
	defer result.Close()

	var points []TrajectoryPointInDB
	for result.Next() {
		rec := result.Record()
		latVal, okLat := rec.ValueByKey("lat").(float64)
		lonVal, okLon := rec.ValueByKey("lon").(float64)
		if !okLat || !okLon || (latVal == 0 && lonVal == 0) {
			continue
		}

		eventType := ""
		if ev, ok := rec.ValueByKey("event_type").(string); ok {
			eventType = ev
		}

		locType := "GPS"
		if strings.Contains(strings.ToUpper(eventType), "WIFI") {
			locType = "WIFI"
		} else if strings.Contains(strings.ToUpper(eventType), "LBS") {
			locType = "LBS"
		}

		points = append(points, TrajectoryPointInDB{
			Time:         rec.Time(),
			Latitude:     latVal,
			Longitude:    lonVal,
			EventType:    eventType,
			LocationType: locType,
		})
	}

	return points, nil
}

// LatestVitalsData InfluxDB 查询到的最新体征数据
type LatestVitalsData struct {
	HeartRate     int       `json:"heart_rate"`
	BloodPressure string    `json:"bp"`
	SpO2          int       `json:"spo2"`
	HRV           int       `json:"hrv"`
	Steps         int       `json:"steps"`
	SpO2Time      time.Time `json:"spo2_time"`
	LastTime      time.Time `json:"last_time"`
}

// QueryLatestVitals 查询 InfluxDB 中设备最近 24 小时的最新各项体征数据
func (s *InfluxDBService) QueryLatestVitals(imei string) (*LatestVitalsData, error) {
	if s == nil || s.queryAPI == nil {
		return nil, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -24h)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "spo2" or r["_field"] == "heart_rate" or r["_field"] == "bp" or r["_field"] == "hrv" or r["_field"] == "steps")
		|> last()
	`, s.bucket, imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	vitals := &LatestVitalsData{}
	for result.Next() {
		rec := result.Record()
		field := rec.Field()
		val := rec.Value()
		t := rec.Time()

		if t.After(vitals.LastTime) {
			vitals.LastTime = t
		}

		switch field {
		case "spo2":
			if floatVal, ok := val.(float64); ok && floatVal > 0 {
				vitals.SpO2 = int(floatVal)
				vitals.SpO2Time = t
			}
		case "heart_rate":
			if floatVal, ok := val.(float64); ok && floatVal > 0 {
				vitals.HeartRate = int(floatVal)
			}
		case "bp":
			if strVal, ok := val.(string); ok && strVal != "" {
				vitals.BloodPressure = strVal
			}
		case "hrv":
			if floatVal, ok := val.(float64); ok && floatVal > 0 {
				vitals.HRV = int(floatVal)
			}
		case "steps":
			if floatVal, ok := val.(float64); ok && floatVal > 0 {
				vitals.Steps = int(floatVal)
			}
		}
	}

	return vitals, nil
}

// QueryLatestSpO2 查询 InfluxDB 中设备最新的血氧上报数据
func (s *InfluxDBService) QueryLatestSpO2(imei string) (int, time.Time, error) {
	if s == nil || s.queryAPI == nil {
		return 0, time.Time{}, fmt.Errorf("InfluxDB 客户端未初始化")
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -24h)
		|> filter(fn: (r) => r["_measurement"] == "device_telemetry")
		|> filter(fn: (r) => r["imei"] == "%s")
		|> filter(fn: (r) => r["_field"] == "spo2")
		|> last()
	`, s.bucket, imei)

	result, err := s.queryAPI.Query(context.Background(), query)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer result.Close()

	for result.Next() {
		rec := result.Record()
		if val, ok := rec.Value().(float64); ok && val > 0 {
			return int(val), rec.Time(), nil
		}
	}
	return 0, time.Time{}, fmt.Errorf("无血氧记录")
}

// Close 关闭 InfluxDB 连接
func (s *InfluxDBService) Close() {
	if s != nil && s.client != nil {
		s.Flush()
		s.client.Close()
		log.Printf("[INFO] InfluxDB 连接已关闭")
	}
}
