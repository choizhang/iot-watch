<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDashboardStore } from '../store/dashboard'
import AmapDashboard from '../components/AmapDashboard.vue'
import { 
  ArrowLeft, 
  Send, 
  Heart, 
  Activity, 
  Battery, 
  MapPin, 
  ShieldAlert, 
  User, 
  Zap,
  Flame,
  Navigation,
  Shield,
  FileText,
  AlertTriangle,
  X,
  TrendingUp,
  Pause,
  Play
} from 'lucide-vue-next'

import { getGoogleMapsInstance } from '../utils/amap'
import { 
  fetchDeviceStatus, 
  fetchGeofences, 
  fetchAlarms, 
  fetchVitalsHistory, 
  fetchTrajectory, 
  fetchHeatmap, 
  type GeofenceItem, 
  type TrajectoryPoint, 
  type AlarmItem, 
  type LandmarkItem 
} from '../api/deviceApi'
import { 
  getMockDeviceStatus, 
  getMockGeofences, 
  getMockAlarms, 
  getMockVitalsHistory, 
  getMockTrajectory, 
  getMockHeatmap 
} from '../mock/deviceMock'

const route = useRoute()
const router = useRouter()
const store = useDashboardStore()

const imei = computed(() => (route.params.imei as string) || store.selectedImei)

// 3 大核心解耦 Tab 视图切换
const activeTab = ref<'realtime' | 'trajectory' | 'heatmap'>('realtime')

// 多维体征趋势多切一卡片 (心率/血压/血氧/HRV/计步)
const selectedMetric = ref<'hr' | 'bp' | 'spo2' | 'hrv' | 'steps'>('hr')

// 全量健康研判报告大抽屉/弹窗状态
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

// ── 模块数据状态声明 ──────────────────────────────────────────────────────────
const deviceData = ref<any>(null)
const geofences = ref<GeofenceItem[]>([])
const alarmList = ref<AlarmItem[]>([])
const vitalsData = ref<{
  hr: number[]
  bpSys: number[]
  bpDia: number[]
  spo2: number[]
  hrv: number[]
  steps: number[]
  hoursLabel: string[]
  stepsHoursLabel: string[]
}>({
  hr: [],
  bpSys: [],
  bpDia: [],
  spo2: [],
  hrv: [],
  steps: [],
  hoursLabel: [],
  stepsHoursLabel: []
})
const trajectoryPointsData = ref<TrajectoryPoint[]>([])
const totalDistance = ref<number>(0)
const avgSpeed = ref<number>(0)
const heatmapDataPoints = ref<{ lng: number; lat: number; count: number }[]>([])
const topLandmarks = ref<LandmarkItem[]>([])

// ── 统一加载与模式切换服务 ──────────────────────────────────────────────────
const loadDetailData = async () => {
  if (!imei.value) return

  if (store.mockMode) {
    // 【全场景 Mock 模式】：从独立 deviceMock 服务抽取富演示数据
    deviceData.value = getMockDeviceStatus(imei.value)
    geofences.value = getMockGeofences(imei.value)
    alarmList.value = getMockAlarms(imei.value)

    const mockVitals = getMockVitalsHistory(imei.value)
    vitalsData.value = {
      hr: mockVitals.hr,
      bpSys: mockVitals.bp_sys,
      bpDia: mockVitals.bp_dia,
      spo2: mockVitals.spo2,
      hrv: mockVitals.hrv || mockVitals.temp || [],
      steps: mockVitals.steps,
      hoursLabel: mockVitals.hours_label || [],
      stepsHoursLabel: mockVitals.steps_hours_label || mockVitals.hours_label || []
    }

    const mockTrack = getMockTrajectory(imei.value, trackTimeFilter.value)
    trajectoryPointsData.value = mockTrack.points
    totalDistance.value = mockTrack.total_distance
    avgSpeed.value = mockTrack.avg_speed

    const mockHeat = getMockHeatmap(imei.value, heatmapTimeFilter.value)
    heatmapDataPoints.value = mockHeat.points
    topLandmarks.value = mockHeat.landmarks
  } else {
    // 【真实 TCP 直连模式】：全量调用 deviceApi 请求服务端 API
    const statusRes = await fetchDeviceStatus(imei.value)
    deviceData.value = statusRes

    geofences.value = await fetchGeofences(imei.value)
    alarmList.value = await fetchAlarms(imei.value)

    const vitalsRes = await fetchVitalsHistory(imei.value)
    vitalsData.value = {
      hr: vitalsRes.hr || [],
      bpSys: vitalsRes.bp_sys || [],
      bpDia: vitalsRes.bp_dia || [],
      spo2: vitalsRes.spo2 || [],
      hrv: vitalsRes.hrv || vitalsRes.temp || [],
      steps: vitalsRes.steps || [],
      hoursLabel: vitalsRes.hours_label || [],
      stepsHoursLabel: vitalsRes.steps_hours_label || vitalsRes.hours_label || []
    }

    const trackRes = await fetchTrajectory(imei.value, trackTimeFilter.value)
    trajectoryPointsData.value = trackRes.points || []
    totalDistance.value = trackRes.total_distance || 0
    avgSpeed.value = trackRes.avg_speed || 0

    const heatRes = await fetchHeatmap(imei.value, heatmapTimeFilter.value)
    heatmapDataPoints.value = heatRes.points || []
    topLandmarks.value = heatRes.landmarks || []
  }
}

// 设备基础信息 Computed
const device = computed(() => {
  if (deviceData.value) return deviceData.value
  const found = store.devices.find(d => d.imei === imei.value)
  if (found) return found
  return {
    imei: imei.value || '',
    owner_name: imei.value ? `未注册设备 #${imei.value.slice(-4)}` : '未知设备',
    owner_phone: '--',
    status: 'offline' as const,
    battery: 0,
    last_heart_rate: 0,
    last_latitude: 0,
    last_longitude: 0,
    address: '暂无数据',
    updated_at: 0,
    fix_mode: 'GPS' as const,
    satellites: 0,
    accuracy: 0,
    rssi: 0,
    bp: '--',
    spo2: 0,
    temperature: 0,
    steps: 0
  }
})

// 智能健康风险评分计算
const healthRiskScore = computed(() => {
  if (device.value.status === 'sos_alert' || alarmList.value.some(a => a.status === 'UNHANDLED')) return 45
  if (device.value.battery < 20 || (device.value.last_heart_rate && device.value.last_heart_rate > 100)) return 68
  if (!device.value.last_heart_rate && vitalsData.value.hr.length === 0) return 90
  return 88
})

// 24小时体征趋势时间点线图计算
const chartMetricInfo = computed(() => {
  let rawValues: number[] = []
  let unit = ''
  let label = ''
  let strokeColor = '#f43f5e'
  let minVal = 40
  let maxVal = 140
  let lowerThreshold = 60
  let upperThreshold = 100

  if (selectedMetric.value === 'hr') {
    rawValues = vitalsData.value.hr || []
    unit = 'bpm'
    label = '心率'
    strokeColor = '#f43f5e'
    minVal = 40
    maxVal = 140
    lowerThreshold = 60
    upperThreshold = 100
  } else if (selectedMetric.value === 'bp') {
    rawValues = vitalsData.value.bpSys || []
    unit = 'mmHg'
    label = '收缩压'
    strokeColor = '#3b82f6'
    minVal = 60
    maxVal = 160
    lowerThreshold = 90
    upperThreshold = 140
  } else if (selectedMetric.value === 'spo2') {
    rawValues = vitalsData.value.spo2 || []
    unit = '%'
    label = '血氧'
    strokeColor = '#06b6d4'
    minVal = 85
    maxVal = 100
    lowerThreshold = 95
    upperThreshold = 99
  } else if (selectedMetric.value === 'hrv') {
    rawValues = vitalsData.value.hrv || []
    unit = 'ms'
    label = '心率变异性(HRV)'
    strokeColor = '#10b981'
    minVal = 0
    maxVal = 120
    lowerThreshold = 30
    upperThreshold = 100
  } else if (selectedMetric.value === 'steps') {
    rawValues = vitalsData.value.steps || []
    unit = '步'
    label = '计步'
    strokeColor = '#6366f1'
    const maxInArr = Math.max(...(rawValues.length ? rawValues : [0]), 1000)
    minVal = 0
    maxVal = maxInArr * 1.15
    lowerThreshold = 3000
    upperThreshold = 8000
  }

  const validRawValues = rawValues.filter(v => v > 0)
  if (!rawValues || rawValues.length === 0 || validRawValues.length === 0) {
    return { 
      points: [], 
      validPoints: [], 
      polylineStr: '', 
      areaStr: '', 
      strokeColor, 
      unit, 
      label, 
      rawValues: [],
      padLeft: 45,
      padRight: 20,
      lowerThreshold,
      upperThreshold,
      upperY: 0,
      lowerY: 0,
      yTicks: []
    }
  }

  const width = 560
  const height = 110
  const padLeft = 45
  const padRight = 20
  const padY = 15

  const computedMin = Math.min(...validRawValues)
  const computedMax = Math.max(...validRawValues)
  const actualMin = Math.min(minVal, computedMin)
  const actualMax = Math.max(maxVal, computedMax)
  const range = actualMax - actualMin || 1

  const getSecondsInDay = (label: string): number => {
    if (!label) return 0
    const parts = label.split(':')
    if (parts.length >= 2) {
      const h = parseInt(parts[0], 10) || 0
      const m = parseInt(parts[1], 10) || 0
      const s = parseInt(parts[2] || '0', 10) || 0
      return h * 3600 + m * 60 + s
    }
    return 0
  }

  const points = rawValues.map((v, i) => {
    let x = padLeft + (i / Math.max(1, rawValues.length - 1)) * (width - padLeft - padRight)
    let timeLabel = vitalsData.value.hoursLabel?.[i] || `${i}:00`

    if (selectedMetric.value === 'steps') {
      timeLabel = vitalsData.value.stepsHoursLabel?.[i] || vitalsData.value.hoursLabel?.[i] || `${i}:00`
      const secs = getSecondsInDay(timeLabel)
      const ratio = Math.max(0, Math.min(1, secs / 86400))
      x = padLeft + ratio * (width - padLeft - padRight)
    }

    const y = v > 0 ? height - padY - ((v - actualMin) / range) * (height - 2 * padY) : height - padY
    return { x, y, value: v, timeLabel, hasData: v > 0 }
  })

  const validPoints = points.filter(p => p.hasData).sort((a, b) => a.x - b.x)

  let polylineStr = ''
  let areaStr = ''
  const botY = (height - padY).toFixed(1)

  if (validPoints.length === 1) {
    const p = validPoints[0]
    const x1 = (p.x - 0.5).toFixed(1)
    const x2 = (p.x + 0.5).toFixed(1)
    const yStr = p.y.toFixed(1)
    polylineStr = `${x1},${yStr} ${x2},${yStr}`
    areaStr = `${x1},${botY} ${x1},${yStr} ${x2},${yStr} ${x2},${botY}`
  } else if (validPoints.length > 1) {
    polylineStr = validPoints.map(p => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
    const firstX = validPoints[0].x.toFixed(1)
    const lastX = validPoints[validPoints.length - 1].x.toFixed(1)
    areaStr = `${firstX},${botY} ${polylineStr} ${lastX},${botY}`
  }

  const upperY = height - padY - ((upperThreshold - actualMin) / range) * (height - 2 * padY)
  const lowerY = height - padY - ((lowerThreshold - actualMin) / range) * (height - 2 * padY)

  // 纵坐标刻度数值生成 (4个均匀分布的刻度值)
  const yTicks = [1, 0.66, 0.33, 0].map(ratio => {
    const val = actualMin + ratio * range
    const y = height - padY - ratio * (height - 2 * padY)
    return {
      val: Math.round(val),
      y
    }
  })

  return { 
    points, 
    validPoints, 
    polylineStr, 
    areaStr, 
    strokeColor, 
    unit, 
    label, 
    rawValues,
    padLeft,
    padRight,
    lowerThreshold,
    upperThreshold,
    upperY,
    lowerY,
    yTicks
  }
})

// 历史轨迹 Polyline 点集
const trajectoryWaypoints = computed(() => {
  return trajectoryPointsData.value.map(p => [p.lng, p.lat] as [number, number])
})

// 当前回放动画位置点
const runnerCurrentPoint = computed(() => {
  return trajectoryWaypoints.value[currentTrackStep.value] || trajectoryWaypoints.value[0] || [device.value.last_longitude || 114.1, device.value.last_latitude || 22.3]
})

// 轨迹播放控制逻辑
const togglePlayback = () => {
  if (isPlayingTrack.value) {
    pausePlayback()
  } else {
    startPlayback()
  }
}

const startPlayback = () => {
  if (trajectoryWaypoints.value.length === 0) return
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
  if (!lat || !lng) {
    resolvedAddressText.value = device.value?.address || '暂无定位数据'
    return
  }
  try {
    const googleMaps = await getGoogleMapsInstance()
    const geocoder = new googleMaps.Geocoder()
    geocoder.geocode({ location: { lat, lng } }, (results: any, status: string) => {
      if (status === 'OK' && results && results[0]) {
        resolvedAddressText.value = results[0].formatted_address
      } else {
        resolvedAddressText.value = device.value?.address || '暂无定位数据'
      }
    })
  } catch (err) {
    resolvedAddressText.value = device.value?.address || '暂无定位数据'
  }
}

const getTimeAgoText = (ts: number) => {
  if (!ts) return '未知'
  const diffSec = Math.floor(Date.now() / 1000) - ts
  if (diffSec < 60) return '刚刚 (实时直连)'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} 分钟前`
  return `${Math.floor(diffSec / 3600)} 小时前`
}

const formatUpdateTime = (timestamp?: number) => {
  if (!timestamp) return '刚刚'
  const date = new Date(timestamp * 1000)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

const isInvalidHealthVal = (val: any) => {
  return !val || val === 0 || val === 'xxx' || val === '--' || String(val) === '0'
}

watch(
  () => [device.value?.last_latitude, device.value?.last_longitude, device.value?.address],
  ([lat, lng]) => {
    if (lat && lng) {
      resolveAddress(Number(lat), Number(lng))
    } else if (device.value?.address) {
      resolvedAddressText.value = device.value.address
    }
  },
  { immediate: true }
)

watch(
  [() => imei.value, () => store.mockMode, () => trackTimeFilter.value, () => heatmapTimeFilter.value],
  () => {
    loadDetailData()
  },
  { immediate: true }
)

onMounted(() => {
  store.fetchAllDevices()
  loadDetailData()
})

onUnmounted(() => {
  if (playbackTimer) clearInterval(playbackTimer)
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
          {{ device.owner_name ? device.owner_name.slice(0, 1) : '长' }}
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
            <span>监护电话: {{ device.owner_phone || '--' }}</span>
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

      <!-- 调阅全量报告按钮 & 指令发送 -->
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

    <!-- ===== 智能健康风险诊断与全貌总结看板 (AI Health Summary) ===== -->
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
              {{ alarmList.length > 0 ? `记录 ${alarmList.length} 起告警事件 (${alarmList[0].alert_type})` : '近 24 小时无告警记录' }}
            </div>
          </div>
        </div>

        <div class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-start space-x-2">
          <Activity :size="16" class="text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <div class="font-bold text-amber-300">血压与 HRV 预警</div>
            <div class="text-[11px] text-slate-400 mt-0.5">
              {{ isInvalidHealthVal(device.bp) && isInvalidHealthVal(device.hrv || device.temperature) ? '暂无血压与 HRV 上报数据' : `血压 ${device.bp || '--'}, 心率变异性 ${device.hrv || device.temperature || '--'} ms` }}
            </div>
          </div>
        </div>

        <div class="bg-slate-900/80 p-2.5 rounded-xl border border-slate-800/80 flex items-start space-x-2">
          <ShieldAlert :size="16" class="text-emerald-400 flex-shrink-0 mt-0.5" />
          <div>
            <div class="font-bold text-emerald-300">围栏与血氧范畴</div>
            <div class="text-[11px] text-slate-400 mt-0.5">
              {{ geofences.length > 0 ? `已激活 ${geofences.length} 个防护围栏` : '暂无启用中的围栏防护规则' }}
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
          {{ isInvalidHealthVal(device.last_heart_rate) ? '--' : device.last_heart_rate }} <span class="text-xs text-slate-400 font-normal">bpm</span>
        </div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span :class="['font-medium', isInvalidHealthVal(device.last_heart_rate) ? 'text-slate-500' : Number(device.last_heart_rate) > 100 ? 'text-red-400 font-bold' : 'text-emerald-400']">
            {{ isInvalidHealthVal(device.last_heart_rate) ? '暂无数据' : Number(device.last_heart_rate) > 100 ? '心率过高告警' : '正常静息范围' }}
          </span>
          <span v-if="!isInvalidHealthVal(device.last_heart_rate) && device.hr_updated_at" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.hr_updated_at) }}</span>
        </div>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">参考血压</span>
          <Activity :size="16" class="text-blue-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ isInvalidHealthVal(device.bp) ? '--' : device.bp }} <span class="text-xs text-slate-400 font-normal">mmHg</span></div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span class="font-medium" :class="isInvalidHealthVal(device.bp) ? 'text-slate-500' : 'text-blue-400'">
            {{ isInvalidHealthVal(device.bp) ? '暂无数据' : '理想血压范围' }}
          </span>
          <span v-if="!isInvalidHealthVal(device.bp) && device.bp_updated_at" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.bp_updated_at) }}</span>
        </div>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">血氧饱和度</span>
          <Zap :size="16" class="text-cyan-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ isInvalidHealthVal(device.spo2) ? '--' : device.spo2 }} <span class="text-xs text-slate-400 font-normal">%</span></div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span class="font-medium" :class="isInvalidHealthVal(device.spo2) ? 'text-slate-500' : 'text-cyan-400'">
            {{ isInvalidHealthVal(device.spo2) ? '暂无数据' : '含氧量良好' }}
          </span>
          <span v-if="!isInvalidHealthVal(device.spo2) && device.spo2_updated_at" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.spo2_updated_at) }}</span>
        </div>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">心率变异性 (HRV)</span>
          <Activity :size="16" class="text-emerald-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ isInvalidHealthVal(device.hrv || device.temperature) ? '--' : (device.hrv || device.temperature) }} <span class="text-xs text-slate-400 font-normal">ms</span></div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span class="font-medium" :class="isInvalidHealthVal(device.hrv || device.temperature) ? 'text-slate-500' : 'text-emerald-400'">
            {{ isInvalidHealthVal(device.hrv || device.temperature) ? '暂无数据' : '自主神经调节良好' }}
          </span>
          <span v-if="!isInvalidHealthVal(device.hrv || device.temperature) && (device.hrv_updated_at || device.temp_updated_at)" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.hrv_updated_at || device.temp_updated_at) }}</span>
        </div>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">设备剩余电量</span>
          <Battery :size="16" class="text-amber-400" />
        </div>
        <div class="text-2xl font-black text-amber-300 mt-1 font-mono">{{ device.battery ?? '--' }} <span class="text-xs text-slate-400 font-normal">%</span></div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span class="font-medium text-amber-400">电量正常</span>
          <span v-if="device.battery !== null && device.battery !== undefined && device.battery !== ''" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.battery_updated_at || device.updated_at) }}</span>
        </div>
      </div>

      <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
        <div class="flex items-center justify-between text-slate-400">
          <span class="text-xs font-medium">运动步数</span>
          <User :size="16" class="text-indigo-400" />
        </div>
        <div class="text-2xl font-black text-white mt-1 font-mono">{{ isInvalidHealthVal(device.steps) ? '--' : device.steps }} <span class="text-xs text-slate-400 font-normal">步</span></div>
        <div class="flex items-center justify-between mt-1 text-[10px]">
          <span class="font-medium" :class="isInvalidHealthVal(device.steps) ? 'text-slate-500' : 'text-indigo-400'">
            {{ isInvalidHealthVal(device.steps) ? '暂无数据' : '健康活力分' }}
          </span>
          <span v-if="!isInvalidHealthVal(device.steps) && device.steps_updated_at" class="text-slate-400 font-medium">获取时间: {{ formatUpdateTime(device.steps_updated_at) }}</span>
        </div>
      </div>
    </div>

    <!-- ===== TAB 1: 📍 实时精细定位视图 ===== -->
    <div v-if="activeTab === 'realtime'" class="grid grid-cols-12 gap-5 flex-1">
      
      <!-- 左侧 7 列: 实时高精地图与定位硬件遥测 -->
      <div class="col-span-12 lg:col-span-7 flex flex-col space-y-4">
        <!-- 实时地图 -->
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[440px] relative flex flex-col">
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-2">
              <MapPin :size="16" class="text-cyan-400" />
              <span class="text-xs font-bold text-slate-200">实时楼栋/门牌级精细定位 (Zoom 15)</span>
            </div>
            <div class="flex items-center space-x-2 font-mono">
              <span :class="['px-2 py-0.5 rounded text-[10px] font-bold border',
                (device.location_type || device.last_location_type) === 'GPS' ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' :
                (device.location_type || device.last_location_type) === 'WIFI' ? 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30' :
                'bg-purple-500/20 text-purple-300 border-purple-500/30']"
              >
                {{ (device.location_type || device.last_location_type || 'GPS') }} 模式定位
              </span>
              <span class="px-2 py-0.5 rounded bg-slate-800/90 text-cyan-300 border border-slate-700 text-[10px] font-bold">
                获取时间: {{ formatUpdateTime(device.updated_at) }} ({{ getTimeAgoText(device.updated_at) }})
              </span>
            </div>
          </div>

          <div class="flex-1 relative">
            <AmapDashboard 
              :devices="[device]" 
              :selectedImei="device.imei"
              :zoomLevel="15"
              :showTrack="false"
              :showHeatmap="false"
              :geofences="geofences"
            />
          </div>
        </div>

        <!-- 详细地址解析与硬件遥测组合栏 -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 grid grid-cols-12 gap-4">
          <div class="col-span-7 space-y-1.5 border-r border-slate-800 pr-4">
            <div class="text-[11px] text-slate-400 font-mono">反向 Geocoder 解析中文地址:</div>
            <div class="text-xs text-cyan-300 font-bold leading-relaxed">{{ resolvedAddressText }}</div>
            <div class="text-[11px] text-slate-400 pt-1">责任网格员: <span class="text-slate-200 font-medium">陈专员 (0755-88889999)</span></div>
          </div>
          <div class="col-span-5 space-y-2 text-[11px] font-mono">
            <div class="grid grid-cols-3 gap-1.5">
              <div class="bg-slate-900/80 p-1.5 rounded-lg border border-slate-800">
                <span class="text-slate-400 block text-[9px]">定位参考</span>
                <span class="font-bold text-[11px]" :class="
                  (device.last_location_type || device.location_type) === 'GPS' ? 'text-emerald-400' :
                  (device.last_location_type || device.location_type) === 'WIFI' ? 'text-cyan-400' : 'text-purple-400'
                ">
                  {{ (device.last_location_type || device.location_type || 'GPS') }}
                </span>
              </div>
              <div class="bg-slate-900/80 p-1.5 rounded-lg border border-slate-800">
                <span class="text-slate-400 block text-[9px]">卫星数</span>
                <span class="text-white font-bold text-[11px]">{{ device.satellites ?? 0 }} 颗</span>
              </div>
              <div class="bg-slate-900/80 p-1.5 rounded-lg border border-slate-800">
                <span class="text-slate-400 block text-[9px]">定位误差</span>
                <span class="text-emerald-400 font-bold text-[11px]">±{{ device.accuracy ?? 18 }}m</span>
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

          <!-- 空状态逻辑：无围栏数据时展示 -->
          <div v-if="geofences.length === 0" class="text-xs text-slate-500 py-6 text-center font-mono">
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
                  <span>防护中心: {{ device?.address || '社区安全圈' }}</span>
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

        <!-- 2. 24h 体征趋势多维指标看板 -->
        <div class="glass-panel rounded-2xl p-4 border border-slate-800 flex flex-col justify-between">
          <div class="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
            <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2">
              <TrendingUp :size="16" class="text-cyan-400" />
              <span>近 24 小时体征趋势</span>
            </h3>

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
                @click="selectedMetric = 'hrv'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'hrv' ? 'bg-emerald-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🫀 HRV变异性</button>
              <button 
                @click="selectedMetric = 'steps'" 
                :class="['px-2 py-0.5 rounded-lg transition', selectedMetric === 'steps' ? 'bg-indigo-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >🚶 计步</button>
            </div>
          </div>

          <!-- 动态渲染 24 小时体征趋势时间点线图 (Line & Point Chart) -->
          <div class="h-36 bg-slate-900/80 rounded-xl border border-slate-800/80 p-3 relative flex flex-col justify-between overflow-hidden">
            <div v-if="chartMetricInfo.rawValues.length === 0" class="w-full h-full flex items-center justify-center text-slate-500 text-xs font-mono">
              暂无 24 小时 {{ chartMetricInfo.label }} 趋势数据
            </div>
            <template v-else>
              <div class="w-full h-full relative">
                <svg class="w-full h-full overflow-visible" viewBox="0 0 560 110" preserveAspectRatio="none">
                  <defs>
                    <linearGradient id="vitalsChartGrad" x1="0%" y1="0%" x2="0%" y2="100%">
                      <stop offset="0%" :stop-color="chartMetricInfo.strokeColor" stop-opacity="0.4" />
                      <stop offset="100%" :stop-color="chartMetricInfo.strokeColor" stop-opacity="0.0" />
                    </linearGradient>
                  </defs>

                  <!-- 背景轴线 & 纵坐标刻度数值 (Y-Axis Values) -->
                  <line :x1="chartMetricInfo.padLeft" y1="12" :x2="chartMetricInfo.padLeft" y2="95" stroke="#334155" stroke-width="1" />

                  <g class="y-axis-labels">
                    <text 
                      v-for="(tick, idx) in chartMetricInfo.yTicks" 
                      :key="idx"
                      :x="chartMetricInfo.padLeft - 6" 
                      :y="tick.y + 3" 
                      text-anchor="end" 
                      fill="#94a3b8" 
                      font-size="9" 
                      font-family="monospace"
                    >
                      {{ tick.val }}
                    </text>
                  </g>

                  <!-- 背景参考平稳网格虚线 -->
                  <line :x1="chartMetricInfo.padLeft" y1="20" :x2="560 - chartMetricInfo.padRight" y2="20" stroke="#334155" stroke-dasharray="3 3" stroke-opacity="0.2" />
                  <line :x1="chartMetricInfo.padLeft" y1="55" :x2="560 - chartMetricInfo.padRight" y2="55" stroke="#334155" stroke-dasharray="3 3" stroke-opacity="0.2" />
                  <line :x1="chartMetricInfo.padLeft" y1="90" :x2="560 - chartMetricInfo.padRight" y2="90" stroke="#334155" stroke-dasharray="3 3" stroke-opacity="0.2" />

                  <!-- 上下 2 条阈值横虚线 (Threshold Dashed Lines) -->
                  <!-- 1. 上阈值横虚线 (上限) -->
                  <g v-if="chartMetricInfo.upperY >= 10 && chartMetricInfo.upperY <= 95">
                    <line 
                      :x1="chartMetricInfo.padLeft" 
                      :y1="chartMetricInfo.upperY" 
                      :x2="560 - chartMetricInfo.padRight" 
                      :y2="chartMetricInfo.upperY" 
                      stroke="#ef4444" 
                      stroke-dasharray="4 4" 
                      stroke-width="1.2" 
                      stroke-opacity="0.85" 
                    />
                    <text 
                      :x="560 - chartMetricInfo.padRight - 2" 
                      :y="chartMetricInfo.upperY - 3" 
                      text-anchor="end" 
                      fill="#f87171" 
                      font-size="8" 
                      font-family="monospace"
                      font-weight="bold"
                    >
                      上限 {{ chartMetricInfo.upperThreshold }} {{ chartMetricInfo.unit }}
                    </text>
                  </g>

                  <!-- 2. 下阈值横虚线 (下限) -->
                  <g v-if="chartMetricInfo.lowerY >= 10 && chartMetricInfo.lowerY <= 95">
                    <line 
                      :x1="chartMetricInfo.padLeft" 
                      :y1="chartMetricInfo.lowerY" 
                      :x2="560 - chartMetricInfo.padRight" 
                      :y2="chartMetricInfo.lowerY" 
                      stroke="#f59e0b" 
                      stroke-dasharray="4 4" 
                      stroke-width="1.2" 
                      stroke-opacity="0.85" 
                    />
                    <text 
                      :x="560 - chartMetricInfo.padRight - 2" 
                      :y="chartMetricInfo.lowerY + 9" 
                      text-anchor="end" 
                      fill="#fbbf24" 
                      font-size="8" 
                      font-family="monospace"
                      font-weight="bold"
                    >
                      下限 {{ chartMetricInfo.lowerThreshold }} {{ chartMetricInfo.unit }}
                    </text>
                  </g>

                  <!-- 渐变阴影填充面积 -->
                  <polygon :points="chartMetricInfo.areaStr" fill="url(#vitalsChartGrad)" />

                  <!-- 时间点连接折线 -->
                  <polyline 
                    :points="chartMetricInfo.polylineStr" 
                    fill="none" 
                    :stroke="chartMetricInfo.strokeColor" 
                    stroke-width="2.5" 
                    stroke-linecap="round" 
                    stroke-linejoin="round" 
                  />

                  <!-- 时间节点：仅在有真实上发数据的时刻渲染节点圆圈 -->
                  <g v-for="(p, idx) in chartMetricInfo.validPoints" :key="idx" class="group cursor-pointer">
                    <circle 
                      :cx="p.x" 
                      :cy="p.y" 
                      r="4" 
                      :fill="chartMetricInfo.strokeColor" 
                      stroke="#0f172a" 
                      stroke-width="2" 
                      class="transition-all duration-200 group-hover:r-6"
                    />
                    <!-- Tooltip 节点数据悬浮浮层 -->
                    <g class="opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none">
                      <rect 
                        :x="Math.max(5, Math.min(470, p.x - 45))" 
                        :y="Math.max(2, p.y - 28)" 
                        width="90" 
                        height="22" 
                        rx="5" 
                        fill="#0f172a" 
                        stroke="#475569" 
                        stroke-width="1" 
                      />
                      <text 
                        :x="Math.max(5, Math.min(470, p.x - 45)) + 45" 
                        :y="Math.max(2, p.y - 28) + 15" 
                        text-anchor="middle" 
                        fill="#ffffff" 
                        font-size="10" 
                        font-family="monospace" 
                        font-weight="bold"
                      >
                        {{ p.timeLabel }} : {{ p.value }} {{ chartMetricInfo.unit }}
                      </text>
                    </g>
                  </g>
                </svg>

                <!-- X 轴时间刻度 -->
                <div class="absolute bottom-0 left-[45px] right-[20px] flex justify-between px-1 text-[9px] font-mono text-slate-400 pointer-events-none transform translate-y-2">
                  <template v-if="selectedMetric === 'steps'">
                    <span v-for="t in ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '23:00']" :key="t">
                      {{ t }}
                    </span>
                  </template>
                  <template v-else>
                    <span v-for="(p, i) in chartMetricInfo.points.filter((_, index) => [0, 4, 8, 12, 16, 20, 23].includes(index))" :key="i">
                      {{ p.timeLabel }}
                    </span>
                  </template>
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
            <span class="text-[10px] text-slate-400 font-mono">累计告警: {{ alarmList.length }} 次</span>
          </div>

          <div v-if="alarmList.length === 0" class="flex-1 flex items-center justify-center text-xs text-slate-500 font-mono py-6">
            暂无历史告警记录
          </div>
          <div v-else class="space-y-2 flex-1 overflow-y-auto max-h-[140px] pr-1">
            <div 
              v-for="item in alarmList"
              :key="item.id"
              :class="['border rounded-xl p-2.5 flex items-center justify-between text-xs',
                item.category === 'SOS' || item.category === 'FALL' ? 'bg-red-950/30 border-red-500/30' : 'bg-slate-900/60 border-slate-800']"
            >
              <div>
                <div class="font-bold flex items-center space-x-2" :class="item.category === 'SOS' || item.category === 'FALL' ? 'text-red-300' : 'text-slate-300'">
                  <span>{{ item.alert_type }}</span>
                  <span class="text-[9px] px-1.5 py-0.5 rounded bg-red-500/30 text-red-200 font-mono">{{ item.category }}</span>
                </div>
                <div class="text-[10px] text-slate-400 mt-0.5">
                  触发心率: {{ item.heart_rate || '--' }} bpm · 位置: ({{ item.latitude.toFixed(4) }}, {{ item.longitude.toFixed(4) }})
                </div>
              </div>
              <div class="text-right font-mono">
                <span :class="['font-bold text-[9px] px-2 py-0.5 rounded-full border',
                  item.status === 'COMPLETED' ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20' : 'text-amber-400 bg-amber-500/10 border-amber-500/20']">
                  {{ item.status === 'COMPLETED' ? '已处理' : '未处理' }}
                </span>
                <div class="text-[10px] text-slate-500 mt-0.5">{{ item.trigger_time }}</div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>

    <!-- ===== TAB 2: 🛣️ 历史轨迹追溯视图 ===== -->
    <div v-else-if="activeTab === 'trajectory'" class="grid grid-cols-12 gap-5 flex-1">
      <div class="col-span-12 lg:col-span-8 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[480px] relative flex flex-col">
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-3">
              <Navigation :size="16" class="text-blue-400" />
              <span class="text-xs font-bold text-slate-200">历史轨迹 Polyline 动画播放</span>
            </div>

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

          <div class="flex-1 relative">
            <AmapDashboard 
              :devices="[device]" 
              :selectedImei="device.imei"
              :showTrack="trajectoryWaypoints.length > 0"
              :trackPoints="trajectoryWaypoints"
              :runnerPoint="runnerCurrentPoint"
              :showHeatmap="false"
            />
          </div>

          <!-- 底部轨迹播放控制器 -->
          <div class="p-3 border-t border-slate-800 bg-slate-900/90 backdrop-blur-md flex items-center justify-between z-10 font-mono text-xs">
            <div class="flex items-center space-x-3">
              <button 
                @click="togglePlayback" 
                :disabled="trajectoryWaypoints.length === 0"
                class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold flex items-center space-x-1.5 shadow-lg shadow-blue-600/30 active:scale-95 transition disabled:opacity-50"
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
              <span v-if="trajectoryPointsData.length > 0">当前驻留点: <strong class="text-blue-400 font-bold">{{ trajectoryPointsData[currentTrackStep]?.locationName }}</strong> ({{ currentTrackStep + 1 }}/{{ trajectoryPointsData.length }})</span>
              <span v-else class="text-slate-500">暂无定位轨迹点</span>
            </div>
          </div>
        </div>
      </div>

      <div class="col-span-12 lg:col-span-4 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl p-5 border border-slate-800 space-y-3">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 border-b border-slate-800 pb-2">
            <Navigation :size="16" class="text-blue-400" />
            <span>今日活动里程与停留汇总</span>
          </h3>

          <div class="grid grid-cols-2 gap-2 text-xs font-mono">
            <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
              <span class="text-slate-400 block text-[10px]">全天移动总里程</span>
              <span class="text-blue-400 font-bold text-sm">{{ totalDistance.toFixed(2) }} 公里</span>
            </div>
            <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
              <span class="text-slate-400 block text-[10px]">平均移动时速</span>
              <span class="text-emerald-400 font-bold text-sm">{{ avgSpeed.toFixed(1) }} km/h</span>
            </div>
          </div>
        </div>

        <div class="glass-panel rounded-2xl p-5 border border-slate-800 flex-1 flex flex-col min-h-[300px]">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 mb-3 border-b border-slate-800 pb-2">
            <MapPin :size="16" class="text-cyan-400" />
            <span>沿途中文地标与采样点明细</span>
          </h3>

          <div v-if="trajectoryPointsData.length === 0" class="flex-1 flex items-center justify-center text-xs text-slate-500 font-mono py-8">
            暂无历史轨迹记录
          </div>
          <div v-else class="space-y-2.5 flex-1 overflow-y-auto max-h-[360px] pr-1 font-mono text-xs">
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
                    <span 
                      v-if="item.location_type || item.fix_mode" 
                      :class="['text-[9px] px-1.5 py-0.2 rounded border font-mono font-normal',
                        (item.location_type || item.fix_mode) === 'GPS' ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30' :
                        (item.location_type || item.fix_mode) === 'WIFI' ? 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30' :
                        'bg-amber-500/20 text-amber-300 border-amber-500/30']"
                    >
                      {{ item.location_type || item.fix_mode }}
                    </span>
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
      <div class="col-span-12 lg:col-span-8 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl overflow-hidden border border-slate-800 flex-1 min-h-[480px] relative flex flex-col">
          <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/80 backdrop-blur-md z-10">
            <div class="flex items-center space-x-3">
              <Flame :size="16" class="text-amber-400" />
              <span class="text-xs font-bold text-slate-200">活动热力密度图层</span>
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
              :showHeatmap="heatmapDataPoints.length > 0"
              :heatmapPoints="heatmapDataPoints"
              :heatmapRadius="heatmapRadius"
            />
          </div>
        </div>
      </div>

      <div class="col-span-12 lg:col-span-4 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl p-5 border border-slate-800 space-y-3">
          <h3 class="text-xs font-bold text-slate-200 flex items-center space-x-2 border-b border-slate-800 pb-2">
            <Flame :size="16" class="text-amber-400" />
            <span>长者常去地标驻留 TOP 榜单</span>
          </h3>

          <div v-if="topLandmarks.length === 0" class="text-xs text-slate-500 py-6 text-center font-mono">
            暂无常驻地标与热力分布数据
          </div>
          <div v-else class="space-y-2.5 font-mono text-xs">
            <div 
              v-for="item in topLandmarks"
              :key="item.rank"
              class="bg-slate-900/80 p-3 rounded-xl border border-slate-800 flex items-center justify-between"
            >
              <div class="flex items-center space-x-2">
                <span class="w-5 h-5 rounded-full bg-amber-500/20 text-amber-400 flex items-center justify-center font-bold text-[10px]">{{ item.rank }}</span>
                <div>
                  <div class="font-bold text-white">{{ item.name }}</div>
                  <div class="text-[10px] text-slate-400">{{ item.address }}</div>
                </div>
              </div>
              <span class="text-amber-400 font-bold">驻留 {{ item.percentage }}%</span>
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
              长者日常活动半径保持在离家安全圈范围内，活动轨迹高度规律，无偏僻危险区域落单记录。
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 调阅全量健康体征研判档案 模态框/大抽屉 ===== -->
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
              <div>· 静息心率上报: <strong class="text-white">{{ isInvalidHealthVal(device.last_heart_rate) ? '暂无数据' : `${device.last_heart_rate} bpm` }}</strong>；</div>
              <div>· 参考血压上报: <strong class="text-amber-300">{{ isInvalidHealthVal(device.bp) ? '暂无数据' : device.bp }}</strong>；</div>
              <div>· 血氧饱和度: <strong class="text-cyan-300">{{ isInvalidHealthVal(device.spo2) ? '暂无数据' : `${device.spo2}%` }}</strong>；表体温度: <strong class="text-emerald-300">{{ isInvalidHealthVal(device.temperature) ? '暂无数据' : `${device.temperature}°C` }}</strong>；</div>
              <div>· 历史告警工单记录: <strong class="text-rose-400">{{ alarmList.length }} 次</strong>。</div>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="bg-slate-950/60 p-4 rounded-2xl border border-slate-800 space-y-3">
              <h4 class="font-bold text-slate-200 flex items-center space-x-2 text-xs font-sans">
                <Heart :size="16" class="text-rose-400" />
                <span>1. 心率与血压数据汇总</span>
              </h4>
              <div class="grid grid-cols-3 gap-2 text-center text-[11px]">
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">最新心率</span>
                  <span class="text-red-400 font-bold text-sm">{{ isInvalidHealthVal(device.last_heart_rate) ? '--' : device.last_heart_rate }}</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">静息心率均值</span>
                  <span class="text-emerald-400 font-bold text-sm">{{ vitalsData.hr.length > 0 ? Math.round(vitalsData.hr.reduce((a,b)=>a+b,0)/vitalsData.hr.length) : '--' }}</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">参考血压</span>
                  <span class="text-amber-300 font-bold text-sm">{{ isInvalidHealthVal(device.bp) ? '--' : device.bp }}</span>
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
                  <span class="text-slate-400 block text-[10px]">血氧值</span>
                  <span class="text-cyan-300 font-bold text-sm">{{ isInvalidHealthVal(device.spo2) ? '--' : `${device.spo2}%` }}</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">体温值</span>
                  <span class="text-emerald-300 font-bold text-sm">{{ isInvalidHealthVal(device.temperature) ? '--' : `${device.temperature}°C` }}</span>
                </div>
                <div class="bg-slate-900 p-2 rounded-xl border border-slate-800">
                  <span class="text-slate-400 block text-[10px]">今日计步</span>
                  <span class="text-indigo-300 font-bold text-sm">{{ isInvalidHealthVal(device.steps) ? '--' : device.steps }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="pt-2 border-t border-slate-800 flex items-center justify-between text-slate-400 text-xs">
            <div>报告生成时间: {{ new Date().toLocaleString() }}</div>
            <div class="flex items-center space-x-4">
              <span>主治医师: <strong class="text-white">林医生 (已审核)</strong></span>
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
