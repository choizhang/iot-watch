package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

// 一些常见的 IMEI 前缀（演示用，非真实设备）
var imeis = []string{
	"351086239665254",
	"1234567890",
	"13800138000",
	"867123456789012",
	"359000000000017",
	"868811000000015",
	"352099001761481",
}

// 多个常用坐标点（香港、深圳、北京等）- 模拟不同设备在不同位置
var locations = []struct {
	Name  string
	Lat   string // 度分格式
	Lon   string // 度分格式
	NS    string
	EW    string
}{
	// 香港葵涌
	{"香港葵涌", "2234.5678", "11405.6789", "N", "E"},
	// 深圳南山
	{"深圳南山", "2238.1234", "11395.4567", "N", "E"},
	// 北京天安门
	{"北京天安门", "3954.5678", "11623.4567", "N", "E"},
	// 上海外滩
	{"上海外滩", "3114.5678", "12128.4567", "N", "E"},
	// 广州天河
	{"广州天河", "2307.5678", "11321.4567", "N", "E"},
}

// 事件类型
var eventTypes = []string{"V1", "V1", "V1", "SOS"} // V1 概率更大，模拟正常心跳

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 默认发送一次
	count := 1
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &count)
	}

	for i := 0; i < count; i++ {
		if err := sendRandom(r); err != nil {
			fmt.Println("✗ 发送失败:", err)
		}
		if i < count-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// sendRandom 发送一条随机设备报文
func sendRandom(r *rand.Rand) error {
	// 随机选一个设备
	imei := imeis[r.Intn(len(imeis))]

	// 随机选一个位置
	loc := locations[r.Intn(len(locations))]

	// 随机事件类型
	event := eventTypes[r.Intn(len(eventTypes))]

	// 随机时间 (HHMMSS)
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	timeStr := fmt.Sprintf("%02d%02d%02d", hour, minute, second)

	// 随机心率 50-120
	heartRate := 50 + r.Intn(70)

	// 随机电量 0-100
	battery := r.Intn(101)

	// 随机日期 (DDMMYY)
	date := fmt.Sprintf("%02d%02d%02d",
		1+r.Intn(28),   // 日
		1+r.Intn(12),   // 月
		20+r.Intn(10))  // 年(2020-2029)

	// 状态位: A=有效定位 V=无效定位
	status := "A"
	if event == "SOS" {
		status = "A"
	}

	// 构造报文
	var payload string
	if event == "SOS" {
		// SOS 报文
		payload = fmt.Sprintf("*HQ,%s,SOS#", imei)
	} else {
		// V1 定位报文
		payload = fmt.Sprintf("*HQ,%s,V1,%s,%s,%s,%s,%s,%d,%d,%s,FFFFBBFF#",
			imei, timeStr, status, loc.Lat, loc.NS, loc.Lon, loc.EW,
			heartRate, battery, date)
	}

	// 连接 TCP 服务器
	conn, err := net.Dial("tcp", "127.0.0.1:5007")
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(payload))
	if err != nil {
		return fmt.Errorf("写入失败: %w", err)
	}

	fmt.Printf("✓ [%s] IMEI=%s, Type=%s, HR=%d, Batt=%d%%, 位置=%s\n",
		time.Now().Format("15:04:05"), imei, event, heartRate, battery, loc.Name)
	return nil
}
