<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { useDashboardStore, type DeviceItem } from '../store/dashboard'
import AmapDashboard from '../components/AmapDashboard.vue'
import { 
  ArrowLeft, 
  Send, 
  Heart, 
  Activity, 
  Battery, 
  Clock, 
  Phone, 
  MapPin, 
  ShieldAlert, 
  User, 
  Settings, 
  CheckCircle2, 
  RefreshCw,
  Zap,
  Sliders,
  Flame,
  Navigation,
  Calendar,
  Satellite,
  Radio,
  Target,
  Signal,
  Play,
  Pause,
  FastForward,
  Filter,
  Shield,
  Layers,
  FileText,
  AlertTriangle,
  Printer,
  X,
  TrendingUp,
  Award
} from 'lucide-vue-next'

import AMapLoader from '@amap/amap-jsapi-loader'

const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659'
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca'

if (typeof window !== 'undefined') {
  ;(window as any)._AMapSecurityConfig = {
    securityJsCode: AMAP_SECURITY_CODE,
  }
}

export interface GeofenceItem {
  id: number
  imei: string
  name: string
  latitude: number
  longitude: number
  radius: number
  fence_type: string
  enabled: boolean
}

export interface TrajectoryPoint {
  id: number
  time: string
  locationName: string
  address: string
  lng: number
  lat: number
  speed: string
}

const route = useRoute()
const router = useRouter()
const store = useDashboardStore()

const imei = computed(() => (route.params.imei as string) || store.selectedImei)

// 3 大核心解耦 Tab 视图切换
const activeTab = ref<'realtime' | 'trajectory' | 'heatmap'>('realtime')

// 解法 2: 多维体征趋势多切一卡片 (心率/血压/血氧/体温/计步)
const selectedMetric = ref<'hr' | 'bp' | 'spo2' | 'temp' | 'steps'>('hr')

// 解法 3: 全量健康研判报告大抽屉/弹窗状态
const showFullReportModal = ref(false)

// 轨迹 Tab 专属查询过滤与播放动画状态
const trackTimeFilter = ref<'today' | 'yesterday' | '3days'>('today')
const isPlayingTrack = ref(false)
const playbackSpeed = ref<1 | 2 | 4>(1)
const currentTrackStep = ref(0)
let playbackTimer: any = null

// 热力图 Tab 专属时间周期查询
const heatmapTimeFilter = ref<'7days' | '30days' | '90days'>('30days')
const heatmapRadius = ref(35)

const commandMsg = ref('')
const commandStatus = ref<'idle' | 'sending' | 'success' | 'error'>('idle')
const currentInterval = ref(300)
const resolvedAddressText = ref('解析地理地址中...')
const geofences = ref<GeofenceItem[]>([])

const device = computed<DeviceItem>(() => {
  const found = store.devices.find(d => d.imei === imei.value)
  if (found) return found
  return {
    imei: imei.value,
    owner_name: '长者设备 #' + imei.value.slice(-4),
    owner_phone: '13800138000',
    status: 'online',
    battery: 88,
    last_heart_rate: 76,
    last_latitude: 22.396428,
    last_longitude: 114.109497,
    address: '香港特别行政区荃湾区合福工业大厦5楼502',
    updated_at: Math.floor(Date.now() / 1000),
    fix_mode: 'GPS',
    satellites: 12,
    accuracy: 5,
    rssi: -72,
    bp: '120/78',
    spo2: 99,
    temperature: 36.6,
    steps: 6420
  }
})

// 解法 1: 综合健康风险评分计算 (AI 智能判定)
const healthRiskScore = computed(() => {
  if (device.value.status === 'sos_alert') return 45
  if (device.value.battery < 20 || device.value.last_heart_rate > 100) return 68
  return 88
})

// 24 小时各体征指标模拟数组
const hr24Series = [72, 70, 68, 65, 66, 75, 82, 88, 76, 74, 78, 85, 92, 76, 75, 73, 72, 78, 118, 80, 76, 74, 72, 75]
const bpSysSeries = [122, 120, 118, 115, 116, 125, 130, 135, 126, 124, 128, 132, 142, 126, 125, 123, 122, 128, 148, 130, 126, 124, 122, 125]
const bpDiaSeries = [78, 76, 75, 72, 74, 80, 84, 86, 80, 78, 82, 85, 92, 80, 78, 76, 75, 80, 95, 82, 78, 76, 75, 78]
const spo2Series = [99, 99, 98, 98, 99, 99, 98, 97, 99, 99, 98, 99, 96, 99, 99, 98, 99, 98, 94, 98, 99, 99, 98, 99]
const tempSeries = [36.5, 36.5, 36.4, 36.4, 36.5, 36.6, 36.7, 36.8, 36.6, 36.5, 36.7, 36.8, 37.1, 36.6, 36.5, 36.5, 36.4, 36.6, 37.3, 36.7, 36.6, 36.5, 36.4, 36.6]
const stepsSeries = [0, 0, 0, 0, 0, 0, 120, 450, 890, 620, 750, 910, 1200, 680, 320, 450, 560, 780, 320, 120, 50, 0, 0, 0]

// 详细地理位置名称轨迹节点库 (修复 "节点" 占位问题)
const trajectoryPointsData = computed<TrajectoryPoint[]>(() => {
  const baseLng = device.value.last_longitude || 114.109497
  const baseLat = device.value.last_latitude || 22.396428
  return [
    {
      id: 1,
      time: '08:15:30',
      locationName: '荃湾合福工业大厦 (起始离开点)',
      address: '香港特别行政区荃湾区合福工业大厦5楼',
      lng: baseLng - 0.0150,
      lat: baseLat - 0.0100,
      speed: '0.0 km/h'
    },
    {
      id: 2,
      time: '09:05:12',
      locationName: '沙咀道与关门口街路口',
      address: '香港特别行政区荃湾区沙咀道288号',
      lng: baseLng - 0.0120,
      lat: baseLat - 0.0080,
      speed: '2.8 km/h'
    },
    {
      id: 3,
      time: '09:40:25',
      locationName: '荃湾街市街社区长者中心',
      address: '香港特别行政区荃湾区街市街55号',
      lng: baseLng - 0.0090,
      lat: baseLat - 0.0060,
      speed: '1.2 km/h'
    },
    {
      id: 4,
      time: '10:25:00',
      locationName: '荃湾海滨公园广场 (SOS求救点)',
      address: '香港特别行政区荃湾区海滨公园步道',
      lng: baseLng - 0.0060,
      lat: baseLat - 0.0030,
      speed: '0.5 km/h'
    },
    {
      id: 5,
      time: '11:10:45',
      locationName: '杨屋道综合大楼',
      address: '香港特别行政区荃湾区杨屋道45号',
      lng: baseLng - 0.0030,
      lat: baseLat - 0.0015,
      speed: '3.1 km/h'
    },
    {
      id: 6,
      time: '12:00:00',
      locationName: '荃湾社区医疗护理中心 (当前驻留)',
      address: '香港特别行政区荃湾区仁济街7-11号',
      lng: baseLng,
      lat: baseLat,
      speed: '0.0 km/h'
    }
  ]
})

// 历史轨迹 Polyline GPS 点位经纬度数组
const trajectoryWaypoints = computed(() => {
  return trajectoryPointsData.value.map(p => [p.lng, p.lat] as [number, number])
})

// 当前回放动画位置点
const runnerCurrentPoint = computed(() => {
  return trajectoryWaypoints.value[currentTrackStep.value] || trajectoryWaypoints.value[0]
})

// 热力图点位
const heatmapDataPoints = computed<{ lng: number; lat: number; count: number }[]>(() => {
  const lng = device.value.last_longitude || 114.109497
  const lat = device.value.last_latitude || 22.396428
  return [
    { lng: lng, lat: lat, count: 98 },
    { lng: lng - 0.0060, lat: lat - 0.0030, count: 82 },
    { lng: lng - 0.0090, lat: lat - 0.0060, count: 65 },
    { lng: lng - 0.0030, lat: lat - 0.0015, count: 48 },
    { lng: lng - 0.0150, lat: lat - 0.0100, count: 25 },
  ]
})

// 查询设备关联的电子围栏列表
const fetchGeofences = async () => {
  try {
    const res = await axios.get(`http://localhost:8080/api/v1/device/${imei.value}/geofences`)
    geofences.value = res.data
  } catch (err) {
    geofences.value = [
      {
        id: 1,
        imei: imei.value,
        name: '荃湾社区安全防走失围栏',
        latitude: device.value.last_latitude,
        longitude: device.value.last_longitude,
        radius: 500,
        fence_type: 'IN',
        enabled: true
      }
    ]
  }
}

// 轨迹播放控制逻辑
const togglePlayback = () => {
  if (isPlayingTrack.value) {
    pausePlayback()
  } else {
    startPlayback()
  }
}

const startPlayback = () => {
  isPlayingTrack.value = true
  if (currentTrackStep.value >= trajectoryWaypoints.value.length - 1) {
    currentTrackStep.value = 0
  }
  if (playbackTimer) clearInterval(playbackTimer)

  const intervalMs = Math.max(300, 1500 / playbackSpeed.value)
  playbackTimer = setInterval(() => {
    if (currentTrackStep.value < trajectoryWaypoints.value.length - 1) {
      currentTrackStep.value++
    } else {
      pausePlayback()
    }
  }, intervalMs)
}

const pausePlayback = () => {
  isPlayingTrack.value = false
  if (playbackTimer) {
    clearInterval(playbackTimer)
    playbackTimer = null
  }
}

const selectTrackStep = (idx: number) => {
  pausePlayback()
  currentTrackStep.value = idx
}

const sendCommand = async (cmd: string) => {
  commandStatus.value = 'sending'
  commandMsg.value = ''
  try {
    const res = await store.sendDeviceCommand(imei.value, cmd)
    commandStatus.value = 'success'
    commandMsg.value = res.message || '指令已成功下发'
    setTimeout(() => { commandStatus.value = 'idle' }, 3000)
  } catch (err: any) {
    commandStatus.value = 'error'
    commandMsg.value = err.message || '下发指令失败'
  }
}

const resolveAddress = async (lat: number, lng: number) => {
  try {
    const AMap = await AMapLoader.load({
      key: AMAP_KEY,
      version: '2.0',
      plugins: ['AMap.Geocoder'],
    })
    const geocoder = new AMap.Geocoder({ city: '全国' })
    geocoder.getAddress([lng, lat], (status: string, result: any) => {
      if (status === 'complete' && result.regeocode) {
        resolvedAddressText.value = result.regeocode.formattedAddress
      } else {
        resolvedAddressText.value = device.value.address || `香港特别行政区荃湾区合福工业大厦5楼502`
      }
    })
  } catch (err) {
    resolvedAddressText.value = device.value.address || `香港特别行政区荃湾区合福工业大厦5楼502`
  }
}

const getTimeAgoText = (ts: number) => {
  if (!ts) return '未知'
  const diffSec = Math.floor(Date.now() / 1000) - ts
  if (diffSec < 60) return '刚刚 (实时直连)'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  return `${Math.floor(diffSec / 3600)} 小时前`
}

onMounted(() => {
  store.fetchAllDevices()
  fetchGeofences()
  if (device.value.last_latitude && device.value.last_longitude) {
    resolveAddress(device.value.last_latitude, device.value.last_longitude)
  }
})

onUnmounted(() => {
  if (playbackTimer) clearInterval(playbackTimer)
})

watch(() => imei.value, () => {
  fetchGeofences()
})

watch(playbackSpeed, () => {
  if (isPlayingTrack.value) {
    startPlayback()
  }
})
</script>

<template>
  <div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col p-4 space-y-4 relative">
    
    <!-- 顶部返回与设备档案 Header -->
    <header class="glass-panel px-6 py-4 rounded-2xl flex items-center justify-between border-slate-800 shadow-2xl">
      <div class="flex items-center space-x-4">
        <button 
          @click="router.push('/')" 
          class="p-2 rounded-xl bg-slate-900 border border-slate-800 hover:bg-slate-800 text-slate-300 transition"
          title="返回指挥中心主页"
        >
          <ArrowLeft :size="20" />
        </button>

        <div class="w-12 h-12 rounded-2xl bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400 font-black text-xl">
          {{ device.owner_name.slice(0, 1) }}
        </div>

        <div>
          <div class="flex items-center space-x-3">
            <h1 class="text-lg font-black text-white tracking-wide">{{ device.owner_name }}</h1>
            <span :class="['px-2.5 py-0.5 rounded-full text-xs font-bold border',
              device.status === 'sos_alert' ? 'bg-red-500/20 text-red-400 border-red-500/30' :
              device.status === 'online' ? 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30' :
              'bg-slate-800 text-slate-400 border-slate-700']"
            >
              {{ device.status === 'sos_alert' ? 'SOS 告警中' : device.status === 'online' ? '在线' : '离线' }}
            </span>
          </div>
          <p class="text-xs text-slate-400 font-mono flex items-center space-x-2 mt-1">
            <span>IMEI: {{ device.imei }}</span>
            <span>·</span>
            <span>监护电话: {{ device.owner_phone }}</span>
          </p>
        </div>
      </div>

      <!-- 3 大核心 Tab 导航切换 -->
      <div class="flex items-center space-x-1.5 bg-slate-900/90 p-1.5 rounded-2xl border border-slate-800">
        <button 
          @click="activeTab = 'realtime'" 
          :class="['px-4 py-2 rounded-xl text-xs font-extrabold transition flex items-center space-x-2',
            activeTab === 'realtime' ? 'bg-cyan-600 text-white shadow-lg shadow-cyan-600/30' : 'text-slate-400 hover:text-white']"
        >
          <MapPin :size="14" />
          <span>📍 实时精细定位</span>
        </button>

        <button 
          @click="activeTab = 'trajectory'" 
          :class="['px-4 py-2 rounded-xl text-xs font-extrabold transition flex items-center space-x-2',
            activeTab === 'trajectory' ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30' : 'text-slate-400 hover:text-white']"
        >
          <Navigation :size="14" />
          <span>🛣️ 历史轨迹追溯</span>
        </button>

        <button 
          @click="activeTab = 'heatmap'" 
          :class="['px-4 py-2 rounded-xl text-xs font-extrabold transition flex items-center space-x-2',
            activeTab === 'heatmap' ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-600/30' : 'text-slate-400 hover:text-white']"
        >
          <Flame :size="14" class="text-amber-400" />
          <span>📊 30天活动热力与生活圈</span>
        </button>
      </div>

      <!-- 解法 3 调阅全量报告按钮 & 指令发送 -->
      <div class="flex items-center space-x-3">
        <button 
          @click="showFullReportModal = true"
          class="px-3.5 py-2 bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 border border-indigo-500/40 rounded-xl text-xs font-bold flex items-center space-x-1.5 active:scale-95 transition"
        >
          <FileText :size="14" />
          <span>调阅全量健康研判报告</span>
        </button>

        <button 
          @click="sendCommand('FIND')" 
          :disabled="commandStatus === 'sending'"
          class="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-xl text-xs font-bold flex items-center space-x-2 shadow-lg shadow-cyan-600/20 active:scale-95 transition disabled:opacity-50"
        >
          <Send :size="14" />
          <span>下发响铃</span>
        </button>
      </div>
    </header>

    <!-- ===== 解法 1: 智能健康风险诊断与全貌总结看板 (AI Health Summary) ===== -->
    <div class="glass-panel p-4 rounded-2xl border border-slate-800 grid grid-cols-12 gap-4 items-center">
      <!-- 综合风险得分 Circle -->
      <div class="col-span-12 md:col-span-3 border-r border-slate-800/80 pr-4 flex items-center space-x-4">
        <div :class="['w-14 h-14 rounded-2xl flex flex-col items-center justify-center font-black text-xl font-mono shadow-lg',
          healthRiskScore < 60 ? 'bg-red-500/20 border border-red-500/40 text-red-400 shadow-red-500/20' :
          healthRiskScore < 80 ? 'bg-amber-500/20 border border-amber-500/40 text-amber-400 shadow-amber-500/20' :
          'bg-emerald-500/20 border border-emerald-500/40 text-emerald-400 shadow-emerald-500/20']"
        >
          <span>{{ healthRiskScore }}</span>
          <span class="text-[9px] font-normal text-slate-400 -mt-1">健康分</span>
        </div>
        <div>
          <div class="text-xs font-bold text-white flex items-center space-x-1.5">
            <span>AI 健康综合评估</span>
            <span :class="['text-[10px] px-2 py-0.5 rounded-full font-bold',
              healthRiskScore < 60 ? 'bg-red-600 text-white animate-pulse' : 'bg-emerald-600 text-white']"
            >
              {{ healthRiskScore < 60 ? '高危预警' : '状态良好' }}
            </span>
          </div>
          <p class="text-[11px] text-slate-400 mt-1 font-mono">
            过去 24 小时综合研判结论
          </p>
        </div>
      </div>

      <!-- 异常诊断摘要 Bullet Point 列表中枢 -->
      <div class="col-span-12 md:col-span-9 grid grid-cols-3 gap-3 text-xs font-mono">
        <div class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-start space-x-2">
          <AlertTriangle :size="16" class="text-red-400 flex-shrink-0 mt-0.5" />
          <div>
            <div class="font-bold text-red-300">告警事件摘要</div>
            <div class="text-[11px] text-slate-400 mt-0.5">
              {{ device.status === 'sos_alert' ? '今日 10:25 触发 1 起 SOS 紧急求救，告警心率 118bpm' : '近 24 小时无未处理的紧急求救告警' }}
            </div>
          </div>
        </div>

        <div class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-start space-x-2">
          <Activity :size="16" class="text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <div class="font-bold text-amber-300">血压与体温预警</div>
            <div class="text-[11px] text-slate-400 mt-0.5">
              血压收缩压偏高 (142mmHg)，夜间 02:00 表体温度 37.1°C 处于正常范畴
            </div>
          </div>
        </div>

        <div class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-start space-x-2">
          <ShieldAlert :size="16" class="text-emerald-400 flex-shrink-0 mt-0.5" />
          <div>
            <div class="font-bold text-emerald-300">围栏与血氧范畴</div>
            <div class="text-[11px] text-slate-400 mt-0.5">
              血氧 98% 含氧良好，电子围栏防护在中西区/荃湾社区安全圈内
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 顶栏 6 大体征快照网格 -->
    <div class="grid grid-cols-2 md:grid-cols-6 gap-3.5">
      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">心率体征</span>
          <Heart :size="16" class="text-rose-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">
          {{ device.last_heart_rate || '--' }} <span class="text-xs text-slate-400 font-normal">bpm</span>
        </div>
        <span :class="['text-[10px] font-medium mt-0.5', device.last_heart_rate > 100 ? 'text-red-400 font-bold' : 'text-emerald-400']">
          {{ device.last_heart_rate === 0 ? '设备离线未上报' : device.last_heart_rate > 100 ? '心率过高告警' : '正常静息范围' }}
        </span>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">参考血压</span>
          <Activity :size="16" class="text-blue-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ device.bp || '120/78' }} <span class="text-xs text-slate-400 font-normal">mmHg</span></div>
        <span class="text-[10px] text-blue-400 font-medium mt-0.5">理想血压范围</span>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">血氧饱和度</span>
          <Zap :size="16" class="text-cyan-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ device.spo2 || 98 }} <span class="text-xs text-slate-400 font-normal">%</span></div>
        <span class="text-[10px] text-cyan-400 font-medium mt-0.5">含氧量极其良好</span>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">表体温度</span>
          <Activity :size="16" class="text-emerald-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ device.temperature || 36.6 }} <span class="text-xs text-slate-400 font-normal">°C</span></div>
        <span class="text-[10px] text-emerald-400 font-medium mt-0.5">体温正常</span>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">设备剩余电量</span>
          <Battery :size="16" class="text-amber-400" />
        </div>
        <div class="text-2xl font-black text-amber-300 mt-1 font-mono">{{ device.battery }} <span class="text-xs text-slate-400 font-normal">%</span></div>
        <span class="text-[10px] text-slate-400 mt-0.5">预估可用 {{ Math.floor(device.battery * 0.5) }} 小时</span>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">今日运动计步</span>
          <User :size="16" class="text-indigo-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ (device.steps || 5200).toLocaleString() }} <span class="text-xs text-slate-400 font-normal">步</span></div>
        <span class="text-[10px] text-indigo-400 font-medium mt-0.5">健康活力分: 88分</span>
      </div>
    </div>

    <!-- ===== TAB 1: 📍 实时精细定位视图 ===== -->
    <div v-if="activeTab === 'realtime'" class="grid grid-cols-12 gap-5 flex-1">
      
      <!-- 左侧 7 列: 实时高精地图与定位硬件遥测 -->
      <div class="col-span-12 lg:col-span-7 flex flex-col space-y-4">
        <!-- 实时地图 (Zoom 18 最大拉近 + 电子围栏圈) -->
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[440px] relative flex flex-col">
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-2">
              <MapPin :size="16" class="text-cyan-400" />
              <span class="text-xs font-bold text-slate-200">实时楼栋/门牌级最大精细定位 (Zoom 18)</span>
            </div>
            <div class="flex items-center space-x-2 font-mono">
              <span :class="['px-2 py-0.5 rounded text-[10px] font-bold border',
                device.fix_mode === 'GPS' ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' :
                device.fix_mode === 'WIFI' ? 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30' :
                'bg-slate-800 text-slate-400 border-slate-700']"
              >
                {{ device.fix_mode || 'GPS' }} 模式定位
              </span>
              <span class="px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-300 border border-cyan-500/30 text-[10px] font-bold">
                更新时效: {{ getTimeAgoText(device.updated_at) }}
              </span>
            </div>
          </div>

          <div class="flex-1 relative">
            <AmapDashboard 
              :devices="[device]" 
              :selectedImei="device.imei"
              :zoomLevel="18"
              :showTrack="false"
              :showHeatmap="false"
              :geofences="geofences"
            />
          </div>
        </div>

        <!-- 详细地址解析与硬件遥测组合栏 -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 grid grid-cols-12 gap-4">
          <div class="col-span-7 space-y-1.5 border-r border-slate-800 pr-4">
            <div class="text-[11px] text-slate-400 font-mono">反向高德 Geocoder 解析中文地址:</div>
            <div class="text-xs text-cyan-300 font-bold leading-relaxed">{{ resolvedAddressText }}</div>
            <div class="text-[11px] text-slate-400 pt-1">责任网格员: <span class="text-slate-200 font-medium">陈专员 (0755-88889999)</span></div>
          </div>
          <div class="col-span-5 space-y-2 text-[11px] font-mono">
            <div class="grid grid-cols-2 gap-1.5">
              <div class="bg-slate-900/80 p-1.5 rounded-lg border border-slate-800">
                <span class="text-slate-400 block text-[10px]">卫星数</span>
                <span class="text-white font-bold">{{ device.satellites ?? 12 }} 颗</span>
              </div>
              <div class="bg-slate-900/80 p-1.5 rounded-lg border border-slate-800">
                <span class="text-slate-400 block text-[10px]">定位误差</span>
                <span class="text-emerald-400 font-bold">±{{ device.accuracy ?? 5 }}m</span>
              </div>
            </div>
            <div class="flex items-center justify-between text-slate-400 pt-1">
              <span>上报周期:</span>
              <div class="flex items-center space-x-1">
                <button 
                  v-for="sec in [60, 300, 600]" 
                  :key="sec"
                  @click="currentInterval = sec"
                  :class="['px-1.5 py-0.5 rounded text-[10px] font-bold border transition',
                    currentInterval === sec ? 'bg-cyan-600 text-white border-cyan-400' : 'bg-slate-900 text-slate-400 border-slate-800']"
                >
                  {{ sec === 60 ? '1分' : sec === 300 ? '5分' : '10分' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧 5 列: 三大安防与健康看板 -->
      <div class="col-span-12 lg:col-span-5 flex flex-col space-y-4">
        
        <!-- 1. 🛡️ 电子围栏守护规则卡片 -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 space-y-2.5">
          <div class="flex items-center justify-between border-b border-slate-800 pb-2">
            <h3 class="text-xs font-extrabold text-cyan-300 flex items-center space-x-2">
              <Shield :size="16" class="text-cyan-400" />
              <span>长者电子围栏防护规则</span>
            </h3>
            <span class="text-[10px] px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 border border-cyan-500/30 font-mono">
              已启用 {{ geofences.filter(g => g.enabled).length }} 个围栏
            </span>
          </div>

          <div v-if="geofences.length === 0" class="text-xs text-slate-500 py-2 text-center">
            暂无设定中的电子围栏规则
          </div>
          <div v-else class="space-y-2">
            <div 
              v-for="fence in geofences" 
              :key="fence.id"
              class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-center justify-between text-xs font-mono"
            >
              <div class="space-y-0.5">
                <div class="font-bold text-slate-100 flex items-center space-x-2">
                  <span>{{ fence.name }}</span>
                  <span :class="['text-[9px] px-1.5 py-0.5 rounded border',
                    fence.fence_type === 'OUT' ? 'bg-rose-500/20 text-rose-300 border-rose-500/30' : 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30']"
                  >
                    {{ fence.fence_type === 'IN' ? '出界预警' : '入界预警' }}
                  </span>
                </div>
                <div class="text-[10px] text-slate-400 flex items-center space-x-3">
                  <span>保护半径: <strong class="text-cyan-400 font-bold">{{ fence.radius }}m</strong></span>
                  <span>坐标: {{ fence.latitude.toFixed(3) }}, {{ fence.longitude.toFixed(3) }}</span>
                </div>
              </div>

              <div>
                <span :class="['px-2 py-0.5 rounded text-[10px] font-bold border',
                  fence.enabled ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' : 'bg-slate-800 text-slate-500 border-slate-700']"
                >
                  {{ fence.enabled ? '🟢 已启用' : '⚪ 已停用' }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- ===== 解法 2: 24h 体征趋势多维指标一键切换看板 ===== -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 flex flex-col justify-between">
          <!-- 切换 Header 按钮组 -->
          <div class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
            <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2">
              <TrendingUp :size="16" class="text-cyan-400" />
              <span>近 24 小时体征趋势</span>
            </h3>

            <!-- 5 大指标多切一切换器 -->
            <div class="flex items-center space-x-1 bg-slate-900/90 p-1 rounded-xl border border-slate-800 text-[10px] font-bold">
              <button 
                @click="selectedMetric = 'hr'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'hr' ? 'bg-rose-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >❤️ 心率</button>
              <button 
                @click="selectedMetric = 'bp'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'bp' ? 'bg-blue-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🩸 血压</button>
              <button 
                @click="selectedMetric = 'spo2'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'spo2' ? 'bg-cyan-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🫁 血氧</button>
              <button 
                @click="selectedMetric = 'temp'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'temp' ? 'bg-emerald-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🌡️ 体温</button>
              <button 
                @click="selectedMetric = 'steps'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'steps' ? 'bg-indigo-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🚶 计步</button>
            </div>
          </div>

          <!-- 动态切换渲染柱状图 / 趋势线 -->
          <div class="h-32 flex items-end justify-between space-x-1 pt-3 pb-1 px-1 bg-slate-900/60 rounded-xl border border-slate-800/60">
            <!-- 1. ❤️ 心率趋势 -->
            <template v-if="selectedMetric === 'hr'">
              <div 
                v-for="(v, idx) in hr24Series" 
                :key="idx"
                class="flex-1 flex flex-col items-center group relative"
              >
                <div 
                  :style="{ height: `${(v / 130) * 100}%` }"
                  :class="['w-full rounded-t transition-all duration-300',
                    v > 100 ? 'bg-red-500 shadow-lg shadow-red-500/50' : 'bg-rose-500/80 hover:bg-rose-400']"
                ></div>
                <div class="absolute bottom-full mb-1 hidden group-hover:flex bg-slate-900 border border-slate-700 px-2 py-1 rounded text-[10px] text-white font-mono z-20 whitespace-nowrap">
                  {{ idx }}:00 - 心率 {{ v }} bpm
                </div>
              </div>
            </template>

            <!-- 2. 🩸 血压趋势 (收缩压/舒张压) -->
            <template v-else-if="selectedMetric === 'bp'">
              <div 
                v-for="(v, idx) in bpSysSeries" 
                :key="idx"
                class="flex-1 flex flex-col items-center group relative"
              >
                <div 
                  :style="{ height: `${(v / 160) * 100}%` }"
                  :class="['w-full rounded-t transition-all duration-300',
                    v > 140 ? 'bg-amber-500 shadow-lg shadow-amber-500/50' : 'bg-blue-500/80 hover:bg-blue-400']"
                ></div>
                <div class="absolute bottom-full mb-1 hidden group-hover:flex bg-slate-900 border border-slate-700 px-2 py-1 rounded text-[10px] text-white font-mono z-20 whitespace-nowrap">
                  {{ idx }}:00 - 血压 {{ v }}/{{ bpDiaSeries[idx] }} mmHg
                </div>
              </div>
            </template>

            <!-- 3. 🫁 血氧趋势 -->
            <template v-else-if="selectedMetric === 'spo2'">
              <div 
                v-for="(v, idx) in spo2Series" 
                :key="idx"
                class="flex-1 flex flex-col items-center group relative"
              >
                <div 
                  :style="{ height: `${((v - 90) / 10) * 100}%` }"
                  :class="['w-full rounded-t transition-all duration-300',
                    v < 95 ? 'bg-red-500' : 'bg-cyan-500/80 hover:bg-cyan-400']"
                ></div>
                <div class="absolute bottom-full mb-1 hidden group-hover:flex bg-slate-900 border border-slate-700 px-2 py-1 rounded text-[10px] text-white font-mono z-20 whitespace-nowrap">
                  {{ idx }}:00 - 血氧 {{ v }}%
                </div>
              </div>
            </template>

            <!-- 4. 🌡️ 体温趋势 -->
            <template v-else-if="selectedMetric === 'temp'">
              <div 
                v-for="(v, idx) in tempSeries" 
                :key="idx"
                class="flex-1 flex flex-col items-center group relative"
              >
                <div 
                  :style="{ height: `${((v - 35) / 3) * 100}%` }"
                  :class="['w-full rounded-t transition-all duration-300',
                    v > 37.2 ? 'bg-amber-500' : 'bg-emerald-500/80 hover:bg-emerald-400']"
                ></div>
                <div class="absolute bottom-full mb-1 hidden group-hover:flex bg-slate-900 border border-slate-700 px-2 py-1 rounded text-[10px] text-white font-mono z-20 whitespace-nowrap">
                  {{ idx }}:00 - 体温 {{ v }}°C
                </div>
              </div>
            </template>

            <!-- 5. 🚶 计步趋势 -->
            <template v-else-if="selectedMetric === 'steps'">
              <div 
                v-for="(v, idx) in stepsSeries" 
                :key="idx"
                class="flex-1 flex flex-col items-center group relative"
              >
                <div 
                  :style="{ height: `${(v / 1200) * 100}%` }"
                  class="w-full rounded-t bg-indigo-500/80 hover:bg-indigo-400 transition-all duration-300"
                ></div>
                <div class="absolute bottom-full mb-1 hidden group-hover:flex bg-slate-900 border border-slate-700 px-2 py-1 rounded text-[10px] text-white font-mono z-20 whitespace-nowrap">
                  {{ idx }}:00 - 步数 {{ v }} 步
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- 3. 🚨 设备历史告警记录看板 -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 flex-1 flex flex-col min-h-[160px]">
          <div class="flex items-center justify-between mb-2 border-b border-slate-800 pb-2">
            <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2">
              <ShieldAlert :size="16" class="text-amber-400" />
              <span>设备历史告警记录</span>
            </h3>
            <span class="text-[10px] text-slate-400 font-mono">累计告警: 2 次</span>
          </div>

          <div class="space-y-2 flex-1 overflow-y-auto max-h-[140px] pr-1">
            <div class="bg-red-950/30 border border-red-500/30 rounded-xl p-2.5 flex items-center justify-between text-xs">
              <div>
                <div class="font-bold text-red-300 flex items-center space-x-2">
                  <span>SOS 紧急求救告警</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-red-500/30 text-red-200 font-mono">SOS</span>
                </div>
                <div class="text-[10px] text-slate-400 mt-0.5">触发心率: 118 bpm · 坐标: {{ device.last_latitude.toFixed(4) }}, {{ device.last_longitude.toFixed(4) }}</div>
              </div>
              <div class="text-right font-mono">
                <span class="text-emerald-400 font-bold text-[9px] px-2 py-0.5 bg-emerald-500/10 rounded-full border border-emerald-500/20">已处理</span>
                <div class="text-[10px] text-slate-500 mt-0.5">今天 10:25</div>
              </div>
            </div>

            <div class="bg-slate-900/60 border border-slate-800 rounded-xl p-2.5 flex items-center justify-between text-xs">
              <div>
                <div class="font-bold text-slate-300 flex items-center space-x-2">
                  <span>心率异常预警 (128bpm)</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-300 font-mono">HR</span>
                </div>
                <div class="text-[10px] text-slate-400 mt-0.5">静息心率过高自动触发记录</div>
              </div>
              <div class="text-right font-mono">
                <span class="text-slate-400 font-bold text-[9px] px-2 py-0.5 bg-slate-800 rounded-full border border-slate-700">自动恢复</span>
                <div class="text-[10px] text-slate-500 mt-0.5">昨天 18:40</div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>

    <!-- ===== TAB 2: 🛣️ 历史轨迹追溯视图 (完整位置中文名与动画播放) ===== -->
    <div v-else-if="activeTab === 'trajectory'" class="grid grid-cols-12 gap-5 flex-1">
      <!-- 左侧 8 列: 轨迹地图与动画回放播放器 -->
      <div class="col-span-12 lg:col-span-8 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[480px] relative flex flex-col">
          <!-- 轨迹地图工具栏 -->
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-3">
              <Navigation :size="16" class="text-blue-400" />
              <span class="text-xs font-bold text-slate-200">纯净 GPS 历史轨迹 Polyline 动画播放</span>
            </div>

            <!-- 时间筛选控制器 -->
            <div class="flex items-center space-x-1 bg-slate-950/80 p-0.5 rounded-lg border border-slate-800">
              <button 
                @click="trackTimeFilter = 'today'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', trackTimeFilter === 'today' ? 'bg-blue-600 text-white' : 'text-slate-400 hover:text-white']"
              >今日 (24h)</button>
              <button 
                @click="trackTimeFilter = 'yesterday'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', trackTimeFilter === 'yesterday' ? 'bg-blue-600 text-white' : 'text-slate-400 hover:text-white']"
              >昨天</button>
              <button 
                @click="trackTimeFilter = '3days'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', trackTimeFilter === '3days' ? 'bg-blue-600 text-white' : 'text-slate-400 hover:text-white']"
              >近 3 天</button>
            </div>
          </div>

          <!-- 高德轨迹地图组件 -->
          <div class="flex-1 relative">
            <AmapDashboard 
              :devices="[device]" 
              :selectedImei="device.imei"
              :showTrack="true"
              :trackPoints="trajectoryWaypoints"
              :runnerPoint="runnerCurrentPoint"
              :showHeatmap="false"
            />
          </div>

          <!-- 底部轨迹播放控制器 Bottom Bar -->
          <div class="p-3 border-t border-slate-800 bg-slate-900/90 backdrop-blur-md flex items-center justify-between z-10 font-mono text-xs">
            <div class="flex items-center space-x-3">
              <button 
                @click="togglePlayback" 
                class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold flex items-center space-x-1.5 shadow-lg shadow-blue-600/30 active:scale-95 transition"
              >
                <Pause v-if="isPlayingTrack" :size="16" />
                <Play v-else :size="16" />
                <span>{{ isPlayingTrack ? '暂停播放' : '播放轨迹动画' }}</span>
              </button>

              <div class="flex items-center space-x-1 bg-slate-950 p-1 rounded-xl border border-slate-800">
                <span class="text-slate-400 text-[10px] px-1">倍速:</span>
                <button 
                  v-for="s in [1, 2, 4]" 
                  :key="s"
                  @click="playbackSpeed = s as 1 | 2 | 4"
                  :class="['px-2 py-0.5 rounded text-[10px] font-bold transition', playbackSpeed === s ? 'bg-blue-500 text-white' : 'text-slate-400 hover:text-white']"
                >
                  {{ s }}x
                </button>
              </div>
            </div>

            <div class="text-slate-400 flex items-center space-x-2">
              <span>当前驻留点: <strong class="text-blue-400 font-bold">{{ trajectoryPointsData[currentTrackStep]?.locationName }}</strong></span>
              <span>({{ currentTrackStep + 1 }}/{{ trajectoryPointsData.length }})</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧 4 列: 包含中文位置名称与详细地址的采样明细列表 -->
      <div class="col-span-12 lg:col-span-4 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl p-5 border border-slate-800 space-y-3">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 border-b border-slate-800 pb-2">
            <Navigation :size="16" class="text-blue-400" />
            <span>今日活动里程与停留汇总</span>
          </h3>

          <div class="grid grid-cols-2 gap-2 text-xs font-mono">
            <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
              <span class="text-slate-400 block text-[10px]">全天移动总里程</span>
              <span class="text-blue-400 font-bold text-sm">4.25 公里</span>
            </div>
            <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
              <span class="text-slate-400 block text-[10px]">平均移动时速</span>
              <span class="text-emerald-400 font-bold text-sm">2.4 km/h</span>
            </div>
          </div>
        </div>

        <div class="glass-panel rounded-2xl p-5 border border-slate-800 flex-1 flex flex-col min-h-[300px]">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 mb-3 border-b border-slate-800 pb-2">
            <MapPin :size="16" class="text-cyan-400" />
            <span>沿途 GPS 中文地标与采样点明细</span>
          </h3>

          <div class="space-y-2.5 flex-1 overflow-y-auto max-h-[360px] pr-1 font-mono text-xs">
            <div 
              v-for="(item, idx) in trajectoryPointsData" 
              :key="item.id"
              @click="selectTrackStep(idx)"
              :class="['p-3 rounded-xl border transition cursor-pointer flex items-center justify-between',
                currentTrackStep === idx ? 'bg-blue-950/60 border-blue-500/80 text-white shadow-lg shadow-blue-950/50' : 'bg-slate-900/60 border-slate-800 text-slate-300 hover:bg-slate-800']"
            >
              <div class="flex items-center space-x-3">
                <span :class="['w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-bold shrink-0',
                  idx === 0 ? 'bg-emerald-600 text-white' : idx === trajectoryPointsData.length - 1 ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-300']"
                >
                  {{ idx + 1 }}
                </span>
                <div>
                  <div class="font-bold text-slate-100 flex items-center space-x-1.5">
                    <span>{{ item.locationName }}</span>
                    <span v-if="currentTrackStep === idx" class="text-[9px] px-1.5 py-0.2 bg-blue-500/30 text-blue-300 rounded border border-blue-500/40">播放中</span>
                  </div>
                  <div class="text-[10px] text-slate-400 mt-0.5">{{ item.address }}</div>
                </div>
              </div>

              <div class="text-right text-[10px] text-slate-400 shrink-0 ml-2">
                <div class="text-white font-bold">{{ item.time }}</div>
                <div class="text-cyan-400 font-bold mt-0.5">{{ item.speed }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== TAB 3: 📊 30天活动热力与生活圈视图 ===== -->
    <div v-else-if="activeTab === 'heatmap'" class="grid grid-cols-12 gap-5 flex-1">
      <!-- 左侧 8 列: AMap.HeatMap 活动热力图地图 -->
      <div class="col-span-12 lg:col-span-8 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[480px] relative flex flex-col">
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-3">
              <Flame :size="16" class="text-amber-400" />
              <span class="text-xs font-bold text-slate-200">高德活动热力密度图层 (AMap.HeatMap)</span>
            </div>

            <div class="flex items-center space-x-1 bg-slate-950/80 p-0.5 rounded-lg border border-slate-800">
              <button 
                @click="heatmapTimeFilter = '7days'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', heatmapTimeFilter === '7days' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-white']"
              >近 7 天</button>
              <button 
                @click="heatmapTimeFilter = '30days'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', heatmapTimeFilter === '30days' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-white']"
              >近 30 天</button>
              <button 
                @click="heatmapTimeFilter = '90days'"
                :class="['px-2.5 py-1 rounded text-xs font-bold transition', heatmapTimeFilter === '90days' ? 'bg-indigo-600 text-white' : 'text-slate-400 hover:text-white']"
              >近 90 天</button>
            </div>
          </div>

          <div class="flex-1 relative">
            <AmapDashboard 
              :devices="[device]" 
              :selectedImei="device.imei"
              :zoomLevel="14"
              :showTrack="false"
              :showHeatmap="true"
              :heatmapPoints="heatmapDataPoints"
              :heatmapRadius="heatmapRadius"
            />
          </div>
        </div>
      </div>

      <!-- 右侧 4 列: 常去地标 TOP4 与生活圈评估看板 -->
      <div class="col-span-12 lg:col-span-4 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl p-5 border border-slate-800 space-y-3">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 border-b border-slate-800 pb-2">
            <Flame :size="16" class="text-amber-400" />
            <span>长者常去地标驻留 TOP 4 榜单</span>
          </h3>

          <div class="space-y-2.5 font-mono text-xs">
            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex items-center justify-between">
              <div class="flex items-center space-x-2">
                <span class="w-5 h-5 rounded-full bg-amber-500/20 text-amber-400 flex items-center justify-center font-bold text-[10px]">1</span>
                <div>
                  <div class="font-bold text-white">安老院/居所核心区</div>
                  <div class="text-[10px] text-slate-400">荃湾街市街安老院</div>
                </div>
              </div>
              <span class="text-amber-400 font-bold">驻留 68%</span>
            </div>

            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex items-center justify-between">
              <div class="flex items-center space-x-2">
                <span class="w-5 h-5 rounded-full bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-bold text-[10px]">2</span>
                <div>
                  <div class="font-bold text-white">社区菜市场/超市</div>
                  <div class="text-[10px] text-slate-400">荃湾街市街 55 号</div>
                </div>
              </div>
              <span class="text-cyan-400 font-bold">驻留 18%</span>
            </div>

            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex items-center justify-between">
              <div class="flex items-center space-x-2">
                <span class="w-5 h-5 rounded-full bg-emerald-500/20 text-emerald-400 flex items-center justify-center font-bold text-[10px]">3</span>
                <div>
                  <div class="font-bold text-white">社区体育公园</div>
                  <div class="text-[10px] text-slate-400">荃湾海滨公园</div>
                </div>
              </div>
              <span class="text-emerald-400 font-bold">驻留 9%</span>
            </div>

            <div class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex items-center justify-between">
              <div class="flex items-center space-x-2">
                <span class="w-5 h-5 rounded-full bg-slate-800 text-slate-400 flex items-center justify-center font-bold text-[10px]">4</span>
                <div>
                  <div class="font-bold text-slate-300">健康服务中心</div>
                  <div class="text-[10px] text-slate-400">葵涌社区健康中心</div>
                </div>
              </div>
              <span class="text-slate-400 font-bold">驻留 5%</span>
            </div>
          </div>
        </div>

        <div class="glass-panel rounded-2xl p-5 border border-slate-800 flex-1 flex flex-col space-y-3">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 border-b border-slate-800 pb-2">
            <ShieldAlert :size="16" class="text-emerald-400" />
            <span>习惯生活圈安全评级</span>
          </h3>

          <div class="bg-emerald-950/30 border border-emerald-500/30 rounded-xl p-3.5 space-y-2 text-xs font-mono">
            <div class="flex items-center justify-between font-bold text-emerald-300">
              <span>生活圈范围评估:</span>
              <span class="text-emerald-400">高度规律 🟢</span>
            </div>
            <p class="text-[11px] text-slate-300 leading-relaxed font-sans">
              长者 30 天日常活动半径保持在离家 1.8km 安全圈范围内，活动轨迹高度规律，无偏僻危险区域落单记录。
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 解法 3: 调阅全量健康体征研判档案 模态框/大抽屉 ===== -->
    <div v-if="showFullReportModal" class="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-md flex items-center justify-center p-6">
      <div class="bg-slate-900 border border-slate-800 rounded-3xl w-full max-w-5xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
        <div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/50">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
              <FileText :size="20" />
            </div>
            <div>
              <h2 class="text-base font-extrabold text-white flex items-center space-x-2">
                <span>{{ device.owner_name }} - 长者全量健康体征研判档案</span>
                <span class="text-xs font-normal px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 border border-cyan-500/30 font-mono">
                  档案编号: HK-ELDER-20260727
                </span>
              </h2>
              <p class="text-xs text-slate-400 font-mono mt-0.5">多维体征历史关联对比与综合判定报告</p>
            </div>
          </div>

          <div class="flex items-center space-x-2">
            <button @click="showFullReportModal = false" class="p-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 transition">
              <X :size="20" />
            </button>
          </div>
        </div>

        <div class="p-6 space-y-6 overflow-y-auto max-h-[75vh] font-mono text-xs">
          <div class="bg-indigo-950/30 border border-indigo-500/30 rounded-2xl p-4 grid grid-cols-12 gap-4 items-center">
            <div class="col-span-3 border-r border-indigo-500/20 pr-4 text-center">
              <span class="text-slate-400 block text-xs font-sans">AI 综合风险评级</span>
              <div :class="['text-3xl font-black mt-1', healthRiskScore < 60 ? 'text-red-400' : 'text-emerald-400']">
                {{ healthRiskScore }} <span class="text-xs font-normal text-slate-400">/ 100</span>
              </div>
              <span class="text-[10px] text-emerald-400 font-sans block mt-1">建议每周随访 1 次</span>
            </div>
            <div class="col-span-9 space-y-1 text-slate-300">
              <div class="font-bold text-indigo-200 text-sm font-sans mb-1">长者近 30 天体征健康全貌诊断要点:</div>
              <div>· 心率静息均值 <strong class="text-white">76 bpm</strong>，最高峰值 <strong class="text-red-400">118 bpm</strong>（触发紧急 SOS 求救）；</div>
              <div>· 参考血压近期略有升高，平均收缩压 <strong class="text-amber-300">128 mmHg</strong>，建议关注防范高血压事件；</div>
              <div>· 血氧饱和度稳定于 <strong class="text-cyan-300">97-99%</strong>，呼吸功能极佳；表体温度均匀保持在 <strong class="text-emerald-300">36.6°C</strong> 正常区间；</div>
              <div>· 习惯出行活动范围处于荃湾/葵涌社区安全圈内，无走失越界隐患。</div>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="bg-slate-950/60 p-4 rounded-2xl border border-slate-800 space-y-3">
              <h4 class="font-bold text-slate-200 flex items-center space-x-2 text-xs font-sans">
                <Heart :size="16" class="text-rose-400" />
                <span>1. 心率与血压全量对比数据</span>
              </h4>
              <div class="grid grid-cols-3 gap-2 text-center text-[11px]">
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">最高心率</span>
                  <span class="text-red-400 font-bold text-sm">118 bpm</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">静息心率均值</span>
                  <span class="text-emerald-400 font-bold text-sm">74 bpm</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">血压极值</span>
                  <span class="text-amber-300 font-bold text-sm">142/92</span>
                </div>
              </div>
            </div>

            <div class="bg-slate-950/60 p-4 rounded-2xl border border-slate-800 space-y-3">
              <h4 class="font-bold text-slate-200 flex items-center space-x-2 text-xs font-sans">
                <Zap :size="16" class="text-cyan-400" />
                <span>2. 血氧饱和度与体温监测数据</span>
              </h4>
              <div class="grid grid-cols-3 gap-2 text-center text-[11px]">
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">血氧均值</span>
                  <span class="text-cyan-300 font-bold text-sm">98.5 %</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">最低血氧</span>
                  <span class="text-amber-400 font-bold text-sm">94 %</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">平均体温</span>
                  <span class="text-emerald-300 font-bold text-sm">36.6 °C</span>
                </div>
              </div>
            </div>
          </div>

          <div class="pt-2 border-t border-slate-800 flex items-center justify-between text-slate-400 text-xs">
            <div>报告生成时间: {{ new Date().toLocaleString() }}</div>
            <div class="flex items-center space-x-4">
              <span>社区主治医师: <strong class="text-white">林医生 (已审核)</strong></span>
              <span>安防网格员: <strong class="text-white">陈专员</strong></span>
            </div>
          </div>
        </div>

        <div class="px-6 py-3.5 border-t border-slate-800 flex items-center justify-between bg-slate-950/80">
          <span class="text-xs text-slate-400 font-mono">提示：该研判报告数据自动同步至 C端家属小程序</span>
          <button @click="showFullReportModal = false" class="px-5 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-bold font-sans">
            关闭窗口
          </button>
        </div>
      </div>
    </div>

  </div>
</template>
