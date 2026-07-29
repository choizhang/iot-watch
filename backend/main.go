package main

import (
	"context"
	"elder-guard-iot/config"
	"elder-guard-iot/handlers"
	"elder-guard-iot/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[ELDER-GUARD] ")

	log.Println("========================================")
	log.Println("   长者安防手环 IoT 后端服务启动中...")
	log.Println("========================================")

	// 1. 初始化配置
	if err := config.InitConfig(); err != nil {
		log.Fatalf("[FATAL] 配置初始化失败: %v", err)
	}

	// 2. 初始化 MySQL
	if err := services.InitMySQL(config.GlobalConfig); err != nil {
		log.Fatalf("[FATAL] MySQL 初始化失败: %v", err)
	}
	mysqlSvc := services.GetMySQLClient()
	defer mysqlSvc.Close()

	// 3. 初始化 Redis
	if err := services.InitRedis(config.GlobalConfig); err != nil {
		log.Fatalf("[FATAL] Redis 初始化失败: %v", err)
	}
	redisSvc := services.GetRedisClient()
	defer redisSvc.Close()

	// 4. 初始化 InfluxDB（可选，失败不致命）
	var influxSvc *services.InfluxDBService
	if err := services.InitInfluxDB(config.GlobalConfig); err != nil {
		log.Printf("[WARN] InfluxDB 初始化失败，将不写入时序数据: %v", err)
	} else {
		influxSvc = services.GetInfluxDBClient()
		defer influxSvc.Close()
	}

	// 5. 初始化告警服务
	services.InitAlertService(redisSvc, mysqlSvc)
	alertSvc := services.GetAlertService()

	// 6. 启动 TCP Server（原生设备接入）
	tcpServer := NewTCPServer(redisSvc, mysqlSvc, influxSvc)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("[FATAL] TCP Server 启动失败: %v", err)
	}
	defer tcpServer.Stop()

	// 7. 启动离线设备检测服务 (每 1 分钟检查一次，超时 5 分钟标为离线)
	offlineDetector := services.NewOfflineDetector(mysqlSvc, redisSvc, 1*time.Minute, 5*time.Minute)
	offlineDetector.Start(context.Background())

	// 8. 创建 Gin 路由器
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 添加中间件
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLogger())

	// 9. 创建处理器 (注入 TCP 指令下发通道)
	deviceHandler := handlers.NewDeviceHandler(redisSvc, mysqlSvc, alertSvc, tcpServer)

	// 10. 注册路由
	registerRoutes(router, deviceHandler)

	// 11. 启动 HTTP 服务
	addr := config.GlobalConfig.GetServerAddr()
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 启动服务（异步）
	go func() {
		log.Printf("[INFO] HTTP 服务正在监听: http://%s", addr)
		log.Printf("[INFO] EMQX Webhook 接收地址: http://%s%s", addr, config.GlobalConfig.EMQX.WebhookPath)
		log.Printf("[INFO] 健康检查地址: http://%s/health", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP 服务启动失败: %v", err)
		}
	}()

	log.Println("========================================")
	log.Println("   服务已就绪，等待设备连接...")
	log.Println("========================================")

	// 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] 正在关闭服务（优雅退出）...")

	// 停止 TCP Server
	tcpServer.Stop()

	// 关闭 HTTP Server（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[WARN] HTTP Server 关闭异常: %v", err)
	}

	// 刷新 InfluxDB 缓冲区
	if influxSvc != nil {
		influxSvc.Flush()
	}

	log.Println("[INFO] 服务已关闭")
}

// registerRoutes 注册路由
func registerRoutes(router *gin.Engine, deviceHandler *handlers.DeviceHandler) {
	// 健康检查
	router.GET("/health", deviceHandler.HandleHealthCheck)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		device := v1.Group("/device")
		{
			// EMQX Webhook 接收端点
			device.POST("/raw-tcp", deviceHandler.HandleRawTCP)

			// 查询设备状态
			device.GET("/:imei/state", deviceHandler.HandleGetDeviceState)

			// 获取设备完整状态
			device.GET("/:imei/status", deviceHandler.HandleGetDeviceStatus)

			// 获取健康指标
			device.GET("/:imei/health", deviceHandler.HandleGetHealthData)

			// 查询设备心率历史
			device.GET("/:imei/heart-rate/history", deviceHandler.HandleGetHeartRateHistory)

			// 查询设备告警
			device.GET("/:imei/alert", deviceHandler.HandleGetDeviceAlert)

			// 下发 TCP 控制指令
			device.POST("/:imei/command", deviceHandler.HandleSendCommand)

			// 电话薄相关
			device.GET("/:imei/contacts", deviceHandler.HandleGetContacts)
			device.POST("/:imei/contacts", deviceHandler.HandleAddContact)
			device.DELETE("/:imei/contacts/:id", deviceHandler.HandleDeleteContact)

			// 定位间隔相关
			device.GET("/:imei/settings", deviceHandler.HandleGetSettings)
			device.POST("/:imei/settings", deviceHandler.HandleUpdateSettings)

			// 提醒设定相关
			device.GET("/:imei/reminders", deviceHandler.HandleGetReminders)
			device.POST("/:imei/reminders", deviceHandler.HandleAddReminder)
			device.DELETE("/:imei/reminders/:id", deviceHandler.HandleDeleteReminder)

			// 电子围栏相关
			device.GET("/:imei/geofences", deviceHandler.HandleGetGeofences)
			device.POST("/:imei/geofences", deviceHandler.HandleAddGeofence)
			device.DELETE("/:imei/geofences/:id", deviceHandler.HandleDeleteGeofence)
			device.PUT("/:imei/geofences/:id/toggle", deviceHandler.HandleToggleGeofence)
		}
	}
}

// corsMiddleware 跨域中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// requestLogger 请求日志中间件
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[HTTP] %s %s %d %v", method, path, status, latency)
	}
}
