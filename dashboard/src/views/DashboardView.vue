<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
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
  Check
} from 'lucide-vue-next'

const store = useDashboardStore()
const filterStatus = ref<'all' | 'online' | 'sos_alert' | 'offline'>('all')
const alarmCategoryFilter = ref<'all' | 'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE'>('all')
const searchQuery = ref('')
const commandStatus = ref<'idle' | 'sending' | 'success' | 'error'>('idle')
const commandMsg = ref('')

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

const filteredAlarmOrders = () => {
  return store.alarmOrders.filter(a => {
    if (alarmCategoryFilter.value === 'all') return true
    return a.category === alarmCategoryFilter.value
  })
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
  <div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col p-4 space-y-4 relative">
    
    <!-- 全局 Toast 提示通知弹窗 -->
    <div v-if="store.toastMessage" class="fixed top-6 left-1/2 -translate-x-1/2 z-50 bg-amber-500 text-slate-950 font-black px-6 py-2.5 rounded-2xl shadow-2xl backdrop-blur-md border border-amber-300 text-xs md:text-sm flex items-center space-x-2 animate-bounce">
      <span>💡 {{ store.toastMessage }}</span>
    </div>

    <!-- Header 指挥中心大屏顶部导航栏 -->
    <header class="glass-panel px-6 py-3.5 rounded-2xl flex items-center justify-between border-slate-800 shadow-2xl">
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

    <!-- 主板区：KPI 统计栏 + 三栏响应式布局 -->
    <div class="grid grid-cols-12 gap-4 flex-1">
      
      <!-- ===== 左栏 (3 列): KPI 指示牌与设备检索 ===== -->
      <div class="col-span-12 lg:col-span-3 flex flex-col space-y-4">
        <!-- 4 KPI 统计网格 -->
        <div class="grid grid-cols-2 gap-3">
          <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
            <span class="text-xs font-medium text-slate-400">管控设备总数</span>
            <div class="text-2xl font-black text-white mt-1 font-mono">{{ store.totalCount }} <span class="text-xs font-normal text-slate-500">台</span></div>
          </div>
          <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between">
            <span class="text-xs font-medium text-slate-400">设备在线率</span>
            <div class="text-2xl font-black text-emerald-400 mt-1 font-mono">
              {{ store.onlineRate }}%
            </div>
          </div>
          <div class="glass-panel-danger p-3.5 rounded-2xl flex flex-col justify-between border-red-500/40">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-red-300">紧急救援 (SOS/跌倒)</span>
              <ShieldAlert :size="16" class="text-red-400 animate-bounce" />
            </div>
            <div class="text-2xl font-black text-red-400 mt-1 font-mono">{{ store.criticalAlarmsCount }} <span class="text-xs font-normal text-red-300">起</span></div>
          </div>
          <div class="glass-panel p-3.5 rounded-2xl flex flex-col justify-between border-amber-500/30">
            <div class="flex items-center justify-between">
              <span class="text-xs font-bold text-amber-300">防护预警 (体征/围栏)</span>
              <Activity :size="16" class="text-amber-400" />
            </div>
            <div class="text-2xl font-black text-amber-400 mt-1 font-mono">{{ store.warningAlarmsCount }} <span class="text-xs font-normal text-amber-300">起</span></div>
          </div>
        </div>

        <!-- 设备列表与过滤面板 -->
        <div class="glass-panel rounded-2xl p-4 flex-1 flex flex-col min-h-[500px]">
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-bold text-slate-200 flex items-center space-x-2">
              <span>辖区设备列表</span>
              <span class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono">{{ filteredDevices().length }}</span>
            </h3>
            <span class="text-[10px] text-slate-500 font-mono">全港 18 区覆盖</span>
          </div>

          <!-- 搜索与状态 Filter -->
          <div class="space-y-2 mb-3">
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

          <!-- 设备滚动长列表 -->
          <div class="space-y-2 flex-1 overflow-y-auto max-h-[480px] pr-1">
            <div 
              v-for="dev in filteredDevices()" 
              :key="dev.imei"
              @click="store.selectDevice(dev.imei)"
              :class="['p-3 rounded-xl border transition cursor-pointer flex items-center justify-between',
                store.selectedImei === dev.imei ? 'bg-cyan-950/40 border-cyan-500/60 shadow-lg shadow-cyan-500/10' :
                dev.status === 'sos_alert' ? 'bg-red-950/20 border-red-500/40 hover:bg-red-900/20' : 'bg-slate-900/60 border-slate-800/80 hover:bg-slate-800/60']"
            >
              <div class="flex items-center space-x-3">
                <div :class="['w-2.5 h-2.5 rounded-full flex-shrink-0',
                  dev.status === 'sos_alert' ? 'bg-red-500 animate-ping' :
                  dev.status === 'online' ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-slate-600']"
                ></div>
                <div>
                  <div class="text-xs font-bold text-slate-100 flex items-center space-x-1.5">
                    <span>{{ dev.owner_name }}</span>
                    <span v-if="dev.status === 'sos_alert'" class="text-[9px] bg-red-600 text-white px-1 rounded font-mono animate-pulse">SOS</span>
                  </div>
                  <div class="text-[10px] text-slate-400 font-mono mt-0.5 truncate max-w-[140px]">{{ dev.address }}</div>
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

      <!-- ===== 中栏 (6 列): 高德地图大屏与选中设备控制 ===== -->
      <div class="col-span-12 lg:col-span-6 flex flex-col space-y-4">
        <!-- 高德地图组件 (地图数据与分类 Tab 同步联动过滤，@reset-view 清空选中状态) -->
        <div class="flex-1 min-h-[480px]">
          <AmapDashboard 
            :devices="filteredDevices()" 
            :selectedImei="store.selectedImei"
            @select-device="store.selectDevice"
            @reset-view="store.clearSelection"
          />
        </div>

        <!-- 当前选中设备卡片与控制下发盘 (只在选中某设备时展示，默认首页空选不挤占空间) -->
        <div v-if="store.selectedDevice" class="glass-panel rounded-2xl p-4 border border-slate-800 flex items-center justify-between">
          <div class="flex items-center space-x-4">
            <div class="w-12 h-12 rounded-2xl bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400 font-extrabold text-lg">
              {{ store.selectedDevice.owner_name.slice(0, 1) }}
            </div>
            <div>
              <div class="flex items-center space-x-2">
                <span class="font-bold text-slate-100 text-sm">{{ store.selectedDevice.owner_name }}</span>
                <span :class="['px-2 py-0.5 rounded-full text-[10px] font-bold',
                  store.selectedDevice.status === 'sos_alert' ? 'bg-red-500/20 text-red-400 border border-red-500/30' :
                  store.selectedDevice.status === 'online' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' :
                  'bg-slate-800 text-slate-400']"
                >
                  {{ store.selectedDevice.status === 'sos_alert' ? 'SOS 告警中' : store.selectedDevice.status === 'online' ? '在线' : '离线' }}
                </span>
              </div>
              <div class="text-xs text-slate-400 flex items-center space-x-3 mt-1 font-mono">
                <span class="flex items-center space-x-1"><Heart :size="12" class="text-rose-400" /> <span>{{ store.selectedDevice.last_heart_rate || '--' }} bpm</span></span>
                <span class="flex items-center space-x-1"><Battery :size="12" class="text-amber-400" /> <span>{{ store.selectedDevice.battery }}%</span></span>
                <span class="px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-[10px] text-cyan-300 font-bold">
                  {{ formatLocationStatus(store.selectedDevice.updated_at) }}
                </span>
                <span class="flex items-center space-x-1"><MapPin :size="12" class="text-cyan-400" /> <span class="truncate max-w-[160px]">{{ store.selectedDevice.address }}</span></span>
              </div>
            </div>
          </div>

          <!-- 指令与详情跳转按钮组 -->
          <div class="flex items-center space-x-2">
            <button 
              @click="sendFindCommand(store.selectedDevice.imei)" 
              :disabled="commandStatus === 'sending'"
              class="px-3.5 py-2 bg-cyan-600 hover:bg-cyan-500 text-white rounded-xl text-xs font-bold flex items-center space-x-1.5 shadow-lg shadow-cyan-600/20 active:scale-95 transition disabled:opacity-50"
            >
              <Send :size="14" />
              <span>下发响铃</span>
            </button>

            <!-- 引入跳转至设备全景健康档案详情页的按钮 -->
            <RouterLink 
              :to="'/device/' + store.selectedDevice.imei"
              class="px-3.5 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-bold flex items-center space-x-1.5 shadow-lg shadow-indigo-600/20 active:scale-95 transition"
            >
              <span>查看健康档案详情</span>
              <ExternalLink :size="14" />
            </RouterLink>
          </div>
        </div>
      </div>

      <!-- ===== 右栏 (3 列): 告警中心与工单分流处理看板 (方案二：Filter Tab + 异色 Badge) ===== -->
      <div class="col-span-12 lg:col-span-3 flex flex-col space-y-4">
        <div class="glass-panel rounded-2xl p-4 flex-1 flex flex-col border-slate-800">
          <!-- 告警中心 Header -->
          <div class="flex items-center justify-between mb-3 border-b border-slate-800 pb-3">
            <h3 class="text-sm font-extrabold text-slate-100 flex items-center space-x-2">
              <Bell :size="18" class="text-cyan-400 animate-pulse" />
              <span>实时告警控制中心</span>
            </h3>
            <span class="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-cyan-400 font-mono font-bold">
              {{ store.alarmOrders.length }} 起记录
            </span>
          </div>

          <!-- 告警分类 Filter Tab 快捷切页 -->
          <div class="flex items-center space-x-1 bg-slate-900 p-1 rounded-xl border border-slate-800 text-[10px] mb-3 overflow-x-auto">
            <button 
              @click="alarmCategoryFilter = 'all'" 
              :class="['px-2 py-1 rounded-lg font-bold transition flex-shrink-0', alarmCategoryFilter === 'all' ? 'bg-cyan-600 text-white shadow' : 'text-slate-400 hover:text-white']"
            >
              全部({{ store.alarmOrders.length }})
            </button>
            <button 
              @click="alarmCategoryFilter = 'SOS'" 
              :class="['px-2 py-1 rounded-lg font-bold transition flex-shrink-0', alarmCategoryFilter === 'SOS' ? 'bg-red-600 text-white shadow' : 'text-red-400 hover:text-white']"
            >
              🆘 SOS({{ store.alarmOrders.filter(a => a.category === 'SOS').length }})
            </button>
            <button 
              @click="alarmCategoryFilter = 'FALL'" 
              :class="['px-2 py-1 rounded-lg font-bold transition flex-shrink-0', alarmCategoryFilter === 'FALL' ? 'bg-orange-600 text-white shadow' : 'text-orange-400 hover:text-white']"
            >
              🤸 跌倒({{ store.alarmOrders.filter(a => a.category === 'FALL').length }})
            </button>
            <button 
              @click="alarmCategoryFilter = 'VITAL'" 
              :class="['px-2 py-1 rounded-lg font-bold transition flex-shrink-0', alarmCategoryFilter === 'VITAL' ? 'bg-pink-600 text-white shadow' : 'text-pink-400 hover:text-white']"
            >
              ❤️ 体征({{ store.alarmOrders.filter(a => a.category === 'VITAL').length }})
            </button>
            <button 
              @click="alarmCategoryFilter = 'GEOFENCE'" 
              :class="['px-2 py-1 rounded-lg font-bold transition flex-shrink-0', alarmCategoryFilter === 'GEOFENCE' ? 'bg-cyan-700 text-white shadow' : 'text-cyan-400 hover:text-white']"
            >
              ⭕ 围栏({{ store.alarmOrders.filter(a => a.category === 'GEOFENCE').length }})
            </button>
          </div>

          <!-- 告警单列表 (渲染异色 Badge 标签卡片) -->
          <div class="space-y-3 flex-1 overflow-y-auto max-h-[500px] pr-1">
            <div 
              v-for="alarm in filteredAlarmOrders()" 
              :key="alarm.id"
              :class="['rounded-xl p-3 space-y-2 border transition',
                alarm.category === 'SOS' ? 'bg-red-950/40 border-red-500/50 hover:border-red-400' :
                alarm.category === 'FALL' ? 'bg-orange-950/40 border-orange-500/50 hover:border-orange-400' :
                alarm.category === 'VITAL' ? 'bg-pink-950/30 border-pink-500/40 hover:border-pink-300' :
                'bg-cyan-950/30 border-cyan-500/40 hover:border-cyan-300']"
            >
              <div class="flex items-center justify-between">
                <span :class="['text-[11px] font-extrabold px-2 py-0.5 rounded-lg border flex items-center space-x-1 font-mono',
                  alarm.category === 'SOS' ? 'bg-red-500/20 text-red-300 border-red-500/40 animate-pulse' :
                  alarm.category === 'FALL' ? 'bg-orange-500/20 text-orange-300 border-orange-500/40' :
                  alarm.category === 'VITAL' ? 'bg-pink-500/20 text-pink-300 border-pink-500/40' :
                  'bg-cyan-500/20 text-cyan-300 border-cyan-500/40']"
                >
                  <ShieldAlert v-if="alarm.category === 'SOS' || alarm.category === 'FALL'" :size="12" />
                  <Activity v-else :size="12" />
                  <span>{{ alarm.alert_type }}</span>
                </span>
                <span class="text-[10px] text-slate-400 font-mono">{{ alarm.trigger_time }}</span>
              </div>

              <div class="text-xs text-slate-300 space-y-0.5 font-mono">
                <div>长者设备: <span class="font-bold text-white">{{ store.devices.find(d => d.imei === alarm.device_imei)?.owner_name || alarm.device_imei }}</span></div>
                <div>体征指标: <span :class="alarm.category === 'VITAL' || alarm.category === 'SOS' ? 'text-rose-400 font-bold' : 'text-slate-300'">{{ alarm.heart_rate }} bpm</span></div>
                <div>告警坐标: <span class="text-slate-400">{{ alarm.latitude.toFixed(4) }}, {{ alarm.longitude.toFixed(4) }}</span></div>
              </div>

              <div class="pt-1 flex items-center space-x-2">
                <button 
                  @click="store.selectDevice(alarm.device_imei)"
                  :class="['flex-1 py-1.5 rounded-lg text-xs font-bold transition flex items-center justify-center space-x-1 shadow-lg',
                    alarm.category === 'SOS' ? 'bg-red-600 hover:bg-red-500 text-white shadow-red-600/30' :
                    alarm.category === 'FALL' ? 'bg-orange-600 hover:bg-orange-500 text-white shadow-orange-600/30' :
                    alarm.category === 'VITAL' ? 'bg-pink-600 hover:bg-pink-500 text-white shadow-pink-600/30' :
                    'bg-cyan-700 hover:bg-cyan-600 text-white shadow-cyan-700/30']"
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
