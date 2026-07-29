package handlers

import (
	"elder-guard-iot/models"
	"elder-guard-iot/services"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CommandSender 下行指令发送接口
type CommandSender interface {
	SendDownstreamCommand(imei string, command string) error
}

// DeviceHandler 设备相关 HTTP 处理
type DeviceHandler struct {
	redisSvc  *services.RedisService
	mysqlSvc  *services.MySQLService
	alertSvc  *services.AlertService
	cmdSender CommandSender
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(redisSvc *services.RedisService, mysqlSvc *services.MySQLService, alertSvc *services.AlertService, cmdSender CommandSender) *DeviceHandler {
	return &DeviceHandler{
		redisSvc:  redisSvc,
		mysqlSvc:  mysqlSvc,
		alertSvc:  alertSvc,
		cmdSender: cmdSender,
	}
}

// HandleRawTCP 处理 EMQX Webhook 原始 TCP 报文
// 接口地址：POST /api/v1/device/raw-tcp
func (h *DeviceHandler) HandleRawTCP(c *gin.Context) {
	var webhookPayload models.EMQXWebhookPayload

	// 解析 JSON 请求体
	if err := c.ShouldBindJSON(&webhookPayload); err != nil {
		log.Printf("[ERROR] 解析 Webhook 请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求格式",
			"error":   err.Error(),
		})
		return
	}

	// 记录接收到的报文
	log.Printf("[INFO] 收到 EMQX Webhook: ClientID=%s, Topic=%s, Payload=%s",
		webhookPayload.ClientID, webhookPayload.Topic, webhookPayload.Payload)

	// 解析设备报文
	payload, err := services.ParseDevicePayload(webhookPayload.Payload)
	if err != nil {
		log.Printf("[ERROR] 解析设备报文失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "报文解析失败",
			"error":   err.Error(),
		})
		return
	}

	// 如果 Webhook 的 clientid 与解析出的 IMEI 不一致，以 clientid 为准
	if webhookPayload.ClientID != "" && webhookPayload.ClientID != payload.IMEI {
		log.Printf("[WARN] ClientID 不一致，Webhook: %s, Payload: %s",
			webhookPayload.ClientID, payload.IMEI)
		payload.IMEI = webhookPayload.ClientID
	}

	log.Printf("[INFO] 解析成功: %s", payload.String())

	// 处理报文（业务逻辑）
	if err := h.alertSvc.ProcessPayload(payload); err != nil {
		log.Printf("[ERROR] 处理报文失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "处理失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "处理成功",
		"data": gin.H{
			"imei":     payload.IMEI,
			"msg_type": payload.MsgType,
			"ts":       payload.Timestamp.UnixMilli(),
		},
	})
}

// HandleGetDeviceState 查询设备状态
// 接口地址：GET /api/v1/device/:imei/state
func (h *DeviceHandler) HandleGetDeviceState(c *gin.Context) {
	imei := c.Param("imei")
	if imei == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少设备 IMEI",
		})
		return
	}

	// 从 Redis 获取状态
	state, err := h.redisSvc.GetDeviceState(imei)
	if err != nil {
		log.Printf("[ERROR] 获取设备状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败",
		})
		return
	}

	if state == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "设备不在线或不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"imei":        state.IMEI,
			"status":      state.Status,
			"heart_rate":  state.HeartRate,
			"battery":     state.Battery,
			"latitude":    state.Latitude,
			"longitude":   state.Longitude,
			"last_update": state.LastUpdate.Format(time.RFC3339),
			"msg_type":    state.MsgType,
			"is_online":   state.IsOnline(),
		},
	})
}

// HandleGetDeviceAlert 查询设备告警
// 接口地址：GET /api/v1/device/:imei/alert
func (h *DeviceHandler) HandleGetDeviceAlert(c *gin.Context) {
	imei := c.Param("imei")
	if imei == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少设备 IMEI",
		})
		return
	}

	sosAlert, fallAlert, err := h.redisSvc.GetAlert(imei)
	if err != nil {
		log.Printf("[ERROR] 获取设备告警失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败",
		})
		return
	}

	hasAlert := (sosAlert != nil) || (fallAlert != nil)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"imei":      imei,
			"has_alert": hasAlert,
			"sos_alert": sosAlert,
			"fall_alert": fallAlert,
		},
	})
}

// HandleGetHealthData 查询设备健康指标
// 接口地址：GET /api/v1/device/:imei/health
func (h *DeviceHandler) HandleGetHealthData(c *gin.Context) {
	// imei := c.Param("imei")
	
	// 实际场景应从 Redis 或 InfluxDB 获取
	// 这里返回模拟数据以配合前端展示
	c.JSON(http.StatusOK, gin.H{
		"blood_pressure": "120/75",
		"heart_rate":     77,
		"blood_oxygen":   99,
		"temperature":    36.7,
		"steps":          5415,
		"sleep":          "--",
		"hrv":            24,
		"updated_at":     time.Now().Format("2006-01-02 15:04:05"),
	})
}

// HandleGetDeviceStatus 获取设备完整状态
// 接口地址：GET /api/v1/device/:imei/status
func (h *DeviceHandler) HandleGetDeviceStatus(c *gin.Context) {
	imei := c.Param("imei")
	
	state, _ := h.redisSvc.GetDeviceState(imei)
	
	status := "offline"
	var lat, lon float64
	var battery, hr int
	locationType := "LBS"
	
	if state != nil {
		status = state.Status
		lat = state.Latitude
		lon = state.Longitude
		battery = state.Battery
		hr = state.HeartRate
		locationType = "GPS"
	} else {
		dev, err := h.mysqlSvc.GetOrCreateDevice(imei)
		if err == nil && dev != nil && (dev.LastLatitude != 0 || dev.LastLongitude != 0) {
			status = dev.Status
			lat = dev.LastLatitude
			lon = dev.LastLongitude
			battery = dev.Battery
			hr = dev.LastHeartRate
			locationType = "GPS"
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"imei":            imei,
		"status":          status,
		"battery":         battery,
		"last_heart_rate": hr,
		"last_latitude":   lat,
		"last_longitude":  lon,
		"address":         "", // 地址由前端高德 Geocoder 实时解析
		"location_type":   locationType,
		"updated_at":      time.Now().Unix(),
	})
}

// HandleHealthCheck 健康检查
// 接口地址：GET /health
func (h *DeviceHandler) HandleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "服务正常运行",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// HandleGetHeartRateHistory 获取设备心率历史数据
// 接口地址：GET /api/v1/device/:imei/heart-rate/history
func (h *DeviceHandler) HandleGetHeartRateHistory(c *gin.Context) {
	imei := c.Param("imei")

	// 默认查询最近 24 小时
	hours := 24
	if h := c.Query("hours"); h != "" {
		fmt.Sscanf(h, "%d", &hours)
	}

	influxSvc := services.GetInfluxDBClient()
	if influxSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "InfluxDB 未连接"})
		return
	}

	points, err := influxSvc.QueryHeartRateHistory(imei, hours)
	if err != nil {
		log.Printf("[WARN] 心率历史查询失败: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"imei":   imei,
			"points": []interface{}{},
			"count":  0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"imei":   imei,
		"points": points,
		"count":  len(points),
	})
}

// HandleGetContacts 获取设备联系人
// 接口地址：GET /api/v1/device/:imei/contacts
func (h *DeviceHandler) HandleGetContacts(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var contacts []models.DeviceContact
	if err := db.Where("imei = ?", imei).Find(&contacts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取联系人失败"})
		return
	}

	// 如果没有联系人，初始化一些默认数据，方便展示
	if len(contacts) == 0 {
		defaultContacts := []models.DeviceContact{
			{IMEI: imei, Name: "女儿小美", Phone: "13800138000", Relation: "家属"},
			{IMEI: imei, Name: "社区医生", Phone: "0755-2345678", Relation: "医疗"},
		}
		for i := range defaultContacts {
			db.Create(&defaultContacts[i])
		}
		contacts = defaultContacts
	}

	c.JSON(http.StatusOK, contacts)
}

// HandleAddContact 新增设备联系人
// 接口地址：POST /api/v1/device/:imei/contacts
func (h *DeviceHandler) HandleAddContact(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var req struct {
		Name     string `json:"name" binding:"required"`
		Phone    string `json:"phone" binding:"required"`
		Relation string `json:"relation"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	contact := models.DeviceContact{
		IMEI:     imei,
		Name:     req.Name,
		Phone:    req.Phone,
		Relation: req.Relation,
	}

	if err := db.Create(&contact).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存联系人失败"})
		return
	}

	c.JSON(http.StatusOK, contact)
}

// HandleDeleteContact 删除设备联系人
// 接口地址: DELETE /api/v1/device/:imei/contacts/:id
func (h *DeviceHandler) HandleDeleteContact(c *gin.Context) {
	imei := c.Param("imei")
	id := c.Param("id")
	db := h.mysqlSvc.GetDB()

	if err := db.Where("imei = ? AND id = ?", imei, id).Delete(&models.DeviceContact{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除联系人失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// HandleGetSettings 获取设备配置
// 接口地址: GET /api/v1/device/:imei/settings
func (h *DeviceHandler) HandleGetSettings(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var setting models.DeviceSetting
	err := db.Where("imei = ?", imei).First(&setting).Error
	if err != nil {
		// 如果不存在，创建默认设置
		setting = models.DeviceSetting{
			IMEI:     imei,
			Interval: 300,
		}
		db.Create(&setting)
	}

	c.JSON(http.StatusOK, setting)
}

// HandleUpdateSettings 更新设备配置
// 接口地址: POST /api/v1/device/:imei/settings
func (h *DeviceHandler) HandleUpdateSettings(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var req struct {
		Interval int `json:"interval" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	var setting models.DeviceSetting
	err := db.Where("imei = ?", imei).First(&setting).Error
	if err != nil {
		setting = models.DeviceSetting{
			IMEI:     imei,
			Interval: req.Interval,
		}
		if err := db.Create(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存设置失败"})
			return
		}
	} else {
		setting.Interval = req.Interval
		if err := db.Save(&setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新设置失败"})
			return
		}
	}

	c.JSON(http.StatusOK, setting)
}

// HandleGetReminders 获取设备提醒设定列表
// 接口地址: GET /api/v1/device/:imei/reminders
func (h *DeviceHandler) HandleGetReminders(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var reminders []models.DeviceReminder
	if err := db.Where("imei = ?", imei).Find(&reminders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取提醒列表失败"})
		return
	}

	// 如果没有提醒，初始化默认提醒数据
	if len(reminders) == 0 {
		defaultReminders := []models.DeviceReminder{
			{IMEI: imei, Time: "08:00", Label: "早晨吃药", Enabled: true},
			{IMEI: imei, Time: "12:00", Label: "午餐时间", Enabled: true},
			{IMEI: imei, Time: "18:00", Label: "傍晚散步", Enabled: false},
		}
		for i := range defaultReminders {
			db.Create(&defaultReminders[i])
		}
		reminders = defaultReminders
	}

	c.JSON(http.StatusOK, reminders)
}

// HandleAddReminder 新增设备提醒设定
// 接口地址: POST /api/v1/device/:imei/reminders
func (h *DeviceHandler) HandleAddReminder(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var req struct {
		Time  string `json:"time" binding:"required"`
		Label string `json:"label" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	reminder := models.DeviceReminder{
		IMEI:    imei,
		Time:    req.Time,
		Label:   req.Label,
		Enabled: true,
	}

	if err := db.Create(&reminder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存提醒失败"})
		return
	}

	c.JSON(http.StatusOK, reminder)
}

// HandleDeleteReminder 删除设备提醒设定
// 接口地址: DELETE /api/v1/device/:imei/reminders/:id
func (h *DeviceHandler) HandleDeleteReminder(c *gin.Context) {
	imei := c.Param("imei")
	id := c.Param("id")
	db := h.mysqlSvc.GetDB()

	if err := db.Where("imei = ? AND id = ?", imei, id).Delete(&models.DeviceReminder{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除提醒失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// HandleSendCommand 向在线设备下发 TCP 控制指令
// 接口地址: POST /api/v1/device/:imei/command
func (h *DeviceHandler) HandleSendCommand(c *gin.Context) {
	imei := c.Param("imei")
	if imei == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少设备 IMEI",
		})
		return
	}

	var req struct {
		Command string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的命令参数",
			"error":   err.Error(),
		})
		return
	}

	if h.cmdSender == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "TCP 下发通道未链接",
		})
		return
	}

	if err := h.cmdSender.SendDownstreamCommand(imei, req.Command); err != nil {
		log.Printf("[WARN] 下发指令失败 IMEI=%s, Cmd=%s: %v", imei, req.Command, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "控制指令已成功下发至设备",
		"data": gin.H{
			"imei":    imei,
			"command": req.Command,
		},
	})
}

// HandleGetGeofences 获取设备电子围栏列表
// 接口地址: GET /api/v1/device/:imei/geofences
func (h *DeviceHandler) HandleGetGeofences(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var geofences []models.Geofence
	if err := db.Where("imei = ?", imei).Find(&geofences).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取电子围栏列表失败"})
		return
	}

	// 如果没有电子围栏，初始化默认的预警围栏
	if len(geofences) == 0 {
		defaultFence := models.Geofence{
			IMEI:      imei,
			Name:      "荃湾安老院安全围栏",
			Latitude:  22.371234,
			Longitude: 114.115678,
			Radius:    500,
			FenceType: "IN",
			Enabled:   true,
		}
		db.Create(&defaultFence)
		geofences = append(geofences, defaultFence)
	}

	c.JSON(http.StatusOK, geofences)
}

// HandleAddGeofence 创建电子围栏
// 接口地址: POST /api/v1/device/:imei/geofences
func (h *DeviceHandler) HandleAddGeofence(c *gin.Context) {
	imei := c.Param("imei")
	db := h.mysqlSvc.GetDB()

	var req struct {
		Name      string  `json:"name" binding:"required"`
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Radius    int     `json:"radius" binding:"required"`
		FenceType string  `json:"fence_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	fenceType := req.FenceType
	if fenceType == "" {
		fenceType = "IN"
	}

	fence := models.Geofence{
		IMEI:      imei,
		Name:      req.Name,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Radius:    req.Radius,
		FenceType: fenceType,
		Enabled:   true,
	}

	if err := db.Create(&fence).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建电子围栏失败"})
		return
	}

	c.JSON(http.StatusOK, fence)
}

// HandleDeleteGeofence 删除电子围栏
// 接口地址: DELETE /api/v1/device/:imei/geofences/:id
func (h *DeviceHandler) HandleDeleteGeofence(c *gin.Context) {
	imei := c.Param("imei")
	id := c.Param("id")
	db := h.mysqlSvc.GetDB()

	if err := db.Where("imei = ? AND id = ?", imei, id).Delete(&models.Geofence{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除电子围栏失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// HandleToggleGeofence 开关电子围栏使能状态
// 接口地址: PUT /api/v1/device/:imei/geofences/:id/toggle
func (h *DeviceHandler) HandleToggleGeofence(c *gin.Context) {
	imei := c.Param("imei")
	id := c.Param("id")
	db := h.mysqlSvc.GetDB()

	var fence models.Geofence
	if err := db.Where("imei = ? AND id = ?", imei, id).First(&fence).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "电子围栏不存在"})
		return
	}

	fence.Enabled = !fence.Enabled
	if err := db.Save(&fence).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新围栏状态失败"})
		return
	}

	c.JSON(http.StatusOK, fence)
}


