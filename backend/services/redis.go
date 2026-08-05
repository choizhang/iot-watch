package services

import (
	"context"
	"elder-guard-iot/config"
	"elder-guard-iot/models"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisService Redis 服务
type RedisService struct {
	client *redis.Client
	ctx    context.Context
}

// redisClient 全局 Redis 客户端
var redisClient *RedisService

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.Config) error {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()

	// 测试连接
	if _, err := client.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis 连接失败: %w", err)
	}

	redisClient = &RedisService{
		client: client,
		ctx:    ctx,
	}

	log.Printf("[INFO] Redis 连接成功: %s", cfg.GetRedisAddr())
	return nil
}

// GetRedisClient 获取 Redis 客户端实例
func GetRedisClient() *RedisService {
	return redisClient
}

// SetDeviceState 更新设备状态到 Redis
func (r *RedisService) SetDeviceState(state *models.DeviceState) error {
	keyState := models.RedisKeyDeviceState(state.IMEI)
	keyStatus := "device:status:" + state.IMEI

	// 序列化状态为 JSON String
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("序列化设备状态失败: %w", err)
	}

	ttl := config.GetDeviceStateTTL()
	if err := r.client.Set(r.ctx, keyState, data, ttl).Err(); err != nil {
		return fmt.Errorf("写入 Redis 失败: %w", err)
	}

	// 同时写入 Hash 格式 (device:status:{imei}) 供 API 和 WebSocket 统一读取
	hashData := map[string]interface{}{
		"status":           state.Status,
		"event_type":       state.MsgType,
		"lat":              fmt.Sprintf("%.6f", state.Latitude),
		"lon":              fmt.Sprintf("%.6f", state.Longitude),
		"heart_rate":       state.HeartRate,
		"battery":          state.Battery,
		"bp":               state.BloodPressure,
		"spo2":             state.SpO2,
		"hrv":              state.HRV,
		"steps":            state.Steps,
		"hr_updated_at":    state.HRUpdatedAt,
		"bp_updated_at":    state.BPUpdatedAt,
		"spo2_updated_at":  state.SpO2UpdatedAt,
		"hrv_updated_at":   state.HRVUpdatedAt,
		"steps_updated_at": state.StepsUpdatedAt,
		"updated_at":       time.Now().Unix(),
	}
	r.client.HMSet(r.ctx, keyStatus, hashData)
	r.client.Expire(r.ctx, keyStatus, ttl)

	log.Printf("[DEBUG] 设备状态已更新: %s & %s, TTL: %v", keyState, keyStatus, ttl)
	return nil
}

// GetDeviceState 从 Redis 获取设备状态
// 兼容两种存储格式：JSON String (旧) 和 Hash (新)
func (r *RedisService) GetDeviceState(imei string) (*models.DeviceState, error) {
	// 兼容两套 key: device:status:{imei} (TCP Server 实际写入) 和 device:state:{imei} (model 定义)
	keys := []string{
		"device:status:" + imei,
		models.RedisKeyDeviceState(imei),
	}

	for _, key := range keys {
		// 先尝试读取为 Hash 格式 (TCP Server 写入的格式)
		hashData, err := r.client.HGetAll(r.ctx, key).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("读取 Redis Hash 失败: %w", err)
		}

		// 如果 Hash 有数据，解析 Hash
		if len(hashData) > 0 {
			state := &models.DeviceState{
				IMEI:           imei,
				MsgType:        hashData["event_type"],
				HeartRate:      parseIntSafe(hashData["heart_rate"]),
				Battery:        parseIntSafe(hashData["battery"]),
				Latitude:       parseFloatSafe(hashData["lat"]),
				Longitude:      parseFloatSafe(hashData["lon"]),
				BloodPressure:  hashData["bp"],
				SpO2:           parseIntSafe(hashData["spo2"]),
				HRV:            parseIntSafe(hashData["hrv"]),
				Steps:          parseIntSafe(hashData["steps"]),
				HRUpdatedAt:    parseInt64Safe(hashData["hr_updated_at"]),
				BPUpdatedAt:    parseInt64Safe(hashData["bp_updated_at"]),
				SpO2UpdatedAt:  parseInt64Safe(hashData["spo2_updated_at"]),
				HRVUpdatedAt:   parseInt64Safe(hashData["hrv_updated_at"]),
				FixMode:        hashData["event_type"],
				StepsUpdatedAt: parseInt64Safe(hashData["steps_updated_at"]),
			}

			// 解析状态
			if status, ok := hashData["status"]; ok && status != "" {
				state.Status = status
			} else {
				state.Status = "online" // Hash 里有数据说明最近上报过，默认在线
			}

			// 解析时间
			if ts := hashData["updated_at"]; ts != "" {
				if t, err := strconv.ParseInt(ts, 10, 64); err == nil {
					state.LastUpdate = time.Unix(t, 0)
				} else {
					state.LastUpdate = time.Now()
				}
			} else {
				state.LastUpdate = time.Now()
			}

			return state, nil
		}

		// Hash 为空，尝试读取 JSON String 格式
		data, err := r.client.Get(r.ctx, key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue // 尝试下一个 key
			}
			return nil, fmt.Errorf("读取 Redis 失败: %w", err)
		}

		var state models.DeviceState
		if err := json.Unmarshal(data, &state); err != nil {
			continue // 尝试下一个 key
		}

		return &state, nil
	}

	return nil, nil // 所有 key 都不存在
}

// parseIntSafe 安全解析 int
func parseIntSafe(s string) int {
	if s == "" {
		return 0
	}
	var i int
	_, _ = fmt.Sscanf(s, "%d", &i)
	return i
}

// parseFloatSafe 安全解析 float64
func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

// SetSOSAlert 设置 SOS 告警标识
func (r *RedisService) SetSOSAlert(imei string) error {
	key := models.RedisKeyAlertSOS(imei)
	ttl := config.GetAlertTTL()

	alertData := map[string]interface{}{
		"imei":      imei,
		"alert_type": "SOS",
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(alertData)
	if err != nil {
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	if err := r.client.Set(r.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("写入 SOS 告警失败: %w", err)
	}

	log.Printf("[CRITICAL ALERT] SOS 告警已写入 Redis: %s, TTL: %v", imei, ttl)
	return nil
}

// SetFallAlert 设置跌倒告警标识
func (r *RedisService) SetFallAlert(imei string) error {
	key := models.RedisKeyAlertFall(imei)
	ttl := config.GetAlertTTL()

	alertData := map[string]interface{}{
		"imei":       imei,
		"alert_type": "FALL",
		"timestamp":  time.Now().Unix(),
	}

	data, err := json.Marshal(alertData)
	if err != nil {
		return fmt.Errorf("序列化告警数据失败: %w", err)
	}

	if err := r.client.Set(r.ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("写入跌倒告警失败: %w", err)
	}

	log.Printf("[CRITICAL ALERT] 跌倒告警已写入 Redis: %s, TTL: %v", imei, ttl)
	return nil
}

// GetAlert 获取告警状态
func (r *RedisService) GetAlert(imei string) (sosAlert, fallAlert map[string]interface{}, err error) {
	// 获取 SOS 告警
	sosData, err := r.client.Get(r.ctx, models.RedisKeyAlertSOS(imei)).Bytes()
	if err != nil && err != redis.Nil {
		return nil, nil, fmt.Errorf("读取 SOS 告警失败: %w", err)
	}
	if sosData != nil {
		json.Unmarshal(sosData, &sosAlert)
	}

	// 获取跌倒告警
	fallData, err := r.client.Get(r.ctx, models.RedisKeyAlertFall(imei)).Bytes()
	if err != nil && err != redis.Nil {
		return nil, nil, fmt.Errorf("读取跌倒告警失败: %w", err)
	}
	if fallData != nil {
		json.Unmarshal(fallData, &fallAlert)
	}

	return sosAlert, fallAlert, nil
}

// Close 关闭 Redis 连接
func (r *RedisService) Close() error {
	return r.client.Close()
}

// GetClient 获取原生 redis.Client 客户端（供其他模块使用）
func (r *RedisService) GetClient() *redis.Client {
	return r.client
}

func parseInt64Safe(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
