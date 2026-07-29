# 长者安防手环 IoT 后端服务

基于 Go + Gin 框架开发的长者智能手环/手表数据接收服务，支持原生 TCP 协议和 EMQX Webhook 两种接入方式。

## 技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 语言/框架 | Go 1.21 + Gin | 高并发，适合 IoT 场景 |
| TCP 协议 | 原生 `net.Listen` + `bufio.Scanner` | 避免依赖，精细控制粘包 |
| 消息队列 | EMQX 5.3 | MQTT Broker，支持 Webhook 转发 |
| 实时缓存 | Redis 7.0 (go-redis/v9) | 设备状态缓存，Hash 结构 |
| 时序数据库 | InfluxDB 2.7 (influxdb-client-go/v2) | GPS 轨迹、体征数据 |
| 关系数据库 | MySQL 8.0 (GORM) | 静态关系数据、告警工单 |
| 容器化 | Docker Compose | 本地开发环境 |

## 项目架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     长者智能手环/手表                             │
│                   (*HQ,IMEI,TYPE,...#)                          │
└─────────────────────┬───────────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌───────────────┐           ┌───────────────┐
│   EMQX 5.3    │           │  TCP :5007   │
│   (MQTT)      │           │  (原生 TCP)   │
└───────┬───────┘           └───────┬───────┘
        │                             │
        │ Webhook POST                │ 直接连接
        │ /api/v1/device/raw-tcp      │
        ▼                             ▼
┌───────────────────────────────────────────────────────────────┐
│                     Go IoT 后端服务                            │
│                      localhost:8080                            │
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐ │
│  │  Gin     │  │   TCP     │  │  Parser  │  │   Service    │ │
│  │ HTTP API │  │  Server   │  │  报文解析 │  │   业务逻辑    │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘ │
│       │              │              │                │          │
│       └──────────────┴──────────────┴────────────────┘          │
│                              │                                  │
│       ┌──────────────────────┼──────────────────────┐         │
│       ▼                      ▼                      ▼         │
│  ┌─────────┐         ┌───────────┐         ┌──────────┐      │
│  │  Redis  │         │ InfluxDB  │         │  MySQL   │      │
│  │(实时状态)│         │ (时序数据) │         │(关系数据) │      │
│  └─────────┘         └───────────┘         └──────────┘      │
└───────────────────────────────────────────────────────────────┘
```

## 目录结构

```
/iot
├── config/
│   └── config.go          # 配置加载（.env + 环境变量）
├── models/
│   ├── emqx.go            # EMQX Webhook 数据结构
│   ├── device.go          # MySQL 数据模型
│   └── redis.go           # Redis 状态模型
├── handlers/
│   └── device.go          # HTTP API 处理器
├── services/
│   ├── parser.go          # 手环报文解析
│   ├── redis.go           # Redis 操作
│   ├── mysql.go           # MySQL 操作
│   ├── influxdb.go        # InfluxDB 操作
│   └── alert.go           # 告警服务（HTTP API 触发）
├── main.go                # 入口：Gin HTTP 服务
├── tcp_server.go          # 原生 TCP Server（独立端口 5007）
├── docker-compose.yml      # 中间件容器编排
├── go.mod / go.sum        # Go 依赖
├── .env.example           # 环境变量示例
└── README.md
```

## 核心决策点

### 1. TCP Server vs EMQX MQTT 接入

**两种接入方式并存：**

| 方式 | 端口 | 适用场景 |
|------|------|----------|
| 原生 TCP Server | `:5007` | 支持私有多协议手环直连 |
| EMQX Webhook | `:8080` | 手环先连 EMQX(MQTT)，再转发 HTTP |

**决策原因**：
- 私有多协议手环可能不支持 MQTT，TCP 直连更通用
- EMQX 作为 gateway 可统一管理设备连接、认证
- 两种方式解析逻辑统一，复用 `parseTCPPayload`

### 2. 数据存储分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                        存储分层                               │
├──────────────┬────────────────┬───────────────────────────┤
│    Redis     │   InfluxDB     │         MySQL              │
│   实时状态    │    时序数据     │        关系数据            │
├──────────────┼────────────────┼───────────────────────────┤
│ 设备在线状态  │ GPS 轨迹点      │ 设备注册信息               │
│ 最新心率     │ 心率时序        │ 长者资料                   │
│ 最新电量     │ SOS/FALL 事件   │ 用户账号                   │
│ 最新位置     │                │ 告警工单                   │
├──────────────┼────────────────┼───────────────────────────┤
│  TTL 10分钟  │ 永久保留        │ 永久保留                   │
│  Hash 结构   │ Measurement    │ GORM 表结构                │
└──────────────┴────────────────┴───────────────────────────┘
```

**决策原因**：

| 存储 | 选型理由 |
|------|----------|
| Redis | 前端/App 查询设备状态需要毫秒级响应，Hash 结构方便局部更新 |
| InfluxDB | GPS 轨迹、体征折线图需要按时间范围查询，InfluxDB 比 MySQL 高效 10x+ |
| MySQL | 告警工单需要事务支持、关联查询、状态机流转 |

### 3. 报文解析设计

**支持两种报文格式：**

```
# 定位+心率报文（V1协议）
*HQ,1234567890,V1,102030,A,2234.5678,N,11405.6789,E,000.0,000,220726,FFFFBBFF#
  │        │  │    │  │   │        │ │        │ │      │ │     └─ 校验/电量
  │        │  │    │  │   │        │ │        │ │      └─ 心率
  │        │  │    │  │   │        │ │        │ └─ 速度
  │        │  │    │  │   │        │ │        └─ 方向
  │        │  │    │  │   │        │ └─ 经度
  │        │  │    │  │   │        └─ 纬度方向(N/S)
  │        │  │    │  │   └─ 纬度
  │        │  │    │  └─ 定位状态(A=有效/V=无效)
  │        │  │    └─ 时间 (HHMMSS)
  │        │  └─ 协议版本
  │        └─ IMEI (10-15位)

# SOS 紧急告警
*HQ,1234567890,SOS#
```

**关键设计**：
- 使用 `bufio.Scanner` 按 `#` 分隔符拆包，解决 TCP 粘包/断包
- 5 分钟读超时，防止僵尸连接
- 解析函数返回结构体，避免全局状态

### 4. SOS/FALL 告警三存储

```
收到 SOS/FALL
    │
    ├─► 1. Redis: 设备状态 → "sos_alert"
    │               alert:sos:{imei} = "1" (TTL 30分钟)
    │               [用于前端实时感知告警状态]
    │
    ├─► 2. InfluxDB: 写入 SOS 时序点
    │               Measurement: device_telemetry
    │               Tag: imei, event_type=SOS
    │               Field: lat, lon, heart_rate, sos_flag=true
    │               [用于告警回溯、轨迹复现]
    │
    └─► 3. MySQL: 生成告警工单
                    alarm_id = 'SOS-20260723-120500-7890'
                    status = 'UNHANDLED'
                    [用于人工处理流程、工单流转]
```

### 5. EMQX Webhook vs 直连 TCP

| 场景 | 推荐方式 | 理由 |
|------|----------|------|
| 大量设备（>1000） | EMQX MQTT | 连接管理、认证、QoS 更完善 |
| 少量设备、私有协议 | TCP 直连 | 减少中间环节，延迟更低 |
| 需要 Webhook 转发 | EMQX → HTTP | 兼容现有系统 |

**本项目两种方式都支持，可根据实际设备协议选择。**

### 6. TCP 安全防护与鉴权

原生 TCP 网关 (`:5007`) 内置了生产级防护机制：

| 防护项 | 机制说明 | 触发后果 |
|--------|----------|----------|
| **IP 并发与频控** | 单 IP 限额 50 个并发连接，新建连接限速 10 次/秒 | 直接拒绝 TCP 握手并关闭 `net.Conn` |
| **设备 IMEI 鉴权** | 优先查 Redis，未命中查 MySQL `devices` 开户表 | 未开户返回 `ERROR:UNREGISTERED_DEVICE` 且不入库 |
| **报文上发速率控制** | 单连接限制最大 5 条/秒 | 返回 `ERROR:RATE_LIMITED` 并进行 200ms 平滑退避 |
| **格式错误熔断惩罚** | 单连接连续 3 次报文解析失败 | 返回 `ERROR:TOO_MANY_ERRORS` 并断开 TCP 连接 |

## API 接口

> 📖 **完整 API 与 TCP 报文协议文档请参阅**：[API_DOCS.md](file:///Users/choizhang/Documents/iot/backend/API_DOCS.md)

### HTTP API (Gin)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/device/raw-tcp` | EMQX Webhook 接收 |
| GET | `/api/v1/device/:imei/state` | 查询设备状态（从 Redis） |
| GET | `/api/v1/device/:imei/alert` | 查询设备告警 |
| GET | `/health` | 健康检查 |

### TCP Server

| 端口 | 协议 | 说明 |
|------|------|------|
| `:5007` | TCP | 手环直连，ASCII 报文（内置 IP 限流/IMEI 鉴权/熔断） |

## 快速开始

### 1. 环境要求

- Go 1.21+
- Docker & Docker Compose
- Mac/Linux (本项目开发环境 macOS)

### 2. 启动中间件

```bash
# 启动所有容器
docker-compose up -d

# 验证服务状态
docker ps --format "table {{.Names}}\t{{.Status}}"
```

### 3. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 填入实际配置
```

### 4. 运行服务

```bash
# 下载依赖
go mod tidy

# 运行
go run .
```

### 5. 测试

```bash
# 模拟 TCP 设备发送定位数据
echo "*HQ,1234567890,V1,102030,A,2234.5678,N,11405.6789,E,72,85,220726,FFFFBBFF#" | nc localhost 5007

# 模拟 TCP 设备发送 SOS
echo "*HQ,1234567890,SOS#" | nc localhost 5007

# 查询设备状态
curl http://localhost:8080/api/v1/device/1234567890/state

# 健康检查
curl http://localhost:8080/health
```

## 报文协议说明

### 解析字段映射

| 报文位置 | 字段名 | 说明 |
|----------|--------|------|
| `parts[0]` | IMEI | 设备唯一标识 |
| `parts[1]` | MsgType | 消息类型：V1/SOS/FALL/HEARTBEAT |
| `parts[5]` | GPSStatus | A=有效定位，V=无效 |
| `parts[6,7]` | Latitude | 纬度（度分格式 + 方向） |
| `parts[8,9]` | Longitude | 经度（度分格式 + 方向） |
| `倒数第3个` | HeartRate | 心率值 |
| `倒数第1个` | Battery | 电量 |

### GPS 坐标转换

```
输入：2234.5678,N (度分格式 DDMM.MMMM)
处理：
  degrees = 22
  minutes = 34.5678
  decimal = 22 + 34.5678/60 = 22.57613°
输出：22.57613 (北纬为正，南纬为负)
```

## 配置参考

### docker-compose.yml 服务端口

| 服务 | 端口 | Web UI |
|------|------|--------|
| EMQX | 1883, 18083, 5006 | http://localhost:18083 |
| Redis | 6379 | - |
| InfluxDB | 8086 | http://localhost:8086 |
| MySQL | 3306 | - |

### .env 关键配置

```bash
# 服务端口
PORT=8080
HOST=0.0.0.0

# MySQL
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=YourPassword123
MYSQL_DATABASE=hkbn_iot

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# InfluxDB (首次启动会在容器内自动初始化)
INFLUXDB_URL=http://localhost:8086
INFLUXDB_ORG=hkbn
INFLUXDB_BUCKET=iot
```

## 数据库表结构

### MySQL 表

| 表名 | 说明 |
|------|------|
| `devices` | 设备注册表 |
| `alarm_orders` | 告警工单表 |

### InfluxDB Measurement

| Measurement | 说明 |
|-------------|------|
| `device_telemetry` | 设备遥测数据（GPS、心率、SOS事件） |
| `gps_data` | GPS 轨迹数据 |
| `health_data` | 健康体征数据 |

### Redis Key 规范

| Key 格式 | 类型 | TTL | 说明 |
|----------|------|-----|------|
| `device:status:{imei}` | Hash | 10min | 设备实时状态 |
| `alert:sos:{imei}` | String | 30min | SOS 告警标识 |
| `alert:fall:{imei}` | String | 30min | 跌倒告警标识 |

## 性能考量

| 优化点 | 实现方式 |
|--------|----------|
| TCP 粘包处理 | `bufio.Scanner` 按 `#` 分隔，避免手动 buffer 管理 |
| 连接超时 | 5 分钟读超时自动关闭，防止僵尸连接 |
| Redis 连接池 | go-redis/v9 默认复用连接 |
| InfluxDB 批量写入 | 100 条 batch，5s 刷新间隔 |
| MySQL 连接池 | MaxOpen=100, MaxIdle=10 |

## 未来扩展点

- [ ] APNs/FCM 推送集成
- [ ] 设备固件 OTA 升级
- [ ] 告警规则引擎（心率超阈值自动告警）
- [ ] 轨迹数据 Geohash 压缩
- [ ] 设备心跳超时离线检测

## License

MIT
