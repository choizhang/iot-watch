<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useDashboardStore } from '../store/dashboard'
import AmapDashboard from '../components/AmapDashboard.vue'
import { 
  Activity, 
  ShieldAlert, 
  Wifi, 
  WifiOff, 
  Volume2, 
  VolumeX, 
  RefreshCw, 
  Send, 
  PhoneCall, 
  CheckCircle2, 
  Clock, 
  User, 
  MapPin, 
  Search,
  ExternalLink,
  Battery,
  Heart,
  Bell,
  Check,
  ChevronRight,
  ChevronDown
} from 'lucide-vue-next'

const store = useDashboardStore()
const filterStatus = ref<'all' | 'online' | 'sos_alert' | 'offline'>('all')
// 默认选定 SOS 紧急求救 Tab (去掉 "全部" Tab)
const alarmCategoryFilter = ref<'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE'>('SOS')
const searchQuery = ref('')
const commandStatus = ref<'idle' | 'sending' | 'success' | 'error'>('idle')
const commandMsg = ref('')

// 辖区折叠/展开状态记录 (默认全部收缩)
const expandedDistricts = ref<Record<string, boolean>>({})

const toggleDistrict = (dist: string) => {
  expandedDistricts.value[dist] = !expandedDistricts.value[dist]
}

const isDistrictExpanded = (dist: string) => {
  return !!expandedDistricts.value[dist]
}

let timer: any = null

onMounted(() => {
  store.fetchAllDevices()
  timer = setInterval(() => {
    if (store.autoRefresh) {
      store.fetchAllDevices()
    }
  }, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const handleStatusFilter = (status: 'all' | 'online' | 'sos_alert' | 'offline') => {
  filterStatus.value = status
}

// 容错与全覆盖的告警分类映射器
const getAlarmCat = (alarm: any): 'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE' => {
  if (alarm && (alarm.category === 'SOS' || alarm.category === 'FALL' || alarm.category === 'VITAL' || alarm.category === 'GEOFENCE')) {
    return alarm.category
  }
  const type = alarm?.alert_type || ''
  if (type.includes('跌倒') || type.includes('摔倒') || type.includes('姿态')) return 'FALL'
  if (type.includes('心率') || type.includes('体征') || type.includes('发烧')) return 'VITAL'
  if (type.includes('越界') || type.includes('围栏')) return 'GEOFENCE'
  return 'SOS'
}

// 主色调视觉样式映射器 (实现 Tab、标题、卡片边框与定位按钮全颜色统一)
const getCategoryTheme = (cat: 'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE') => {
  if (cat === 'SOS') {
    return {
      tabActive: 'bg-red-600 text-white shadow-lg shadow-red-600/30 border-red-400/50',
      tabInactive: 'text-red-400 hover:text-white hover:bg-red-950/30',
      cardBorder: 'bg-red-950/40 border-red-500/50 hover:border-red-400',
      badge: 'bg-red-500/20 text-red-300 border-red-500/40 animate-pulse',
      button: 'bg-red-600 hover:bg-red-500 text-white shadow-red-600/30',
      headerTitle: '实时 SOS 紧急求救'
    }
  }
  if (cat === 'FALL') {
    return {
      tabActive: 'bg-orange-600 text-white shadow-lg shadow-orange-600/30 border-orange-400/50',
      tabInactive: 'text-orange-400 hover:text-white hover:bg-orange-950/30',
      cardBorder: 'bg-orange-950/40 border-orange-500/50 hover:border-orange-400',
      badge: 'bg-orange-500/20 text-orange-300 border-orange-500/40',
      button: 'bg-orange-600 hover:bg-orange-500 text-white shadow-orange-600/30',
      headerTitle: '实时跌倒姿态异常'
    }
  }
  if (cat === 'VITAL') {
    return {
      tabActive: 'bg-pink-600 text-white shadow-lg shadow-pink-600/30 border-pink-400/50',
      tabInactive: 'text-pink-400 hover:text-white hover:bg-pink-950/30',
      cardBorder: 'bg-pink-950/40 border-pink-500/50 hover:border-pink-400',
      badge: 'bg-pink-500/20 text-pink-300 border-pink-500/40',
      button: 'bg-pink-600 hover:bg-pink-500 text-white shadow-pink-600/30',
      headerTitle: '实时体征健康风险'
    }
  }
  return {
    tabActive: 'bg-cyan-600 text-white shadow-lg shadow-cyan-600/30 border-cyan-400/50',
    tabInactive: 'text-cyan-400 hover:text-white hover:bg-cyan-950/30',
    cardBorder: 'bg-cyan-950/40 border-cyan-500/50 hover:border-cyan-400',
    badge: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/40',
    button: 'bg-cyan-600 hover:bg-cyan-500 text-white shadow-cyan-600/30',
    headerTitle: '实时电子围栏出界'
  }
}

const countSOS = computed(() => store.alarmOrders.filter(a => getAlarmCat(a) === 'SOS').length)
const countFALL = computed(() => store.alarmOrders.filter(a => getAlarmCat(a) === 'FALL').length)
const countVITAL = computed(() => store.alarmOrders.filter(a => getAlarmCat(a) === 'VITAL').length)
const countGEOFENCE = computed(() => store.alarmOrders.filter(a => getAlarmCat(a) === 'GEOFENCE').length)

const criticalCount = computed(() => store.alarmOrders.filter(a => {
  const cat = getAlarmCat(a)
  return cat === 'SOS' || cat === 'FALL'
}).length)

const warningCount = computed(() => store.alarmOrders.filter(a => {
  const cat = getAlarmCat(a)
  return cat === 'VITAL' || cat === 'GEOFENCE'
}).length)

// 按时间正序 (较老的数据优先展示，最早触发的排在最前 #01)
const filteredAlarmOrders = computed(() => {
  const list = store.alarmOrders.filter(a => getAlarmCat(a) === alarmCategoryFilter.value)
  return list.slice().sort((a, b) => a.id - b.id)
})

// 获取设备列表中告警态设备的精细化 Badge 标签
const getDeviceAlarmBadge = (imei: string) => {
  const alarm = store.alarmOrders.find(a => a.device_imei === imei)
  const cat = alarm ? getAlarmCat(alarm) : 'SOS'
  if (cat === 'SOS') return { text: 'SOS求救', style: 'bg-red-600 text-white animate-pulse' }
  if (cat === 'FALL') return { text: '跌倒告警', style: 'bg-orange-600 text-white font-bold' }
  if (cat === 'VITAL') return { text: '体征异常', style: 'bg-pink-600 text-white font-bold' }
  if (cat === 'GEOFENCE') return { text: '围栏越界', style: 'bg-cyan-600 text-white font-bold' }
  return { text: '告警中', style: 'bg-red-600 text-white' }
}

const sendFindCommand = async (imei: string) => {
  commandStatus.value = 'sending'
  commandMsg.value = ''
  try {
    const res = await store.sendDeviceCommand(imei, 'FIND')
    commandStatus.value = 'success'
    commandMsg.value = res.message || '寻物指令已成功下发至设备'
    setTimeout(() => { commandStatus.value = 'idle' }, 3000)
  } catch (err: any) {
    commandStatus.value = 'error'
    commandMsg.value = err.message || '下发指令失败'
  }
}

const filteredDevices = () => {
  return store.devices.filter(d => {
    const matchStatus = filterStatus.value === 'all' || d.status === filterStatus.value
    const matchQuery = !searchQuery.value || d.owner_name.includes(searchQuery.value) || d.imei.includes(searchQuery.value)
    return matchStatus && matchQuery
  })
}

// 解析设备所属城市
const getDeviceCity = (dev: any): string => {
  if (dev.city) return dev.city
  const lat = Number(dev.last_latitude)
  const lon = Number(dev.last_longitude)
  if ((lat >= 22.0 && lat <= 22.6 && lon >= 113.8 && lon <= 114.5) || dev.imei === '1234567890') {
    return '香港'
  }
  const raw = (dev.owner_name || '') + (dev.address || '') + (dev.device_name || '')
  if (raw.includes('香港') || raw.includes('葵涌') || raw.includes('葵青') || raw.includes('荃湾') || raw.includes('945M')) {
    return '香港'
  }
  return '成都市'
}

// 展开/收缩控制 (一级城市默认展开)
const expandedCities = ref<Set<string>>(new Set(['成都市', '香港']))

const isCityExpanded = (city: string) => expandedCities.value.has(city)
const toggleCity = (city: string) => {
  if (expandedCities.value.has(city)) {
    expandedCities.value.delete(city)
  } else {
    expandedCities.value.add(city)
  }
}

// 解析设备所属行政区
const getDeviceDistrict = (dev: any): string => {
  const city = getDeviceCity(dev)
  const raw = (dev.owner_name?.match(/\((.*?)\)/)?.[1] || '') + (dev.address || '')
  if (city === '成都市') {
    if (raw.includes('武侯')) return '武侯区'
    if (raw.includes('锦江')) return '锦江区'
    if (raw.includes('青羊')) return '青羊区'
    if (raw.includes('高新')) return '高新区'
    if (raw.includes('金牛')) return '金牛区'
    if (raw.includes('成华')) return '成华区'
    if (raw.includes('双流')) return '双流区'
    if (raw.includes('温江')) return '温江区'
    return '成都市辖区'
  } else {
    if (raw.includes('荃湾')) return '荃湾区'
    if (raw.includes('葵涌') || raw.includes('葵青')) return '葵青区'
    if (raw.includes('沙田')) return '沙田区'
    if (raw.includes('油尖旺') || raw.includes('旺角') || raw.includes('尖沙咀')) return '油尖旺区'
    if (raw.includes('深水埗') || raw.includes('长沙湾')) return '深水埗区'
    if (raw.includes('九龙城') || raw.includes('红磡')) return '九龙城区'
    if (raw.includes('黄大仙')) return '黄大仙区'
    if (raw.includes('观塘') || raw.includes('九龙湾')) return '观塘区'
    if (raw.includes('屯门')) return '屯门区'
    if (raw.includes('元朗')) return '元朗区'
    if (raw.includes('北角') || raw.includes('北区') || raw.includes('上水')) return '北区'
    if (raw.includes('大埔')) return '大埔区'
    if (raw.includes('西贡') || raw.includes('将军澳')) return '西贡区'
    if (raw.includes('中西区') || raw.includes('中环') || raw.includes('坚尼地城')) return '中西区'
    if (raw.includes('湾仔') || raw.includes('铜锣湾')) return '湾仔区'
    if (raw.includes('东区')) return '东区'
    if (raw.includes('南区') || raw.includes('香港仔')) return '南区'
    if (raw.includes('离岛') || raw.includes('东涌')) return '离岛区'
    return '港九新界'
  }
}

// 三级分组：城市 → 区 → 设备
const groupedByCity = computed(() => {
  const devs = filteredDevices()
  // city → district → devices
  const cityMap = new Map<string, Map<string, any[]>>()

  devs.forEach(d => {
    const city = getDeviceCity(d)
    const dist = getDeviceDistrict(d)
    if (!cityMap.has(city)) cityMap.set(city, new Map())
    const distMap = cityMap.get(city)!
    if (!distMap.has(dist)) distMap.set(dist, [])
    distMap.get(dist)!.push(d)
  })

  const result: { city: string; total: number; districts: { district: string; devices: any[] }[] }[] = []

  // 城市顺序: 成都市优先（真实设备所在地），然后香港
  const cityOrder = ['成都市', '香港']
  cityOrder.forEach(city => {
    if (!cityMap.has(city)) return
    const distMap = cityMap.get(city)!
    const groups: { district: string; devices: any[] }[] = []
    distMap.forEach((deviceList, dist) => {
      deviceList.sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
      groups.push({ district: dist, devices: deviceList })
    })
    groups.sort((a, b) => {
      const aSos = a.devices.some(d => d.status === 'sos_alert') ? 1 : 0
      const bSos = b.devices.some(d => d.status === 'sos_alert') ? 1 : 0
      if (aSos !== bSos) return bSos - aSos
      return b.devices.length - a.devices.length
    })
    result.push({ city, total: Array.from(distMap.values()).reduce((s, v) => s + v.length, 0), districts: groups })
  })

  // 将其余未在 cityOrder 中的城市追加
  cityMap.forEach((distMap, city) => {
    if (cityOrder.includes(city)) return
    const groups: { district: string; devices: any[] }[] = []
    distMap.forEach((deviceList, dist) => {
      deviceList.sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
      groups.push({ district: dist, devices: deviceList })
    })
    result.push({ city, total: Array.from(distMap.values()).reduce((s, v) => s + v.length, 0), districts: groups })
  })

  return result
})

const formatTime = (ts: number) => {
  if (!ts) return '--'
  const d = new Date(ts * 1000)
  return d.toTimeString().split(' ')[0]
}

const formatLocationStatus = (ts: number) => {
  if (!ts) return '⚪ 位置缺失'
  const diffSec = Math.floor(Date.now() / 1000) - ts
  if (diffSec < 300) return '🟢 实时 GPS'
  if (diffSec < 1800) return `🟡 辅助定位 (${Math.floor(diffSec / 60)}分钟前)`
  return `⚪ 最后已知定位 (${Math.floor(diffSec / 60)}分钟前)`
}
</script>

<template>
  <div class="h-screen bg-slate-950 text-slate-100 flex flex-col p-3 md:p-4 space-y-3 relative overflow-hidden">
    
    <!-- 全局 Toast 提示通知弹窗 -->
    <div v-if="store.toastMessage" class="fixed top-6 left-1/2 -translate-x-1/2 z-50 bg-amber-500 text-slate-950 font-black px-6 py-2.5 rounded-2xl shadow-2xl backdrop-blur-md border border-amber-300 text-xs md:text-sm flex items-center space-x-2 animate-bounce">
      <span>💡 {{ store.toastMessage }}</span>
    </div>

    <!-- Header 指挥中心大屏顶部导航栏 -->
    <header class="glass-panel px-5 py-3 rounded-2xl flex items-center justify-between border-slate-800 shadow-2xl flex-shrink-0">
      <div class="flex items-center space-x-3">
        <div class="w-10 h-10 rounded-xl bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
          <Activity :size="24" class="animate-pulse" />
        </div>
        <div>
          <h1 class="text-xl font-extrabold tracking-wide bg-gradient-to-r from-cyan-400 via-blue-400 to-indigo-400 bg-clip-text text-transparent">
            社区养老智慧安防指挥控制中心
          </h1>
          <p class="text-xs text-slate-400 font-mono flex items-center space-x-2">
            <span>ELDER-GUARD IoT COMMAND CENTER</span>
            <span>·</span>
            <span>系统状态: 正常运行中</span>
          </p>
        </div>
      </div>

      <!-- 右侧控制区 -->
      <div class="flex items-center space-x-3">
        <!-- Mock 模式切换按钮 (100台全场景 MOCK vs 真实后端 TCP) -->
        <button 
          @click="store.toggleMockMode()" 
          :class="['px-3 py-1.5 rounded-xl text-xs font-bold transition flex items-center space-x-1.5 border active:scale-95',
            store.mockMode ? 'bg-amber-500/10 border-amber-500/30 text-amber-400' : 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400']"
        >
          <span class="w-2 h-2 rounded-full" :class="store.mockMode ? 'bg-amber-400 animate-ping' : 'bg-emerald-400'"></span>
          <span>{{ store.mockMode ? '测试 MOCK 模式 (100台)' : '实时 TCP 直连模式' }}</span>
        </button>

        <!-- 声光报警开关 -->
        <button 
          @click="store.toggleSound()" 
          class="p-2.5 rounded-xl bg-slate-900 border border-slate-800 hover:bg-slate-800 text-slate-300 transition"
          :title="store.soundEnabled ? '声音警报已开启' : '静音模式'"
        >
          <Volume2 v-if="store.soundEnabled" :size="18" class="text-cyan-400" />
          <VolumeX v-else :size="18" class="text-slate-500" />
        </button>

        <!-- 手动刷新 -->
        <button 
          @click="store.fetchAllDevices()" 
          class="p-2.5 rounded-xl bg-slate-900 border border-slate-800 hover:bg-slate-800 text-slate-300 transition"
          title="手动刷新"
        >
          <RefreshCw :size="18" :class="store.loading ? 'animate-spin text-cyan-400' : 'text-slate-400'" />
        </button>

        <!-- 跳转 C端移动端 -->
        <a 
          href="http://localhost:5173" 
          target="_blank"
          class="px-3.5 py-2 bg-slate-900 hover:bg-slate-800 border border-slate-800 text-cyan-400 hover:text-cyan-300 rounded-xl text-xs font-bold flex items-center space-x-1.5 transition"
        >
          <span>进入 C端家属小程序</span>
          <ExternalLink :size="14" />
        </a>
      </div>
    </header>

    <!-- 主板区：KPI 统计栏 + 三栏响应式全屏自适应布局 (min-h-0 overflow-hidden 撑满视口) -->
    <div class="grid grid-cols-12 gap-3.5 flex-1 min-h-0 overflow-hidden">
      
      <!-- ===== 左栏 (3 列): KPI 指示牌与设备检索 ===== -->
      <div class="col-span-12 lg:col-span-3 flex flex-col space-y-3 h-full min-h-0 overflow-hidden">
        <!-- 4 KPI 统计网格 -->
        <div class="grid grid-cols-2 gap-2.5 flex-shrink-0">
          <div class="glass-panel p-3 rounded-2xl flex flex-col justify-between">
            <span class="text-xs font-medium text-slate-400">管控设备总数</span>
            <div class="text-2xl font-black text-white mt-1 font-mono">{{ store.totalCount }} <span class="text-xs font-normal text-slate-500">台</span></div>
          </div>
          <div class="glass-panel p-3 rounded-2xl flex flex-col justify-between">
            <span class="text-xs font-medium text-slate-400">设备在线率</span>
            <div class="text-2xl font-black text-emerald-400 mt-1 font-mono">
              {{ store.onlineRate }}%
            </div>
          </div>
          <div class="glass-panel-danger p-3 rounded-2xl flex flex-col justify-between border-red-500/40">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-red-300">紧急救援 (SOS/跌倒)</span>
              <ShieldAlert :size="16" class="text-red-400 animate-bounce" />
            </div>
            <div class="text-2xl font-black text-red-400 mt-1 font-mono">{{ criticalCount }} <span class="text-xs font-normal text-red-300">起</span></div>
          </div>
          <div class="glass-panel p-3 rounded-2xl flex flex-col justify-between border-amber-500/30">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-amber-300">防护预警 (体征/围栏)</span>
              <Activity :size="16" class="text-amber-400" />
            </div>
            <div class="text-2xl font-black text-amber-400 mt-1 font-mono">{{ warningCount }} <span class="text-xs font-normal text-amber-300">起</span></div>
          </div>
        </div>

        <!-- 设备列表与过滤面板 (flex-1 min-h-0 内部高度自适应) -->
        <div class="glass-panel rounded-2xl p-3.5 flex-1 flex flex-col min-h-0 overflow-hidden border border-slate-800">
          <div class="flex items-center justify-between mb-2.5 flex-shrink-0">
            <h3 class="text-sm font-bold text-slate-200 flex items-center space-x-2">
              <span>辖区设备列表</span>
              <span class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">{{ filteredDevices().length }}</span>
            </h3>
            <span class="text-[10px] text-slate-500 font-mono">全港 18 区覆盖</span>
          </div>

          <!-- 搜索与状态 Filter -->
          <div class="space-y-2 mb-2.5 flex-shrink-0">
            <div class="relative">
              <Search :size="14" class="absolute left-3 top-2.5 text-slate-500" />
              <input 
                v-model="searchQuery"
                type="text" 
                placeholder="搜索姓名或 IMEI..." 
                class="w-full bg-slate-900/80 border border-slate-800 rounded-xl pl-8 pr-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-cyan-500"
              />
            </div>

            <div class="grid grid-cols-4 gap-1 bg-slate-900 p-1 rounded-xl border border-slate-800 text-[11px]">
              <button 
                @click="handleStatusFilter('all')" 
                :class="['py-1 rounded-lg font-bold transition', filterStatus === 'all' ? 'bg-cyan-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >
                全部({{ store.totalCount }})
              </button>
              <button 
                @click="handleStatusFilter('online')" 
                :class="['py-1 rounded-lg font-bold transition', filterStatus === 'online' ? 'bg-emerald-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >
                在线({{ store.onlineCount }})
              </button>
              <button 
                @click="handleStatusFilter('sos_alert')" 
                :class="['py-1 rounded-lg font-bold transition', filterStatus === 'sos_alert' ? 'bg-red-600 text-white shadow' : 'text-slate-400 hover:text-white']"
              >
                告警({{ store.sosCount }})
              </button>
              <button 
                @click="handleStatusFilter('offline')" 
                :class="['py-1 rounded-lg font-bold transition', filterStatus === 'offline' ? 'bg-slate-700 text-white shadow' : 'text-slate-400 hover:text-white']"
              >
                离线({{ store.offlineCount }})
              </button>
            </div>
          </div>

          <!-- 设备按城市 → 区域分组列表 -->
          <div class="space-y-2 flex-1 overflow-y-auto min-h-0 pr-1 custom-scrollbar">
            <template v-for="cityGroup in groupedByCity" :key="cityGroup.city">
              <!-- 城市一级收缩/展开 Header -->
              <div 
                @click="toggleCity(cityGroup.city)"
                class="flex items-center justify-between px-2.5 py-1.5 rounded-xl cursor-pointer hover:bg-slate-800/80 transition border border-slate-800 select-none my-1 backdrop-blur-md"
                :class="cityGroup.city === '成都市' ? 'bg-emerald-950/40 border-emerald-900/60' : 'bg-blue-950/40 border-blue-900/60'"
              >
                <div class="flex items-center space-x-2">
                  <ChevronRight v-if="!isCityExpanded(cityGroup.city)" :size="15" class="text-slate-400" />
                  <ChevronDown v-else :size="15" class="text-cyan-400" />
                  <span class="text-xs font-black tracking-wider uppercase px-2 py-0.5 rounded-md"
                    :class="cityGroup.city === '成都市' ? 'bg-emerald-900/70 text-emerald-300 border border-emerald-700/60' : 'bg-blue-900/70 text-blue-300 border border-blue-700/60'"
                  >
                    {{ cityGroup.city === '成都市' ? '🏙 成都市' : '🌃 香港' }}
                  </span>
                </div>
                <span class="text-[11px] font-mono text-slate-400 bg-slate-900 px-2 py-0.5 rounded-full border border-slate-800">
                  {{ cityGroup.total }} 台
                </span>
              </div>

              <!-- 城市下一级区列表 (按城市展开状态渲染) -->
              <div v-if="isCityExpanded(cityGroup.city)" class="space-y-2 pl-1 transition-all">
                <div v-for="group in cityGroup.districts" :key="cityGroup.city + group.district" class="space-y-1.5">
                  <!-- 区域 Header -->
                  <div
                    @click="toggleDistrict(cityGroup.city + group.district)"
                    class="flex items-center justify-between text-[11px] font-bold text-slate-200 bg-slate-900/90 hover:bg-slate-800/90 px-3 py-2 rounded-xl border border-slate-800 cursor-pointer sticky top-0 backdrop-blur-md z-10 transition select-none"
                  >
                    <span class="flex items-center space-x-2 text-cyan-300">
                      <ChevronRight v-if="!isDistrictExpanded(cityGroup.city + group.district)" :size="14" class="text-slate-400" />
                      <ChevronDown v-else :size="14" class="text-cyan-400" />
                      <MapPin :size="13" class="text-cyan-400" />
                      <span>{{ group.district }}</span>
                    </span>
                    <span class="text-[10px] font-mono text-slate-400 bg-slate-800 px-2 py-0.5 rounded-full border border-slate-700/50">
                      {{ group.devices.length }} 台
                    </span>
                  </div>

                  <!-- 区域内设备卡片 -->
                  <div v-if="isDistrictExpanded(cityGroup.city + group.district)" class="space-y-1.5 pl-1">
                    <div
                      v-for="dev in group.devices"
                      :key="dev.imei"
                      @click="store.selectDevice(dev.imei)"
                      :class="['p-2.5 rounded-xl border transition cursor-pointer flex items-center justify-between',
                        store.selectedImei === dev.imei ? 'bg-cyan-950/40 border-cyan-500/60 shadow-lg shadow-cyan-500/10' :
                        dev.status === 'sos_alert' ? 'bg-red-950/20 border-red-500/40 hover:bg-red-900/20' : 'bg-slate-900/60 border-slate-800/80 hover:bg-slate-800/60']"
                    >
                    <div class="flex items-center space-x-2.5">
                      <div :class="['w-2.5 h-2.5 rounded-full flex-shrink-0',
                        dev.status === 'sos_alert' ? 'bg-red-500 animate-ping' :
                        dev.status === 'online' ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-slate-600']"
                      ></div>
                      <div>
                        <div class="text-xs font-bold text-slate-100 flex items-center space-x-1.5">
                          <span>{{ dev.owner_name }}</span>
                          <span v-if="dev.status === 'sos_alert'" :class="['text-[9px] px-1.5 py-0.5 rounded font-mono font-bold flex items-center space-x-1', getDeviceAlarmBadge(dev.imei).style]">
                            {{ getDeviceAlarmBadge(dev.imei).text }}
                          </span>
                        </div>
                        <div class="text-[10px] text-slate-400 font-mono mt-0.5 leading-snug break-all">{{ dev.address }}</div>
                      </div>
                    </div>
                    <div class="text-right font-mono">
                      <div class="text-xs text-slate-300">{{ dev.last_heart_rate ? dev.last_heart_rate + ' bpm' : '--' }}</div>
                      <div class="text-[10px] text-slate-500">{{ formatTime(dev.updated_at) }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </div>
        </div>
      </div>

      <!-- ===== 中栏 (6 列): 高德地图大屏与选中设备控制 (h-full min-h-0 高度填充) ===== -->
      <div class="col-span-12 lg:col-span-6 flex flex-col space-y-3 h-full min-h-0 overflow-hidden">
        <!-- 高德地图组件 (flex-1 min-h-0 地图高度自适应) -->
        <div class="flex-1 min-h-0 rounded-2xl overflow-hidden border border-slate-800">
          <AmapDashboard 
            :devices="filteredDevices()" 
            :selectedImei="store.selectedImei"
            @select-device="store.selectDevice"
            @reset-view="store.clearSelection"
          />
        </div>

        <!-- 当前选中设备卡片与控制下发盘 (地址与定位状态同行，高度最小化，地址全量完整展示不截断) -->
        <div v-if="store.selectedDevice" class="glass-panel rounded-2xl p-2 px-3 border border-slate-800 flex items-center justify-between gap-3 flex-shrink-0">
          <div class="flex items-center space-x-2.5 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-xl bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400 font-black text-sm flex-shrink-0">
              {{ store.selectedDevice.owner_name.slice(0, 1) }}
            </div>
            <div class="min-w-0 flex-1">
              <!-- 单行/多行智能自适应：姓名、告警态、心率、电量、定位状态 Pill、地址 同行连写无省号截断 -->
              <div class="flex items-center space-x-2 flex-wrap gap-y-1 font-mono text-xs">
                <span class="font-extrabold text-slate-100 text-sm whitespace-nowrap">{{ store.selectedDevice.owner_name }}</span>
                <span :class="['px-2 py-0.5 rounded-full text-[10px] font-bold whitespace-nowrap',
                  store.selectedDevice.status === 'sos_alert' ? 'bg-red-500/20 text-red-400 border border-red-500/30' :
                  store.selectedDevice.status === 'online' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' :
                  'bg-slate-800 text-slate-400']"
                >
                  {{ store.selectedDevice.status === 'sos_alert' ? 'SOS 告警中' : store.selectedDevice.status === 'online' ? '在线' : '离线' }}
                </span>
                <span class="flex items-center space-x-1 text-slate-300 font-bold whitespace-nowrap">
                  <Heart :size="12" class="text-rose-400" />
                  <span>{{ store.selectedDevice.last_heart_rate || '--' }} bpm</span>
                </span>
                <span class="flex items-center space-x-1 text-slate-300 font-bold whitespace-nowrap">
                  <Battery :size="12" class="text-amber-400" />
                  <span>{{ store.selectedDevice.battery }}%</span>
                </span>
                <span class="px-1.5 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px] text-cyan-300 font-bold whitespace-nowrap">
                  {{ formatLocationStatus(store.selectedDevice.updated_at) }}
                </span>
                <!-- 地址全量无死角展示，不使用 truncate 或 省略号 -->
                <span class="text-xs font-mono text-slate-300 flex items-center space-x-1">
                  <MapPin :size="12" class="text-cyan-400 flex-shrink-0" />
                  <span class="font-medium text-slate-200 leading-snug">
                    {{ store.selectedDevice.address }}
                  </span>
                </span>
              </div>
            </div>
          </div>

          <!-- 指令与详情跳转按钮组 (flex-shrink-0 whitespace-nowrap 绝对防止分行/缩变) -->
          <div class="flex items-center space-x-2 flex-shrink-0">
            <button 
              @click="sendFindCommand(store.selectedDevice.imei)" 
              :disabled="commandStatus === 'sending'"
              class="px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white rounded-xl text-xs font-extrabold flex items-center space-x-1.5 shadow-lg shadow-cyan-600/20 active:scale-95 transition disabled:opacity-50 whitespace-nowrap flex-shrink-0"
            >
              <Send :size="14" />
              <span>下发响铃</span>
            </button>

            <!-- 跳转至设备详情页按钮 -->
            <RouterLink 
              :to="'/device/' + store.selectedDevice.imei"
              class="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-extrabold flex items-center space-x-1.5 shadow-lg shadow-indigo-600/20 active:scale-95 transition whitespace-nowrap flex-shrink-0"
            >
              <span>查看用户详情</span>
              <ExternalLink :size="14" />
            </RouterLink>
          </div>
        </div>
      </div>

      <!-- ===== 右栏 (3 列): 告警中心与工单分流处理看板 (h-full min-h-0 独立滚动) ===== -->
      <div class="col-span-12 lg:col-span-3 flex flex-col space-y-3 h-full min-h-0 overflow-hidden">
        <div class="glass-panel rounded-2xl p-3.5 flex-1 flex flex-col border border-slate-800 min-h-0 overflow-hidden">
          <!-- 告警中心 Header -->
          <div class="flex items-center justify-between mb-2.5 border-b border-slate-800 pb-2.5 flex-shrink-0">
            <h3 class="text-sm font-extrabold text-slate-100 flex items-center space-x-2">
              <Bell :size="18" class="text-cyan-400 animate-pulse" />
              <span>{{ getCategoryTheme(alarmCategoryFilter).headerTitle }}</span>
            </h3>
          </div>

          <!-- 告警分类 Filter Tab 快捷切页 -->
          <div class="grid grid-cols-4 gap-1.5 bg-slate-900/90 p-1.5 rounded-2xl border border-slate-800 text-[11px] mb-2.5 shadow-inner flex-shrink-0">
            <button 
              @click="alarmCategoryFilter = 'SOS'" 
              :class="['py-2 px-1 rounded-xl font-black transition-all duration-200 flex items-center justify-center space-x-1 whitespace-nowrap active:scale-95 border',
                alarmCategoryFilter === 'SOS' ? getCategoryTheme('SOS').tabActive : getCategoryTheme('SOS').tabInactive + ' border-transparent']"
            >
              <span>🆘 SOS({{ countSOS }})</span>
            </button>
            <button 
              @click="alarmCategoryFilter = 'FALL'" 
              :class="['py-2 px-1 rounded-xl font-black transition-all duration-200 flex items-center justify-center space-x-1 whitespace-nowrap active:scale-95 border',
                alarmCategoryFilter === 'FALL' ? getCategoryTheme('FALL').tabActive : getCategoryTheme('FALL').tabInactive + ' border-transparent']"
            >
              <span>🤸 跌倒({{ countFALL }})</span>
            </button>
            <button 
              @click="alarmCategoryFilter = 'VITAL'" 
              :class="['py-2 px-1 rounded-xl font-black transition-all duration-200 flex items-center justify-center space-x-1 whitespace-nowrap active:scale-95 border',
                alarmCategoryFilter === 'VITAL' ? getCategoryTheme('VITAL').tabActive : getCategoryTheme('VITAL').tabInactive + ' border-transparent']"
            >
              <span>❤️ 体征({{ countVITAL }})</span>
            </button>
            <button 
              @click="alarmCategoryFilter = 'GEOFENCE'" 
              :class="['py-2 px-1 rounded-xl font-black transition-all duration-200 flex items-center justify-center space-x-1 whitespace-nowrap active:scale-95 border',
                alarmCategoryFilter === 'GEOFENCE' ? getCategoryTheme('GEOFENCE').tabActive : getCategoryTheme('GEOFENCE').tabInactive + ' border-transparent']"
            >
              <span>⭕ 围栏({{ countGEOFENCE }})</span>
            </button>
          </div>

          <!-- 告警单列表 (告警时间字体样式调和统一) -->
          <div class="space-y-2.5 flex-1 overflow-y-auto min-h-0 pr-1 custom-scrollbar">
            <div 
              v-for="(alarm, idx) in filteredAlarmOrders" 
              :key="alarm.id"
              :class="['rounded-xl p-3 space-y-2 border transition', getCategoryTheme(alarmCategoryFilter).cardBorder]"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-1.5">
                  <span class="text-[10px] font-mono font-bold px-1.5 py-0.5 rounded bg-slate-900/90 border border-slate-800 text-slate-300">
                    #{{ (idx + 1) < 10 ? '0' + (idx + 1) : (idx + 1) }}
                  </span>
                  <span :class="['text-[11px] font-extrabold px-2 py-0.5 rounded-lg border flex items-center space-x-1 font-mono', getCategoryTheme(alarmCategoryFilter).badge]">
                    <ShieldAlert v-if="alarmCategoryFilter === 'SOS' || alarmCategoryFilter === 'FALL'" :size="12" />
                    <Activity v-else :size="12" />
                    <span>{{ alarm.alert_type }}</span>
                  </span>
                </div>
              </div>

              <div class="text-xs text-slate-300 space-y-0.5 font-mono">
                <div>长者设备: <span class="font-bold text-white">{{ store.devices.find(d => d.imei === alarm.device_imei)?.owner_name || alarm.device_imei }}</span></div>
                <div>体征指标: <span :class="alarmCategoryFilter === 'VITAL' || alarmCategoryFilter === 'SOS' ? 'text-rose-400 font-bold' : 'text-slate-300'">{{ alarm.heart_rate }} bpm</span></div>
                <div>告警时间: <span class="text-slate-200 font-medium">{{ alarm.trigger_time }}</span></div>
              </div>

              <div class="pt-1 flex items-center space-x-2">
                <button 
                  @click="store.selectDevice(alarm.device_imei)"
                  :class="['flex-1 py-1.5 rounded-lg text-xs font-bold transition flex items-center justify-center space-x-1 shadow-lg', getCategoryTheme(alarmCategoryFilter).button]"
                >
                  <MapPin :size="12" />
                  <span>定位此设备</span>
                </button>
                <RouterLink 
                  :to="'/device/' + alarm.device_imei"
                  class="py-1.5 px-2 bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-700 rounded-lg text-xs font-bold transition flex items-center justify-center"
                >
                  <span>处理工单</span>
                </RouterLink>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>
