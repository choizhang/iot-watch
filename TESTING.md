# IoT 智慧养老系统全场景测试指南 (TESTING.md)

本指南规范并引导开发与测试人员对 **IoT 长者守护与社区安防系统** 进行后端协议/三库断言测试以及前端 E2E UI 闭环功能测试。

---

## ⚡ 核心测试能力双引擎概览

| 测试维度 | 技术方案 | 运行目录 | 核心作用与断言点 |
| :--- | :--- | :--- | :--- |
| **后端/数据库断言引擎** | Go 原生测试 (`test_engine`) | `backend/` | **0 Token 极速运行 (0.3s)**，断言协议解析、吞吐延迟与三库（MySQL / Redis / InfluxDB）一致性 |
| **E2E UI 闭环测试引擎** | Playwright + Chromium | `e2e/` | **全闭环 UI 断言**，模拟设备发包 ➔ 后端处理 ➔ 自动打开 CMS 大屏与 H5 移动端断言 DOM 响应 |

---

## 🚀 1. 快速运行自动化测试命令

### 方案 A：运行后端 API 与三库一致性测试 (0.3s 极速)
```bash
cd backend && go run ./cmd/test_engine
```
> 运行完成后自动更新根目录下的 [test_report.md](file:///Users/choizhang/Documents/iot/test_report.md) 文件。

### 方案 B：运行 E2E 硬件到 Web 多端闭环 UI 测试
```bash
cd e2e && npx playwright test
```
> Playwright 会自动模拟设备发包并在 Chromium 浏览器中打开 **CMS 安防大屏 (5174)** 与 **H5 移动端 (5173)** 进行断言，测试报告生成在 `e2e/playwright-report/index.html`。

#### 更多 E2E 实用指令：
- **有头模式运行（可实时看到浏览器自动操作）**：`cd e2e && npx playwright test --headed`
- **查看 HTML 可视化测试报告与运行截图**：`cd e2e && npx playwright show-report`

---

## 📋 2. E2E 全链路闭环测试用例矩阵 (7 大 Core Scenarios)

| 用例编号 | 场景名称 | 设备发包特征 | CMS 安防大屏 (5174) 响应 | H5 家属移动端 (5173) 响应 |
| :--- | :--- | :--- | :--- | :--- |
| **E2E-01** | **SOS 紧急求救联动** | `msgType="SOS"`, HR=125bpm | 红色 SOS 标签与高亮告警 | 长者卡片状态变为紧急求救 |
| **E2E-02** | **定位模式动态切换** | 连续发 3 包 (GPS ➔ WIFI ➔ LBS) | 定位模式文字平滑切换 | 坐标信息精准刷新 |
| **E2E-03** | **电子围栏越界告警** | 坐标偏离社区中心 1.2km | 地图标点位置更新 | 长者位置越界感知 |
| **E2E-04** | **跌倒检测告警** | `msgType="FALL"` | 告警风暴列表新增记录 | 页面呈现异常提醒 |
| **E2E-05** | **高心率体征预警** | HR=125 bpm | 心率指标高亮显示 | 首页心率数值刷出 |
| **E2E-06** | **低电量与离线预警** | `battery=8%` | 低电量数值与状态刷新 | 首页显示低电警示 |
| **E2E-07** | **正常漫游基线** | 正常 GPS 定位, HR=72, bat=85 | 管控设备总数与地图标点 | 正常上线定位展示 |

---

## 📁 3. 日志与辅助清理

为了保持根目录整洁，所有后台日志和 PID 文件均保存在 `.logs/` 中：
- 查看后端日志：`tail -f .logs/backend.log`
- 查看移动端日志：`tail -f .logs/frontend.log`
- 查看大屏日志：`tail -f .logs/dashboard.log`
- 停止所有服务：`./stop.sh`

---

*文档最后更新时间：2026-07-28*
