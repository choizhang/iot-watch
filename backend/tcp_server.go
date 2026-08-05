package main

import (
	"bufio"
	"context"
	"elder-guard-iot/config"
	"elder-guard-iot/handlers"
	"elder-guard-iot/models"
	"elder-guard-iot/services"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
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
	stopOnce  sync.Once        // 防止重复关闭 quit channel
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

		// 开启 TCP KeepAlive：OS 内核每 30s 发一次探针，连续 3 次无响应（≈90s）后
		// 自动关闭连接，彻底解决手表侧静默断开导致的"僵尸连接"假在线问题
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
			log.Printf("[INFO] [安全增强] 已为客户端 %s 启用 TCP KeepAlive (探测周期: 30s)", clientIP)
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

// splitByDelim 自定义分隔符，支持 '#', ']', '\n', '\r' 等多协议通用帧尾拆包
func (s *TCPServer) splitByDelim(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '#' || data[i] == ']' || data[i] == '\n' {
			return i + 1, data[:i+1], nil
		}
	}

	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// processPayload 处理接收到的报文，返回是否成功解析格式
func (s *TCPServer) processPayload(conn net.Conn, clientAddr string, payload string, boundIMEI *string) bool {
	log.Printf("[RAW TCP RECEIVED] [%s] Hex=%x | String=%s", clientAddr, []byte(payload), payload)

	parsedData, err := parseTCPPayload(payload)
	if err != nil {
		log.Printf("[WARN] [%s] 报文无缝容错解析中: %v", clientAddr, err)
		return true // 保持连接继续读取后续报文，不盲目熔断
	}

	log.Printf("[TCP PARSED] IMEI=%s, Type=%s, Lat=%.6f, Lon=%.6f, HR=%d, Batt=%d%%",
		parsedData.IMEI, parsedData.EventType,
		parsedData.Latitude, parsedData.Longitude,
		parsedData.HeartRate, parsedData.Battery)

	// 绑定 TCP 会话
	if boundIMEI != nil {
		if parsedData.IMEI == "" && *boundIMEI != "" {
			parsedData.IMEI = *boundIMEI
		} else if *boundIMEI == "" && parsedData.IMEI != "" {
			*boundIMEI = parsedData.IMEI
			s.sessions.Store(parsedData.IMEI, conn)
			log.Printf("[INFO] [%s] 成功绑定设备 TCP 会话句柄: IMEI=%s", clientAddr, parsedData.IMEI)
		} else if parsedData.IMEI != "" && *boundIMEI != parsedData.IMEI {
			// 同连接变更 IMEI
			s.sessions.Delete(*boundIMEI)
			*boundIMEI = parsedData.IMEI
			s.sessions.Store(parsedData.IMEI, conn)
		}
	}

	if parsedData.IMEI == "" {
		return true
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

	// 写入 InfluxDB 时序数据库
	if err := s.writeToInfluxDB(parsedData); err != nil {
		log.Printf("[ERROR] [%s] InfluxDB 写入失败: %v", clientAddr, err)
	}

	// 写入 MySQL 事件表
	if err := s.writeToMySQL(parsedData); err != nil {
		log.Printf("[ERROR] [%s] MySQL 事件写入失败: %v", clientAddr, err)
	}

	// 更新 MySQL 中的最后心跳时间与状态
	s.mysqlSvc.UpdateDeviceStatus(parsedData.IMEI, "online", time.Now(),
		parsedData.Latitude, parsedData.Longitude, parsedData.HeartRate, parsedData.Battery,
		parsedData.BloodPressure, parsedData.SpO2, parsedData.HRV, parsedData.Steps)

	if parsedData.AckResponse != "" {
		conn.Write([]byte(parsedData.AckResponse))
		log.Printf("[TCP ACK SENT] [%s] 发送响应应答包: %s", clientAddr, parsedData.AckResponse)

		// 若手环未锁定 GPS，下发 GPS 唤醒与定位频度设置指令
		if parsedData.Latitude == 0 && (parsedData.EventType == "VER" || parsedData.EventType == "LK" || parsedData.EventType == "WEATHER" || parsedData.EventType == "UD") {
			cmdGpsOn := fmt.Sprintf("[CS*%s*0006*GPSON,1]", parsedData.IMEI)
			cmdUpload := fmt.Sprintf("[CS*%s*000A*UPLOAD,30]", parsedData.IMEI)
			conn.Write([]byte(cmdGpsOn))
			conn.Write([]byte(cmdUpload))
			log.Printf("[TCP CMD DOWN] [%s] 主动下发 GPSON 唤醒指令: %s, %s", clientAddr, cmdGpsOn, cmdUpload)
		}
	} else {
		conn.Write([]byte("OK\n"))
	}
	return true
}

// isDeviceRegistered 校验设备 IMEI 是否已注册 (未注册时自动在线建档注册)
func (s *TCPServer) isDeviceRegistered(imei string) bool {
	if imei == "" {
		return false
	}

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

	suffix := imei
	if len(imei) >= 4 {
		suffix = imei[len(imei)-4:]
	}

	// 自动为新连入的真实硬件设备建档注册
	newDevice := models.Device{
		IMEI:        imei,
		DeviceName:  "真实智能手环 (" + suffix + ")",
		DeviceModel: "UWS6121E",
		OwnerName:   "长者用户",
		OwnerPhone:  "13800138000",
		Status:      "online",
		Battery:     50,
	}
	if err := s.mysqlSvc.GetDB().Create(&newDevice).Error; err == nil {
		log.Printf("[INFO] [硬件自适应建档] 成功自动为真实手环建立数据库档册: IMEI=%s", imei)
		s.redisSvc.GetClient().Set(ctx, cacheKey, "1", 10*time.Minute)
		return true
	}

	s.redisSvc.GetClient().Set(ctx, cacheKey, "0", 1*time.Minute)
	return false
}

// ParsedDeviceData 解析后的设备数据
type ParsedDeviceData struct {
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
	Timestamp     time.Time
	AckResponse   string
}

// IsAlert 判断是否为告警类型消息
func (p *ParsedDeviceData) IsAlert() bool {
	return p.SOSFlag || p.EventType == models.MsgTypeSOS || p.EventType == models.MsgTypeFall || p.EventType == "AL"
}

var (
	macRegex    = regexp.MustCompile(`([0-9a-fA-F]{2}[:\-][0-9a-fA-F]{2}[:\-][0-9a-fA-F]{2}[:\-][0-9a-fA-F]{2}[:\-][0-9a-fA-F]{2}[:\-][0-9a-fA-F]{2})`)
	cellRegex   = regexp.MustCompile(`@?(\d{3})!(\d+)!(\d+)!(\d+)`)
	wifiCacheMu sync.Mutex
	wifiCache   = make(map[string]*deviceWiFiCache)
)

type deviceWiFiCache struct {
	Cell      *handlers.CellTowerInfo
	BSSIDs    map[string]bool
	UpdatedAt time.Time
}

func tryResolveWiFiAndLBS(payload string, result *ParsedDeviceData) {
	if result == nil || (result.Latitude != 0 && result.Longitude != 0) {
		return
	}

	imei := result.IMEI
	wifiCacheMu.Lock()
	defer wifiCacheMu.Unlock()

	if imei == "" && len(wifiCache) > 0 {
		var recentKey string
		var latest time.Time
		for k, cache := range wifiCache {
			if cache.UpdatedAt.After(latest) && time.Since(cache.UpdatedAt) < 5*time.Second {
				latest = cache.UpdatedAt
				recentKey = k
			}
		}
		imei = recentKey
	}

	if imei == "" {
		return
	}

	cache, exists := wifiCache[imei]
	if !exists || time.Since(cache.UpdatedAt) > 10*time.Second {
		cache = &deviceWiFiCache{
			BSSIDs: make(map[string]bool),
		}
		wifiCache[imei] = cache
	}
	cache.UpdatedAt = time.Now()

	// 提取基站信息
	cellMatches := cellRegex.FindStringSubmatch(payload)
	if len(cellMatches) >= 5 {
		mcc, _ := strconv.Atoi(cellMatches[1])
		mnc, _ := strconv.Atoi(cellMatches[2])
		lac, _ := strconv.Atoi(cellMatches[3])
		cid, _ := strconv.Atoi(cellMatches[4])
		if cid > 0 {
			cache.Cell = &handlers.CellTowerInfo{
				MCC:    mcc,
				MNC:    mnc,
				LAC:    lac,
				CellID: cid,
			}
		}
	}

	// 提取全部 MAC 地址并做多热点聚合
	macMatches := macRegex.FindAllStringSubmatch(payload, -1)
	for _, m := range macMatches {
		if len(m) >= 2 {
			cache.BSSIDs[m[1]] = true
		}
	}

	var bssids []string
	for b := range cache.BSSIDs {
		bssids = append(bssids, b)
	}

	if len(bssids) > 0 || cache.Cell != nil {
		if lat, lng, _, locType, err := handlers.ResolveLocationFromWiFiAndLBS(bssids, cache.Cell); err == nil && lat != 0 && lng != 0 {
			result.Latitude = lat
			result.Longitude = lng
			result.EventType = "WIFI"
			if locType != "" {
				result.EventType = locType
			}
			result.IMEI = imei
		}
	}
}

// parseTCPPayload 解析 TCP 报文 (全面兼容 [CS*IMEI*LEN*CMD,...] 协议与 *HQ,IMEI,...# 协议)
func parseTCPPayload(payload string) (*ParsedDeviceData, error) {
	res, err := parseTCPPayloadInternal(payload)
	if err == nil && res != nil {
		tryResolveWiFiAndLBS(payload, res)
	}
	return res, err
}

func parseTCPPayloadInternal(payload string) (*ParsedDeviceData, error) {
	payload = strings.TrimSpace(payload)

	result := &ParsedDeviceData{
		RawPayload: payload,
		Timestamp:  time.Now(),
	}

	// 1. 处理 [厂商*IMEI*LEN*CMD,内容] 格式协议 (如 [CS*351086239665254*0014*LK,1785492499,0,5,43])
	if strings.HasPrefix(payload, "[") && strings.HasSuffix(payload, "]") {
		content := strings.TrimPrefix(payload, "[")
		content = strings.TrimSuffix(content, "]")

		sections := strings.SplitN(content, "*", 4)
		if len(sections) < 4 {
			return nil, fmt.Errorf("[方括号协议] 分隔节不足: %s", payload)
		}

		vendor := sections[0]
		imei := sections[1]
		cmdAndData := sections[3]

		if len(imei) < 10 || len(imei) > 20 {
			return nil, fmt.Errorf("[方括号协议] 无效 IMEI: %s", imei)
		}

		result.IMEI = imei
		cmdParts := strings.Split(cmdAndData, ",")
		cmd := strings.ToUpper(cmdParts[0])
		result.EventType = cmd

		// 心跳包 [CS*IMEI*LEN*LK,时间戳,步数,状态,电量]
		if cmd == "LK" {
			if len(cmdParts) >= 3 {
				if st, err := parseInt(cmdParts[2]); err == nil && st > 0 {
					result.Steps = st
				}
			}
			if len(cmdParts) >= 5 {
				if batt, err := parseInt(cmdParts[4]); err == nil {
					result.Battery = batt
				}
			} else if len(cmdParts) >= 4 {
				if batt, err := parseInt(cmdParts[3]); err == nil {
					result.Battery = batt
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*LK]", vendor, imei)
			return result, nil
		}

		// 版本包 [CS*IMEI*LEN*VER,...]
		if cmd == "VER" {
			result.AckResponse = fmt.Sprintf("[%s*%s*0003*VER]", vendor, imei)
			return result, nil
		}

		// 混合定位包 WEATHER [CS*IMEI*LEN*WEATHER,3,1785727050,209,1N0.0000E0.0000@460!...#]
		if cmd == "WEATHER" {
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*WEATHER]", vendor, imei)
			if len(cmdParts) >= 5 {
				mixStr := cmdParts[4]
				gpsPart := mixStr
				if idx := strings.Index(mixStr, "@"); idx != -1 {
					gpsPart = mixStr[:idx]
				}
				gpsPart = strings.TrimSuffix(gpsPart, "#")

				gpsRegex := regexp.MustCompile(`^(\d+)N([0-9.]+)E([0-9.]+)`)
				matches := gpsRegex.FindStringSubmatch(gpsPart)
				if len(matches) >= 4 {
					satellites, _ := strconv.Atoi(matches[1])
					latVal, _ := strconv.ParseFloat(matches[2], 64)
					lonVal, _ := strconv.ParseFloat(matches[3], 64)

					if satellites >= 3 && latVal > 0 && lonVal > 0 {
						if latVal > 90 {
							if parsedLat, err := parseCoordinate(matches[2], "N"); err == nil {
								latVal = parsedLat
							}
							if parsedLon, err := parseCoordinate(matches[3], "E"); err == nil {
								lonVal = parsedLon
							}
						}
						result.Latitude = latVal
						result.Longitude = lonVal
						result.EventType = "GPS"
					}
				}
			}
			if len(cmdParts) >= 4 {
				if batt, err := parseInt(cmdParts[3]); err == nil && batt <= 100 && batt > 0 {
					result.Battery = batt
				}
			}
			return result, nil
		}

		// 位置包与报警包 UD / UD2 / AL
		if cmd == "UD" || cmd == "UD2" || cmd == "AL" {
			if cmd == "AL" {
				result.SOSFlag = true
				result.EventType = models.MsgTypeSOS
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*%s]", vendor, imei, cmd)

			// 解析 GPS 基础位置 (数据从 cmdParts[1] 开始: 日期, 时间, 定位A/V, 纬度, N/S, 经度, E/W...)
			if len(cmdParts) >= 9 && strings.ToUpper(cmdParts[3]) == "A" {
				if lat, err := parseCoordinate(cmdParts[4], cmdParts[5]); err == nil {
					result.Latitude = lat
				}
				if lng, err := parseCoordinate(cmdParts[6], cmdParts[7]); err == nil {
					result.Longitude = lng
				}
			}
			// 解析电量与心率
			if len(cmdParts) >= 15 {
				if hr, err := parseInt(cmdParts[13]); err == nil && hr > 0 {
					result.HeartRate = hr
				}
				if batt, err := parseInt(cmdParts[14]); err == nil {
					result.Battery = batt
				}
			}
			return result, nil
		}

		// 血压与心率数据包 [CS*IMEI*LEN*BPUP,时间戳,舒张压(低压),收缩压(高压),心率]
		if cmd == "BPUP" {
			if len(cmdParts) >= 4 {
				dia, _ := parseInt(cmdParts[2])
				sys, _ := parseInt(cmdParts[3])
				if sys > 0 && dia > 0 {
					result.BloodPressure = fmt.Sprintf("%d/%d", sys, dia)
				}
			}
			if len(cmdParts) >= 5 {
				if hr, err := parseInt(cmdParts[4]); err == nil {
					result.HeartRate = hr
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*BPUP]", vendor, imei)
			return result, nil
		}

		// 血氧数据包 [CS*IMEI*LEN*SPO2,时间戳,血氧值] 或 [CS*IMEI*LEN*SPO2,血氧值]
		if cmd == "SPO2" || cmd == "SPO" || cmd == "BLOOD_OXYGEN" || cmd == "BO" {
			if len(cmdParts) >= 3 {
				if val, err := parseInt(cmdParts[2]); err == nil && val > 0 {
					result.SpO2 = val
				}
			}
			if result.SpO2 == 0 && len(cmdParts) >= 2 {
				if val, err := parseInt(cmdParts[1]); err == nil && val > 0 {
					result.SpO2 = val
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*%s]", vendor, imei, cmd)
			return result, nil
		}

		// 体温/HRV 数据包 [CS*IMEI*LEN*BT/HRV/TEMP,时间戳,数值]
		if cmd == "BT" || cmd == "TEMP" || cmd == "HRV" {
			if len(cmdParts) >= 3 {
				if val, err := parseInt(cmdParts[2]); err == nil && val > 0 {
					result.HRV = val
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*%s]", vendor, imei, cmd)
			return result, nil
		}

		// 心率数据包 [CS*IMEI*LEN*HEART,时间戳,心率] 或 [CS*IMEI*LEN*HT,时间戳,心率]
		if cmd == "HEART" || cmd == "HT" || cmd == "HR" || cmd == "HEARTRATE" {
			if len(cmdParts) >= 3 {
				if hr, err := parseInt(cmdParts[2]); err == nil && hr > 0 {
					result.HeartRate = hr
				}
			} else if len(cmdParts) >= 2 {
				if hr, err := parseInt(cmdParts[1]); err == nil && hr > 0 {
					result.HeartRate = hr
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0002*%s]", vendor, imei, cmd)
			return result, nil
		}

		// 心率/血压单发包 (hrtstart / bphrt)
		if cmd == "HRTSTART" || cmd == "BPHRT" {
			if len(cmdParts) >= 4 {
				if hr, err := parseInt(cmdParts[3]); err == nil {
					result.HeartRate = hr
				}
			}
			result.AckResponse = fmt.Sprintf("[%s*%s*0008*%s]", vendor, imei, cmd)
			return result, nil
		}

		// 默认通用 ACK 应答
		result.AckResponse = fmt.Sprintf("[%s*%s*0002*%s]", vendor, imei, cmd)
		return result, nil
	}

	// 2. 处理 *HQ,IMEI,TYPE,...# 格式协议
	if strings.HasPrefix(payload, "*HQ,") {
		cleanPayload := strings.TrimPrefix(payload, "*HQ,")
		cleanPayload = strings.TrimSuffix(cleanPayload, "#")

		parts := strings.Split(cleanPayload, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("报文字段不足")
		}

		result.IMEI = parts[0]
		if len(result.IMEI) < 10 || len(result.IMEI) > 20 {
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

	// 3. 处理 Wi-Fi 多热点基站续传行 (如 wifi1!60:3a:7c...#-42#)
	if strings.HasPrefix(payload, "wifi") {
		result.EventType = "WIFI"
		return result, nil
	}

	return nil, fmt.Errorf("未知的报文协议头: %s", payload)
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
			IMEI:          data.IMEI,
			EventType:     alertType,
			Latitude:      data.Latitude,
			Longitude:     data.Longitude,
			HeartRate:     data.HeartRate,
			Battery:       data.Battery,
			BloodPressure: data.BloodPressure,
			SpO2:          data.SpO2,
			HRV:           data.HRV,
			Steps:         data.Steps,
			SOSFlag:       true,
			RawPayload:    data.RawPayload,
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

	nowTs := time.Now().Unix()
	fields := map[string]interface{}{
		"status":     status,
		"battery":    data.Battery,
		"updated_at": nowTs,
	}
	if data.Latitude != 0 && data.Longitude != 0 {
		fields["lat"] = data.Latitude
		fields["lon"] = data.Longitude
	}
	if data.EventType != "" && data.EventType != "LK" && data.EventType != "VER" && data.EventType != "HB" && data.EventType != "WIFI_DATA" {
		fields["event_type"] = data.EventType
	}

	if data.HeartRate > 0 {
		fields["heart_rate"] = data.HeartRate
		fields["hr_updated_at"] = nowTs
	}
	if data.BloodPressure != "" {
		fields["bp"] = data.BloodPressure
		fields["bp_updated_at"] = nowTs
	}
	if data.SpO2 > 0 {
		fields["spo2"] = data.SpO2
		fields["spo2_updated_at"] = nowTs
	}
	if data.HRV > 0 {
		fields["hrv"] = data.HRV
		fields["hrv_updated_at"] = nowTs
	}
	if data.Steps > 0 {
		fields["steps"] = data.Steps
		fields["steps_updated_at"] = nowTs
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
		IMEI:          data.IMEI,
		EventType:     data.EventType,
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		HeartRate:     data.HeartRate,
		Battery:       data.Battery,
		BloodPressure: data.BloodPressure,
		SpO2:          data.SpO2,
		HRV:           data.HRV,
		Steps:         data.Steps,
		SOSFlag:       data.SOSFlag,
		RawPayload:    data.RawPayload,
	})
}

// writeToMySQL 写入 MySQL 事件记录（保存包含位置信息与 SOS/告警的事件记录，用于历史轨迹追溯与告警工单统计）
func (s *TCPServer) writeToMySQL(data *ParsedDeviceData) error {
	// 若非告警事件且无有效坐标位置，则不写 MySQL 仅存入 InfluxDB 时序库
	if !data.IsAlert() && data.Latitude == 0 && data.Longitude == 0 {
		return nil
	}

	locType := "GPS"
	if strings.Contains(strings.ToUpper(data.RawPayload), "WIFI") || data.EventType == "WIFI" {
		locType = "WIFI"
	} else if strings.Contains(strings.ToUpper(data.RawPayload), "LBS") || data.EventType == "LBS" {
		locType = "LBS"
	}

	event := &models.DeviceEvent{
		IMEI:          data.IMEI,
		EventType:     data.EventType,
		LocationType:  locType,
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		HeartRate:     data.HeartRate,
		Battery:       data.Battery,
		BloodPressure: data.BloodPressure,
		SpO2:          data.SpO2,
		HRV:           data.HRV,
		Steps:         data.Steps,
		RawPayload:    data.RawPayload,
		EventTime:     data.Timestamp,
		CreatedAt:     time.Now(),
	}
	return s.mysqlSvc.SaveDeviceEvent(event)
}

// Stop 优雅停止 TCP 服务器
func (s *TCPServer) Stop() error {
	log.Printf("[INFO] 正在停止 TCP 服务器...")

	s.stopOnce.Do(func() {
		close(s.quit)
	})

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("[WARN] 关闭监听器失败: %v", err)
		}
	}

	// 强制关闭所有活动 TCP 会话以立即解绑协程
	s.sessions.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		return true
	})

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
