# 社区养老智慧安防指挥控制中心 (B端大屏)

基于 Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS + 高德地图 JS API v2.0 开发的独立社区/机构养老指挥控制大屏系统（B端）。与 C 端家属移动端彻底解耦，可独立部署于监控中心、安防控制室或局域网机房。

---

## 🌟 核心功能与架构特性

### 1. 沉浸式暗色科技指挥视觉 (Slate-950 Command Theme)
- 针对 16:9 / 21:9 监控控制大屏及 PC 监控台深度优化的 Slate-950 高对比暗色科技 UI。
- 水波纹红光 SOS 告警脉冲动画、发光气泡与全屏高灵敏度响应。

### 2. 100 台全场景 IoT 设备 Mock 模拟器
- **全港 18 区覆盖**：真实散布于中西区、湾仔、油尖旺、荃湾、葵青、沙田、屯门、元朗、观塘等香港全境。
- **全场景覆盖**：
  - 设备状态：🟢 在线 (70%)、🔴 SOS 告警 (10%)、⚪ 离线失联 (20%)；
  - 定位技术：`GPS 卫星高精` (±5m)、`WIFI 室内定位` (±20m)、`LBS 基站定位` (±250m)。
  - 体征遥测：心率 (60-145bpm)、血压、血氧 (94-99%)、表体温度、步数、锁定卫星数、信号 RSSI。

### 3. 解耦的三大独立详情 Tab 视角 (`/device/:imei`)
- **📍 实时精细定位**：`Zoom 18` 最大门牌/楼栋精细视角，高德逆地理中文地址全量平铺，物联网硬件定位遥测网格。
- **🛣️ 历史轨迹追溯**：`今日(24h)` / `昨天` / `近3天` 独立时间选择器，纯净 GPS Polyline 路线，带 `1x/2x/4x` 轨迹动画播放控制器与精确时间轴日志。
- **📊 30天活动热力与生活圈分析**：`近7天` / `近30天` / `近90天` 中长期统计，高德炫彩热力图 (`AMap.HeatMap`) 结合常去地标 TOP 4 榜单与习惯生活圈安全评估。

### 4. 地图多主题一键无缝切换与视角重置
- **🌙 科技深色 (`Dark`)**：监控大屏首选，告警发光感极致；
- **☀️ 标准彩色 (`Normal`)**：地形、海域、绿地与街道 100% 清晰可见；
- **🛰️ 卫星实景 (`Satellite`)**：高德官方卫星航拍图层，核对建筑物屋顶与山体；
- **🎯 重置全景 (`Reset Overview`)**：一键平滑还原至全港全区俯瞰视角 (`Zoom 11`)。

### 5. 性能优化与 0ms 缓存机制
- **单例高德 SDK 缓存 (`amap.ts`)**：全局 Promise 预热，页面切换 0ms 脚本下载等待。
- **Vue `<KeepAlive>` 常驻**：路由切换时 DOM 节点与地图实例常驻内存，实现毫秒级秒开。

---

## 🚀 端口与运行

- **开发服务端口**: `http://localhost:5174`
- **构建输出目录**: `dashboard/dist/`

```bash
# 1. 进入大屏项目目录
cd dashboard

# 2. 安装依赖
pnpm install

# 3. 本地启动开发服务 (端口 5174)
pnpm dev

# 4. 生产环境编译打包
pnpm build
```

---

## 🏗️ 独立部署 (Nginx 静态部署)

产物为纯静态 HTML/CSS/JS 文件，可通过 Nginx 或 Docker 进行独立部署：

```nginx
server {
    listen 80;
    server_name dashboard.iot.example.com;

    location / {
        root /var/www/dashboard/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://iot-backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
