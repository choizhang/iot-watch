package services

import (
	"elder-guard-iot/models"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDevicePayload 解析手环设备报文
// 支持标准长者手环的 ASCII 报文格式
// 报文示例：*HQ,1234567890,SOS,120000,A,2230.1234,N,11400.5678,E,85,90#
// 格式说明：
//   *HQ       - 协议头
//   1234567890 - IMEI (10位)
//   SOS       - 消息类型 (HEARTBEAT/SOS/FALL/LOCATION/HEART_RATE)
//   120000    - 时间戳 (HHMMSS)
//   A         - 定位状态 (A=有效定位, V=无效)
//   2230.1234 - 纬度 (DDMM.MMMM)
//   N         - 纬度方向 (N/S)
//   11400.5678 - 经度 (DDDMM.MMMM)
//   E         - 经度方向 (E/W)
//   85        - 心率值
//   90        - 电量百分比
func ParseDevicePayload(payloadStr string) (*models.DevicePayload, error) {
	// 去除首尾空白字符
	payloadStr = strings.TrimSpace(payloadStr)
	if payloadStr == "" {
		return nil, fmt.Errorf("空报文")
	}

	// 检查协议头和尾缀
	if !strings.HasPrefix(payloadStr, "*HQ,") || !strings.HasSuffix(payloadStr, "#") {
		return nil, fmt.Errorf("无效报文格式，缺少协议头或尾缀")
	}

	// 去除首尾的协议标识
	payloadStr = strings.TrimPrefix(payloadStr, "*HQ,")
	payloadStr = strings.TrimSuffix(payloadStr, "#")

	// 按逗号分割
	parts := strings.Split(payloadStr, ",")
	if len(parts) < 10 {
		return nil, fmt.Errorf("报文字段不足，期望至少10个字段，实际: %d", len(parts))
	}

	result := &models.DevicePayload{
		RawPayload: payloadStr,
		Timestamp:  time.Now(),
	}

	// 解析 IMEI (第1个字段，10位数字)
	result.IMEI = parts[0]
	if !regexp.MustCompile(`^\d{10,15}$`).MatchString(result.IMEI) {
		return nil, fmt.Errorf("无效的 IMEI: %s", result.IMEI)
	}

	// 解析消息类型 (第2个字段)
	result.MsgType = strings.ToUpper(parts[1])
	switch result.MsgType {
	case "SOS", "FALL", "LOCATION", "HEARTBEAT", "HEART_RATE":
		// 有效的消息类型
	case "HQ", "GPRS", "TCP":
		// 某些设备使用这些作为心跳标识
		result.MsgType = models.MsgTypeHeartbeat
	default:
		// 尝试根据上下文判断
		result.MsgType = models.MsgTypeHeartbeat
	}

	// 查找 "A" 的位置判断 GPS 定位状态是否有效
	gpsValid := false
	gpsIndex := -1
	for i := 2; i < len(parts) && i < 6; i++ {
		if strings.ToUpper(parts[i]) == "A" {
			gpsValid = true
			gpsIndex = i
			break
		}
	}

	// 解析经纬度 (如果定位有效)
	if gpsValid && gpsIndex > 0 && len(parts) >= gpsIndex+5 {
		if lat, err := parseCoordinate(parts[gpsIndex+1], parts[gpsIndex+2]); err == nil {
			result.Latitude = lat
		}
		if lng, err := parseCoordinate(parts[gpsIndex+3], parts[gpsIndex+4]); err == nil {
			result.Longitude = lng
		}
	}

	// 解析心率 (倒数第2个字段或特定字段)
	if len(parts) >= 2 {
		// 尝试从不同位置解析心率
		for i := len(parts) - 2; i >= 0; i-- {
			if hr, err := strconv.Atoi(parts[i]); err == nil && hr > 0 && hr < 300 {
				result.HeartRate = hr
				break
			}
		}
	}

	// 解析电量 (最后一个字段)
	if len(parts) >= 2 {
		if batt, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			if batt >= 0 && batt <= 100 {
				result.Battery = batt
			} else if batt > 100 {
				// 某些设备使用 0-255 或其他范围
				result.Battery = (batt * 100) / 255
			}
		}
	}

	return result, nil
}

// parseCoordinate 解析经纬度坐标
// 输入格式：DDMM.MMMM (度分格式)
// 返回十进制度数
func parseCoordinate(value, direction string) (float64, error) {
	// 解析度分格式
	coord, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	// 提取度和分
	degrees := int(coord / 100)
	minutes := coord - float64(degrees*100)

	// 转换为十进制度
	decimal := float64(degrees) + minutes/60.0

	// 根据方向调整正负
	direction = strings.ToUpper(direction)
	if direction == "S" || direction == "W" {
		decimal = -decimal
	}

	return decimal, nil
}

// ValidatePayload 验证报文格式是否合法
func ValidatePayload(payloadStr string) bool {
	_, err := ParseDevicePayload(payloadStr)
	return err == nil
}
