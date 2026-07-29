# 长者安防手环 IoT 后端 API & TCP 协议文档

本文档详细说明了长者安防手环 IoT 后端服务提供的 **HTTP RESTful API** 以及 **原生 TCP 网关报文协议** 规范。

---

## 目录

1. [服务概览](#1-服务概览)
2. [HTTP RESTful API 规范](#2-http-restful-api-规范)
   - [2.1 系统与健康检查](#21-系统与健康检查)
   - [2.2 设备状态与告警](#22-设备状态与告警)
   - [2.3 硬件指令控制](#23-硬件指令控制)
   - [2.4 设备通讯录管理](#24-设备通讯录管理)
   - [2.5 上报周期设置](#25-上报周期设置)
   - [2.6 提醒事项管理](#26-提醒事项管理)
   - [2.7 EMQX Webhook 接入](#27-emqx-webhook-接入)
3. [原生 TCP 网关协议规范 (`:5007`)](#3-原生-tcp-网关协议规范-5007)
   - [3.1 上行报文规范](#31-上行报文规范)
   - [3.2 服务端响应与错误码](#32-服务端响应与错误码)
   - [3.3 下行控制报文](#33-下行控制报文)
   - [3.4 安全与防护规则](#34-安全与防护规则)
4. [curl / netcat 联调测试示例](#4-curl--netcat-联调测试示例)

---

## 1. 服务概览

- **HTTP REST API 地址**: `http://localhost:8080` (基准路径 `/api/v1`)
- **TCP 网关接入端口**: `localhost:5007` (分隔符 `#`)
- **数据响应通用格式**: JSON

---

## 2. HTTP RESTful API 规范

### 2.1 系统与健康检查

#### GET `/health`
- **说明**: 检查后端服务、数据库及中间件运行状态。
- **请求参数**: 无
- **响应示例**:
  ```json
  {
    "code": 200,
    "message": "服务正常运行",
    "time": "2026-07-27T10:30:00+08:00"
  }
  ```

---

### 2.2 设备状态与告警

#### GET `/api/v1/device/:imei/status`
- **说明**: 获取设备完整实时状态信息（位置、在线状态、电量、心率）。
- **路径参数**:
  - `imei` (string): 设备唯一标识号
- **响应示例**:
  ```json
  {
    "imei": "1234567890",
    "status": "online",
    "battery": 85,
    "last_heart_rate": 75,
    "last_latitude": 22.570733,
    "last_longitude": 113.862608,
    "address": "",
    "location_type": "GPS",
    "updated_at": 1785129000
  }
  ```

#### GET `/api/v1/device/:imei/state`
- **说明**: 从 Redis 快速查询设备最新在线/告警状态。
- **响应示例**:
  ```json
  {
    "code": 200,
    "data": {
      "imei": "1234567890",
      "status": "online",
      "heart_rate": 75,
      "battery": 85,
      "latitude": 22.570733,
      "longitude": 113.862608,
      "last_update": "2026-07-27T10:25:00Z",
      "msg_type": "V1",
      "is_online": true
    }
  }
  ```

#### GET `/api/v1/device/:imei/health`
- **说明**: 获取长者最新的健康体征指标数据（血压、血氧、体温、步数等）。
- **响应示例**:
  ```json
  {
    "blood_pressure": "120/75",
    "heart_rate": 77,
    "blood_oxygen": 99,
    "temperature": 36.7,
    "steps": 5415,
    "sleep": "7小时30分钟",
    "hrv": 24,
    "updated_at": "2026-07-27 10:30:00"
  }
  ```

#### GET `/api/v1/device/:imei/heart-rate/history?hours=24`
- **说明**: 从 InfluxDB 查询设备的历史心率时序点。
- **查询参数**: `hours` (int, 默认 24)
- **响应示例**:
  ```json
  {
    "imei": "1234567890",
    "count": 2,
    "points": [
      { "time": "2026-07-27T10:00:00Z", "heart_rate": 72 },
      { "time": "2026-07-27T10:15:00Z", "heart_rate": 76 }
    ]
  }
  ```

#### GET `/api/v1/device/:imei/alert`
- **说明**: 查询设备当前是否有未处理的 SOS / 跌倒告警。
- **响应示例**:
  ```json
  {
    "code": 200,
    "data": {
      "imei": "1234567890",
      "has_alert": true,
      "sos_alert": { "status": "sos_alert", "updated_at": 1785129000 },
      "fall_alert": null
    }
  }
  ```

---

### 2.3 硬件指令控制

#### POST `/api/v1/device/:imei/command`
- **说明**: 向保持 TCP 长连接的设备下发控制指令。
- **请求体**:
  ```json
  {
    "command": "FIND"
  }
  ```
- **支持指令**:
  - `FIND`: 寻找手环（触发手环响铃与震动）
  - `POWEROFF`: 关机指令
  - `RESET`: 重启指令
  - 自定义硬件报文（如 `*HQ,1234567890,FIND#`）
- **响应示例**:
  ```json
  {
    "code": 200,
    "message": "控制指令已成功下发至设备",
    "data": {
      "imei": "1234567890",
      "command": "FIND"
    }
  }
  ```

---

### 2.4 设备通讯录管理

#### GET `/api/v1/device/:imei/contacts`
- **说明**: 获取手环可拨打的亲情通讯录列表。

#### POST `/api/v1/device/:imei/contacts`
- **说明**: 添加联系人。
- **请求体**:
  ```json
  {
    "name": "女儿小美",
    "phone": "13800138000",
    "relation": "家属"
  }
  ```

#### DELETE `/api/v1/device/:imei/contacts/:id`
- **说明**: 删除指定的联系人。

---

### 2.5 上报周期设置

#### GET `/api/v1/device/:imei/settings`
- **说明**: 获取设备当前的 GPS 上报间隔设置（单位：秒）。

#### POST `/api/v1/device/:imei/settings`
- **说明**: 修改设备的 GPS 上报间隔。
- **请求体**:
  ```json
  {
    "interval": 300
  }
  ```

---

### 2.6 提醒事项管理

#### GET `/api/v1/device/:imei/reminders`
- **说明**: 获取手环闹钟/吃药提醒列表。

#### POST `/api/v1/device/:imei/reminders`
- **说明**: 新增提醒事项。
- **请求体**:
  ```json
  {
    "time": "08:00",
    "label": "早晨吃药"
  }
  ```

#### DELETE `/api/v1/device/:imei/reminders/:id`
- **说明**: 删除提醒事项。

---

### 2.7 EMQX Webhook 接入

#### POST `/api/v1/device/raw-tcp`
- **说明**: 接收 EMQX MQTT Broker 转发的设备报文 Webhook。
- **请求体**:
  ```json
  {
    "clientid": "1234567890",
    "topic": "device/raw",
    "payload": "*HQ,1234567890,V1,102030,A,2234.5678,N,11405.6789,E,000.0,000,75,85,220726,FFFFBBFF#"
  }
  ```

---

## 3. 原生 TCP 网关协议规范 (`:5007`)

### 3.1 上行报文规范

设备通过 TCP Socket 建立连接，以 `#` 字符作为帧尾结束符。

#### ① 定位与健康体征上报 (V1)
- **报文格式**:
  `*HQ,IMEI,V1,HHMMSS,A,Latitude,N/S,Longitude,E/W,Speed,Direction,HeartRate,Battery,DDMMYY,Status#`
- **实例**:
  `*HQ,1234567890,V1,102030,A,2234.5678,N,11405.6789,E,000.0,000,75,85,220726,FFFFBBFF#`

#### ② SOS 紧急求救告警
- **报文格式**:
  `*HQ,IMEI,SOS#`

---

### 3.2 服务端响应与错误码

| 响应文本 | 说明 | 触发原因 |
| :--- | :--- | :--- |
| `OK\n` | 上报成功 | 报文解析成功，身份鉴权通过 |
| `ERROR:PARSE_FAILED\n` | 格式解析错误 | 报文缺少必填字段或不符合 `*HQ` 协议 |
| `ERROR:UNREGISTERED_DEVICE\n` | 未注册设备拒绝 | IMEI 未在 MySQL `devices` 表开户 |
| `ERROR:RATE_LIMITED\n` | 发送频率限制 | 单连接 1 秒内上报 > 5 条报文 |
| `ERROR:TOO_MANY_ERRORS\n` | 恶意熔断 | 单连接连续 3 次解析失败，服务器强制断开 |

---

### 3.3 下行控制报文

当在 HTTP 接口提交控制请求后，服务器向绑定的 TCP Socket 下发：

- **寻找手表**: `*HQ,1234567890,FIND#\n`
- **远程关机**: `*HQ,1234567890,POWEROFF#\n`

---

### 3.4 安全与防护规则

1. **IP 并发上限**: 单 IP 最多保持 **50** 个并发长连接。
2. **建连频率限速**: 单 IP 每秒新建连接不得超过 **10** 次。
3. **读超时机制**: 单连接 **5 分钟** 无数据上报自动断开句柄。

---

## 4. curl / netcat 联调测试示例

```bash
# 1. 使用 netcat 模拟 TCP 设备上发定位报文
echo "*HQ,1234567890,V1,102030,A,2234.5678,N,11405.6789,E,000.0,000,75,85,220726,FFFFBBFF#" | nc localhost 5007

# 2. 使用 netcat 模拟发送 SOS 告警
echo "*HQ,1234567890,SOS#" | nc localhost 5007

# 3. HTTP 查询设备最新状态
curl http://localhost:8080/api/v1/device/1234567890/status

# 4. HTTP 向在线设备下发寻找手环指令
curl -X POST http://localhost:8080/api/v1/device/1234567890/command \
  -H "Content-Type: application/json" \
  -d '{"command":"FIND"}'

# 5. 健康检查
curl http://localhost:8080/health
```
