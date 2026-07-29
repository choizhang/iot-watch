# IoT 物联网系统 - 前端项目 (Frontend)

本项目是基于 **Vue 3 + TypeScript + Vite** 构建的 IoT 物联网系统前端展示与管理平台。项目集成了高德地图（AMap）进行设备 GPS 定位展示，并使用 Tailwind CSS 与 Lucide Icons 打造现代化、响应式的用户界面。

## 🛠️ 技术栈

*   **核心框架**：Vue 3 (Composition API, `<script setup>`)
*   **构建工具**：Vite
*   **编程语言**：TypeScript
*   **状态管理**：Pinia
*   **路由管理**：Vue Router
*   **样式库**：Tailwind CSS + PostCSS
*   **网络请求**：Axios
*   **地图服务**：高德地图 JSAPI Loader (`@amap/amap-jsapi-loader`)
*   **图标库**：Lucide Vue Next

## 📂 项目结构

```text
frontend/
├── public/              # 静态资源
├── src/
│   ├── api/             # 接口请求定义
│   ├── assets/          # 静态图片、字体等
│   ├── components/      # 通用 UI 组件
│   ├── composables/     # 自定义组合式函数 (Composables)
│   ├── lib/             # 第三方库初始化/配置
│   ├── pages/           # 页面视图组件
│   ├── router/          # 路由配置
│   ├── store/           # Pinia 状态管理
│   ├── utils/           # 工具函数
│   ├── views/           # 视图组件
│   ├── App.vue          # 根组件
│   ├── main.ts          # 入口文件
│   └── style.css        # 全局样式与 Tailwind 引入
├── index.html           # HTML 模板
├── package.json         # 依赖与脚本配置
├── tsconfig.json        # TypeScript 配置
└── vite.config.ts       # Vite 配置
```

## 🚀 快速开始

### 前提条件

确保您的开发环境已安装以下工具：
*   [Node.js](https://nodejs.org/) (推荐 v18 或更高版本)
*   [pnpm](https://pnpm.io/) (推荐) 或 npm / yarn

### 安装依赖

```bash
pnpm install
# 或者
npm install
# 或者
yarn install
```

### 本地开发

启动本地开发服务器（支持热重载）：

```bash
pnpm dev
```

启动后，在浏览器中打开命令行中输出的地址（默认为 `http://localhost:5173`）即可访问。

### 构建生产版本

打包并压缩代码以用于生产环境部署：

```bash
pnpm build
```

打包生成的文件将存放在 `dist` 目录中。

### 预览生产版本

在本地预览打包后的生产版本：

```bash
pnpm preview
```

### 代码规范与检查

*   **TypeScript 类型检查**：`pnpm check`
*   **ESLint 代码检查**：`pnpm lint`
*   **ESLint 自动修复**：`pnpm lint:fix`
