# 🛡️ Elder-Guard IoT 社区养老智慧安防平台

> **Elder-Guard IoT Security Platform** 是一套全栈分布式长者守护与社区安防系统。系统包含硬件 TCP 字节流/EMQX Webhook 接入网关、Go 微服务后端、三库存储引擎（MySQL + Redis + InfluxDB）、C 端家属移动端 H5 界面以及 B 端社区安防指挥控制大屏。

---

## 📐 系统架构与技术选型

```
 ┌──────────────────────────────────────────────────────────┐
 │                  IoT 手环终端 (TCP / ASCII 协议)           │
 └────────────────────────────┬─────────────────────────────┘
                              │ TCP :5007 / EMQX Webhook
                              ▼
 ┌──────────────────────────────────────────────────────────┐
 │                   Go 后端微服务网关 (:8080)               │
 │  - TCP Socket Server        - EMQX Webhook Handler       │
 │  - HQ 协议解算/逆地理编码    - 围栏/告警触发引擎           │
 └───────┬────────────────────┬────────────────────┬────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ MySQL 8.0 持久化 │ │ Redis 高速缓存   │ │ InfluxDB 时序轨迹│
│ (设备/围栏/工单) │ │ (实时锁/去重/状态│ │ (心率/电量/轨迹点│
└──────────────────┘ └──────────────────┘ └──────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │ REST API / 轮询 / WebSocket
                              ▼
            ┌──────────────────┴──────────────────┐
            ▼                                     ▼
┌───────────────────────┐             ┌───────────────────────┐
│ C 端家属移动端 (5173)  │             │ B 端安防指挥大屏 (5174)│
│ Vue 3 + Pinia + 高德  │             │ Vue 3 + TailwindCSS  │
└───────────────────────┘             └───────────────────────┘
```

### 技术栈列表

- **后端 (Backend)**: Golang 1.20+ / Gin / GORM / Go-Redis / InfluxDB Client v2 / Native TCP Socket Server
- **C端移动端 (Frontend)**: Vue 3 / TypeScript / Pinia / Vite / TailwindCSS / 高德地图 API
- **B端指挥大屏 (Dashboard)**: Vue 3 / TypeScript / Pinia / Vite / Lucide Icons / 炫酷暗黑科技视觉系统
- **中间件 (Middleware)**: Docker Compose (MySQL 8.0, Redis 7.0, InfluxDB 2.7)
- **自动化测试套件 (Testing)**:
  - 后端多库断言引擎 (`backend/cmd/test_engine`)
  - Playwright E2E UI 闭环自动化测试 (`e2e/`)

---

## 🔥 核心业务功能

1. **🚨 SOS 紧急求救风暴与全端联动**：
   手环触发 SOS 按键时，后端 10ms 内写入高优先级 Redis 告警链并下发告警工单，B 端大屏即刻弹窗并响起声光警报，C 端卡片高亮呈送。
2. **🌐 GPS / WIFI / LBS 多模态定位自动解算**：
   根据手环室外 GPS 信号、室内 WIFI AP 扫描与基站 LBS 强度，动态计算最佳坐标并匹配高德地图逆地理编码。
3. **⭕ 电子围栏越界实时判定**：
   支持基于保护半径的电子围栏规则配置，当长者超出安全区域时自动触发 OUT 出界告警。
4. **💓 异常跌倒与高心率体征预警**：
   实时采集心率、血氧、体温与步数，当心率突破预警临界值（如 >120 bpm）或触发跌倒算法时，自动推送异常日志。

---

## 🚀 快速启动指南

### 1. 环境变量配置
复制环境变量模版文件：
```bash
cp .env.example .env
```
*(注：敏感配置、数据库密码及 Token 已做掩码化处理)*

### 2. 一键启动全部服务
脚本会自动一键启动 Docker 镜像、初始化数据库架构并拉起 Go 后端及前端服务：
```bash
./start.sh
```

### 3. 访问应用端点
- **📱 C端家属移动端**: `http://localhost:5173`
- **🖥️ B端社区安防大屏**: `http://localhost:5174`
- **🏥 后端健康检查地址**: `http://localhost:8080/health`

### 4. 一键停止与清理
一键优雅退出所有前端 Node 进程、Go 进程及清理 Docker 容器与日志：
```bash
./stop.sh
```

---

## 🧪 自动化测试体系

本项目具备 0 Token 消耗的后端极速断言引擎与全闭环 UI 端到端测试体系：

### 1. 运行后端三库一致性断言引擎 (0.3秒极速)
```bash
cd backend && go run ./cmd/test_engine
```
自动校验协议解析、吞吐延时与 MySQL / Redis / InfluxDB 100% 数据一致性，结果保存于 `test_report.md`。

### 2. 运行 Playwright E2E 端到端 UI 测试
```bash
cd e2e && npx playwright test
```
Playwright 自动模拟硬件发包并在 Chromium 浏览器中打开 CMS 安防大屏与 H5 移动端，执行零假阳性的硬核 UI 断言。

查看可视化 HTML 网页测试报告：
```bash
cd e2e && npx playwright show-report
```

---

## 📂 项目目录结构

```
iot/
├── backend/                  # Go 后端服务根目录
│   ├── cmd/test_engine/      # 后端三库一致性断言测试引擎
│   ├── config/               # 服务与中间件配置解析
│   ├── handlers/             # HTTP & Webhook REST 路由处理器
│   ├── models/               # MySQL & Redis 数据结构模型
│   ├── services/             # TCP 解算、告警逻辑与三库 Service 抽象
│   └── main.go               # 服务入口与路由注册
├── frontend/                 # C端家属移动端 (Vue 3 + Vite)
├── dashboard/                # B端社区安防大屏 (Vue 3 + Vite)
├── e2e/                      # Playwright E2E UI 全闭环测试套件
├── docker-compose.yml        # Docker 中间件一键编排配置
├── start.sh                  # 一键启动脚本
├── stop.sh                   # 一键停止与日志清理脚本
├── TESTING.md                # 全场景自动化测试指南
└── README.md                 # 项目主说明文档
```

---

## 🔐 隐私与安全说明

- 所有配置文件、敏感密钥与日志目录已加入 `.gitignore`。
- 代码库中不包含任何真实个人私密身份与敏感 Token。

---

## 📄 开源协议 (License)

本项目采用 [MIT License](LICENSE) 开源协议。欢迎学习交流、提交 PR 与二次开发。

