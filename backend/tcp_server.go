package main

import (
	"bufio"
	"context"
	"elder-guard-iot/config"
	"elder-guard-iot/models"
	"elder-guard-iot/services"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter IP 频率与并发限制器
type IPRateLimiter struct {
	mu             sync.Mutex
	activeConns    map[string]int        // 每个 IP 当前活动连接数
	connTimestamps map[string][]time.Time // 每个 IP 建立连接的时间戳列表
	maxConns       int                   // 单 IP 最大并发数（默认 50）
	maxRate        int                   // 单 IP 每秒新建连接数上限（默认 10）
}

// NewIPRateLimiter 创建 IP 限制器
func NewIPRateLimiter(maxConns, maxRate int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		activeConns:    make(map[string]int),
		connTimestamps: make(map[string][]time.Time),
		maxConns:       maxConns,
		maxRate:        maxRate,
	}
	go limiter.cleanupLoop()
	return limiter
}

// AllowNewConnection 检查是否允许新连接建立
func (l *IPRateLimiter) AllowNewConnection(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.activeConns[ip] >= l.maxConns {
		return false
	}

	now := time.Now()
	cutoff := now.Add(-1 * time.Second)
	timestamps := l.connTimestamps[ip]
	validIndex := 0
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			break
		}
		validIndex++
	}
	l.connTimestamps[ip] = timestamps[validIndex:]

	if len(l.connTimestamps[ip]) >= l.maxRate {
		return false
	}

	l.activeConns[ip]++
	l.connTimestamps[ip] = append(l.connTimestamps[ip], now)
	return true
}

// ReleaseConnection 释放连接计数
func (l *IPRateLimiter) ReleaseConnection(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if count, ok := l.activeConns[ip]; ok {
		if count <= 1 {
			delete(l.activeConns, ip)
		} else {
			l.activeConns[ip]--
		}
	}
}

// cleanupLoop 定期清理过期的时间戳记录
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-1 * time.Second)
		for ip, timestamps := range l.connTimestamps {
			validIndex := 0
			for _, ts := range timestamps {
				if ts.After(cutoff) {
					break
				}
				validIndex++
			}
			if validIndex >= len(timestamps) {
				delete(l.connTimestamps, ip)
			} else {
				l.connTimestamps[ip] = timestamps[validIndex:]
			}
		}
		l.mu.Unlock()
	}
}

// TCPServer TCP 服务器
type TCPServer struct {
	addr      string           // 监听地址
	listener  net.Listener     // 监听器
	wg        sync.WaitGroup   // WaitGroup 等待所有连接处理结束
	quit      chan struct{}    // 退出信号
	redisSvc  *services.RedisService
	mysqlSvc  *services.MySQLService
	influxSvc *services.InfluxDBService
	ipLimiter *IPRateLimiter   // IP 频控限制器
	sessions  sync.Map         // 在线设备 TCP 会话池 (IMEI string -> net.Conn)
}

// NewTCPServer 创建 TCP 服务器实例
func NewTCPServer(redisSvc *services.RedisService, mysqlSvc *services.MySQLService, influxSvc *services.InfluxDBService) *TCPServer {
	cfg := config.GlobalConfig
	return &TCPServer{
		addr:      fmt.Sprintf("%s:%s", cfg.Host, "5007"), // 使用 5007 避免与 EMQX 冲突
		quit:      make(chan struct{}),
		redisSvc:  redisSvc,
		mysqlSvc:  mysqlSvc,
		influxSvc: influxSvc,
		ipLimiter: NewIPRateLimiter(50, 10),
	}
}

// SendDownstreamCommand 向在线设备下发 TCP 控制指令
func (s *TCPServer) SendDownstreamCommand(imei string, command string) error {
	val, ok := s.sessions.Load(imei)
	if !ok {
		return fmt.Errorf("设备 %s 当前没有活跃的 TCP 长连接", imei)
	}

	conn, ok := val.(net.Conn)
	if !ok || conn == nil {
		s.sessions.Delete(imei)
		return fmt.Errorf("设备 %s 的 TCP 连接句柄已失效", imei)
	}

	var rawCmd string
	if strings.HasPrefix(command, "*") || strings.HasPrefix(command, "[") {
		rawCmd = command + "\n"
	} else {
		rawCmd = fmt.Sprintf("*HQ,%s,%s#\n", imei, command)
	}

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Write([]byte(rawCmd))
	if err != nil {
		s.sessions.Delete(imei)
		return fmt.Errorf("向设备 %s 下发指令失败: %w", imei, err)
	}

	log.Printf("[TCP SEND] [%s] 成功下发控制指令: %s", imei, strings.TrimSpace(rawCmd))
	return nil
}

// Start 启动 TCP 服务器
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("TCP 监听失败: %w", err)
	}
	s.listener = listener

	log.Printf("[INFO] TCP 服务器启动成功，监听端口: %s (安全防护 & 双向指令解耦已开启)", s.addr)
	go s.acceptLoop()

	return nil
}

// acceptLoop 接受连接的主循环
func (s *TCPServer) acceptLoop() {
	for {
		select {
		case <-s.quit:
			log.Printf("[INFO] TCP 服务器停止接受新连接")
			return
		default:
		}

		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-s.quit:
				return
			default:
			}
			log.Printf("[WARN] 接受连接失败: %v", err)
			continue
		}

		clientIP, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		if splitErr != nil {
			clientIP = conn.RemoteAddr().String()
		}

		if !s.ipLimiter.AllowNewConnection(clientIP) {
			log.Printf("[WARN] [安全拦截] 客户端 IP %s 超过连接数或频率限制，拒绝建立 TCP 连接", clientIP)
			conn.Close()
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn, clientIP)
	}
}

// handleConnection 处理单个 TCP 连接
func (s *TCPServer) handleConnection(conn net.Conn, clientIP string) {
	defer s.wg.Done()
	defer s.ipLimiter.ReleaseConnection(clientIP)

	addr := conn.RemoteAddr().String()
	log.Printf("[INFO] 新连接建立: %s", addr)

	var boundIMEI string
	defer func() {
		if boundIMEI != "" {
			s.sessions.Delete(boundIMEI)
			log.Printf("[INFO] [%s] 设备离线，解绑 TCP 会话: IMEI=%s", addr, boundIMEI)
		}
	}()

	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	scanner.Split(s.splitByDelim)

	var consecutiveErrors int
	var packetTimestamps []time.Time

	for scanner.Scan() {
		payload := scanner.Text()
		if payload == "" {
			continue
		}

		now := time.Now()
		cutoff := now.Add(-1 * time.Second)
		validIdx := 0
		for _, ts := range packetTimestamps {
			if ts.After(cutoff) {
				break
			}
			validIdx++
		}
		packetTimestamps = packetTimestamps[validIdx:]

		if len(packetTimestamps) >= 5 {
			log.Printf("[WARN] [安全限流] [%s] 报文发送过频 (%d条/秒)，触发限流退避", addr, len(packetTimestamps)+1)
			conn.Write([]byte("ERROR:RATE_LIMITED\n"))
			time.Sleep(200 * time.Millisecond)
			continue
		}
		packetTimestamps = append(packetTimestamps, now)

		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		success := s.processPayload(conn, addr, payload, &boundIMEI)
		if !success {
			consecutiveErrors++
			if consecutiveErrors >= 3 {
				log.Printf("[ERROR] [安全熔断] [%s] 连续 %d 次报文格式解析失败，强制断开连接", addr, consecutiveErrors)
				conn.Write([]byte("ERROR:TOO_MANY_ERRORS\n"))
				break
			}
		} else {
			consecutiveErrors = 0
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ERROR] 读取数据异常 [%s]: %v", addr, err)
	}

	log.Printf("[INFO] 连接关闭: %s", addr)
	conn.Close()
}

// splitByDelim 自定义分隔符，按 '#' 拆包
func (s *TCPServer) splitByDelim(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '#' {
			return i + 1, data[:i], nil
		}
	}

	if atEOF && len(data) > 0 {
		return len(data), nil, nil
	}
	return 0, nil, nil
}

// processPayload 处理接收到的报文，返回是否成功解析格式
func (s *TCPServer) processPayload(conn net.Conn, clientAddr string, payload string, boundIMEI *string) bool {
	log.Printf("[TCP RECEIVE] [%s] 原始报文: %s", clientAddr, payload)

	parsedData, err := parseTCPPayload(payload)
	if err != nil {
		log.Printf("[ERROR] [%s] 报文解析失败: %v", clientAddr, err)
		conn.Write([]byte("ERROR:PARSE_FAILED\n"))
		return false
	}

	log.Printf("[TCP PARSED] IMEI=%s, Type=%s, Lat=%.6f, Lon=%.6f, HR=%d, Batt=%d%%",
		parsedData.IMEI, parsedData.EventType,
		parsedData.Latitude, parsedData.Longitude,
		parsedData.HeartRate, parsedData.Battery)

	// 绑定 TCP 会话
	if boundIMEI != nil {
		if *boundIMEI == "" {
			*boundIMEI = parsedData.IMEI
			s.sessions.Store(parsedData.IMEI, conn)
			log.Printf("[INFO] [%s] 成功绑定设备 TCP 会话句柄: IMEI=%s", clientAddr, parsedData.IMEI)
		} else if *boundIMEI != parsedData.IMEI {
			// 同连接变更 IMEI
			s.sessions.Delete(*boundIMEI)
			*boundIMEI = parsedData.IMEI
			s.sessions.Store(parsedData.IMEI, conn)
		}
	}

	// 检查设备 IMEI 注册与合法性
	if !s.isDeviceRegistered(parsedData.IMEI) {
		log.Printf("[WARN] [安全拦截] [%s] 未注册的非法设备尝试上发数据: IMEI=%s", clientAddr, parsedData.IMEI)
		conn.Write([]byte("ERROR:UNREGISTERED_DEVICE\n"))
		return true
	}

	// 处理 SOS 告警
	if parsedData.EventType == models.MsgTypeSOS {
		s.handleSOSAlert(parsedData)
	}

	// 写入 Redis
	if err := s.writeToRedis(parsedData); err != nil {
		log.Printf("[ERROR] [%s] Redis 写入失败: %v", clientAddr, err)
	}

	// 写入 InfluxDB
	if err := s.writeToInfluxDB(parsedData); err != nil {
		log.Printf("[ERROR] [%s] InfluxDB 写入失败: %v", clientAddr, err)
	}

	// 更新 MySQL 中的最后心跳时间与状态
	s.mysqlSvc.UpdateDeviceStatus(parsedData.IMEI, "online", time.Now(),
		parsedData.Latitude, parsedData.Longitude, parsedData.HeartRate, parsedData.Battery)

	conn.Write([]byte("OK\n"))
	return true
}

// isDeviceRegistered 校验设备 IMEI 是否已注册
func (s *TCPServer) isDeviceRegistered(imei string) bool {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("device:registered:%s", imei)

	if val, err := s.redisSvc.GetClient().Get(ctx, cacheKey).Result(); err == nil {
		return val == "1"
	}

	var device models.Device
	err := s.mysqlSvc.GetDB().Where("imei = ?", imei).First(&device).Error
	if err == nil {
		s.redisSvc.GetClient().Set(ctx, cacheKey, "1", 10*time.Minute)
		return true
	}

	s.redisSvc.GetClient().Set(ctx, cacheKey, "0", 1*time.Minute)
	return false
}

// ParsedDeviceData 解析后的设备数据
type ParsedDeviceData struct {
	IMEI       string
	EventType  string
	Latitude   float64
	Longitude  float64
	HeartRate  int
	Battery    int
	SOSFlag    bool
	RawPayload string
	Timestamp  time.Time
}

// IsAlert 判断是否为告警类型消息
func (p *ParsedDeviceData) IsAlert() bool {
	return p.EventType == models.MsgTypeSOS || p.EventType == models.MsgTypeFall
}

// parseTCPPayload 解析 TCP 报文
func parseTCPPayload(payload string) (*ParsedDeviceData, error) {
	payload = strings.TrimSpace(payload)

	result := &ParsedDeviceData{
		RawPayload: payload,
		Timestamp:  time.Now(),
	}

	if !strings.HasPrefix(payload, "*HQ,") {
		return nil, fmt.Errorf("无效报文格式")
	}

	payload = strings.TrimPrefix(payload, "*HQ,")
	payload = strings.TrimSuffix(payload, "#")

	parts := strings.Split(payload, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("报文字段不足")
	}

	result.IMEI = parts[0]
	if len(result.IMEI) < 10 || len(result.IMEI) > 15 {
		return nil, fmt.Errorf("无效 IMEI: %s", result.IMEI)
	}

	eventType := strings.ToUpper(parts[1])
	result.EventType = eventType

	if len(parts) >= 8 && strings.ToUpper(parts[3]) == "A" {
		if lat, err := parseCoordinate(parts[4], parts[5]); err == nil {
			result.Latitude = lat
		}
		if lng, err := parseCoordinate(parts[6], parts[7]); err == nil {
			result.Longitude = lng
		}
	}

	if len(parts) >= 10 {
		if hr, err := parseInt(parts[8]); err == nil {
			result.HeartRate = hr
		}
		if batt, err := parseInt(parts[9]); err == nil {
			if batt > 100 {
				result.Battery = (batt * 100) / 255
			} else {
				result.Battery = batt
			}
		}
	}

	result.SOSFlag = result.EventType == "SOS"

	return result, nil
}

// parseCoordinate 解析经纬度坐标
func parseCoordinate(value, direction string) (float64, error) {
	coord, err := parseFloat(value)
	if err != nil {
		return 0, err
	}
	degrees := int(coord / 100)
	minutes := coord - float64(degrees*100)
	decimal := float64(degrees) + minutes/60.0
	if direction == "S" || direction == "W" {
		decimal = -decimal
	}
	return decimal, nil
}

// parseFloat 安全解析浮点数
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// parseInt 安全解析整数
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// handleSOSAlert 处理 SOS 告警
func (s *TCPServer) handleSOSAlert(data *ParsedDeviceData) {
	alertType := data.EventType
	log.Printf("\n"+
		"╔══════════════════════════════════════════════════════════════╗\n"+
		"║  [CRITICAL ALERT] 收到长者 %s 告警！设备号：%-15s ║\n"+
		"╠══════════════════════════════════════════════════════════════╣\n"+
		"║  经度：%.6f                                           ║\n"+
		"║  纬度：%.6f                                           ║\n"+
		"║  心率：%-3d bpm                                        ║\n"+
		"║  时间：%s                                           ║\n"+
		"╚══════════════════════════════════════════════════════════════╝\n",
		alertType, data.IMEI, data.Longitude, data.Latitude, data.HeartRate,
		data.Timestamp.Format("2006-01-02 15:04:05"))

	ctx := context.Background()

	deviceKey := fmt.Sprintf("device:status:%s", data.IMEI)
	s.redisSvc.GetClient().HSet(ctx, deviceKey, map[string]interface{}{
		"status":     "sos_alert",
		"alert_type": alertType,
		"updated_at": time.Now().Unix(),
	})
	sosKey := fmt.Sprintf("alert:%s:%s", strings.ToLower(alertType), data.IMEI)
	s.redisSvc.GetClient().Set(ctx, sosKey, "1", 30*time.Minute)

	if s.influxSvc != nil {
		s.influxSvc.WriteTelemetry(&services.TelemetryData{
			IMEI:       data.IMEI,
			EventType:  alertType,
			Latitude:   data.Latitude,
			Longitude:  data.Longitude,
			HeartRate:  data.HeartRate,
			Battery:    data.Battery,
			SOSFlag:    true,
			RawPayload: data.RawPayload,
		})
	}

	imeiSuffix := data.IMEI
	if len(imeiSuffix) > 4 {
		imeiSuffix = imeiSuffix[len(imeiSuffix)-4:]
	}
	alarmID := fmt.Sprintf("%s-%s-%s",
		strings.ToUpper(alertType),
		data.Timestamp.Format("20060102-150405"),
		imeiSuffix)

	order := &models.AlarmOrder{
		AlarmID:     alarmID,
		DeviceIMEI:  data.IMEI,
		AlertType:   alertType,
		TriggerTime: data.Timestamp,
		Latitude:    data.Latitude,
		Longitude:   data.Longitude,
		HeartRate:   data.HeartRate,
		Status:      models.AlarmStatusUnhandeld,
		CreatedAt:   time.Now(),
	}

	if err := s.mysqlSvc.SaveAlarmOrder(order); err != nil {
		log.Printf("[ERROR] [MySQL] 保存告警工单失败: %v", err)
	} else {
		log.Printf("[INFO] [MySQL] 告警工单已生成: AlarmID=%s, Status=UNHANDLED", alarmID)
	}
}

// writeToRedis 写入 Redis 设备状态
func (s *TCPServer) writeToRedis(data *ParsedDeviceData) error {
	ctx := context.Background()
	key := fmt.Sprintf("device:status:%s", data.IMEI)

	status := "online"
	if data.SOSFlag {
		status = "alert"
	}

	fields := map[string]interface{}{
		"status":     status,
		"heart_rate": data.HeartRate,
		"battery":    data.Battery,
		"lat":        data.Latitude,
		"lon":        data.Longitude,
		"event_type": data.EventType,
		"updated_at": time.Now().Unix(),
	}

	if err := s.redisSvc.GetClient().HSet(ctx, key, fields).Err(); err != nil {
		return err
	}

	return s.redisSvc.GetClient().Expire(ctx, key, 10*time.Minute).Err()
}

// writeToInfluxDB 写入 InfluxDB 时序数据
func (s *TCPServer) writeToInfluxDB(data *ParsedDeviceData) error {
	if s.influxSvc == nil {
		return nil
	}

	return s.influxSvc.WriteTelemetry(&services.TelemetryData{
		IMEI:       data.IMEI,
		EventType:  data.EventType,
		Latitude:   data.Latitude,
		Longitude:  data.Longitude,
		HeartRate:  data.HeartRate,
		Battery:    data.Battery,
		SOSFlag:    data.SOSFlag,
		RawPayload: data.RawPayload,
	})
}

// writeToMySQL 写入 MySQL 事件记录
func (s *TCPServer) writeToMySQL(data *ParsedDeviceData) error {
	event := &models.DeviceEvent{
		IMEI:       data.IMEI,
		EventType:  data.EventType,
		Latitude:   data.Latitude,
		Longitude:  data.Longitude,
		HeartRate:  data.HeartRate,
		Battery:    data.Battery,
		RawPayload: data.RawPayload,
		EventTime:  data.Timestamp,
		CreatedAt:  time.Now(),
	}
	return s.mysqlSvc.SaveDeviceEvent(event)
}

// Stop 优雅停止 TCP 服务器
func (s *TCPServer) Stop() error {
	log.Printf("[INFO] 正在停止 TCP 服务器...")

	close(s.quit)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("[WARN] 关闭监听器失败: %v", err)
		}
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[INFO] TCP 服务器已停止，所有连接已关闭")
	case <-time.After(30 * time.Second):
		log.Printf("[WARN] TCP 服务器停止超时")
	}

	return nil
}
