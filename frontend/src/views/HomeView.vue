<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useDeviceStore } from '../store/device'
import AmapContainer from '../components/AmapContainer.vue'
import { 
  Settings, 
  Bell, 
  MapPin, 
  Navigation, 
  History, 
  Grid, 
  MessageCircle, 
  Clock, 
  Book, 
  AlarmClock, 
  MoreHorizontal,
  ChevronDown,
  ChevronRight,
  LocateFixed,
  Heart,
  ShieldAlert,
  Droplets,
  Thermometer,
  Footprints,
  Moon,
  Activity
} from 'lucide-vue-next'

const store = useDeviceStore()
const resolvedAddress = ref('')

// IMEI 切换器
const showImeiSwitcher = ref(false)
const customImei = ref('')
const availableImeis = ref([
  '1234567890',
  '13800138000',
  '867123456789012',
  '359000000000017',
  '868811000000015',
  '352099001761481',
])

const switchImei = (imei: string) => {
  if (!imei) return
  store.setImei(imei)
  showImeiSwitcher.value = false
  customImei.value = ''
}

const onAddressResolved = (address: string) => {
  resolvedAddress.value = address
}

const centerMap = () => {
  if (store.status?.last_latitude && store.status?.last_longitude) {
    // 重新触发定位居中（通过改变 key 或者调用组件方法，这里简单处理，store 数据变化会自动触发 watch）
    console.log('执行立即定位，当前坐标:', [store.status.last_longitude, store.status.last_latitude])
    // 强制触发一次 store 的更新感知
    const lat = store.status.last_latitude
    store.status.last_latitude = 0
    setTimeout(() => {
      store.status!.last_latitude = lat
    }, 10)
  }
}

onMounted(() => {
  store.fetchStatus()
  store.fetchHealth()
})

const formatTime = (ts: number | string) => {
  if (!ts) return '--'
  const date = typeof ts === 'number' ? new Date(ts * 1000) : new Date(ts)
  
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}
</script>

<template>
  <div class="min-h-screen bg-slate-50 pb-20">
    <!-- Header -->
    <header class="bg-white px-4 py-3 flex items-center justify-between shadow-sm sticky top-0 z-10">
      <div class="flex items-center space-x-3">
        <div class="w-10 h-10 bg-slate-100 rounded-full flex items-center justify-center overflow-hidden">
          <img src="https://coresg-normal.trae.ai/api/ide/v1/text_to_image?prompt=smart%20watch%20for%20elderly%20modern%20design&image_size=square" alt="watch" class="w-full h-full object-cover" />
        </div>
        <div>
          <div class="flex items-center space-x-1 cursor-pointer" @click="showImeiSwitcher = !showImeiSwitcher">
            <span class="font-bold text-slate-800">{{ store.currentImei.slice(-6) }}</span>
            <ChevronDown :size="16" class="text-slate-500" />
          </div>
          <div class="flex items-center space-x-1">
            <div :class="['w-2 h-2 rounded-full', store.status?.status === 'online' ? 'bg-green-500' : 'bg-gray-400']"></div>
            <span class="text-xs text-slate-500">{{ store.status?.status === 'online' ? '在线' : '离线' }}</span>
            <span v-if="store.status?.status === 'online' && store.status?.battery !== undefined" class="text-xs text-slate-500 ml-1">· 电量 {{ store.status.battery }}%</span>
          </div>
        </div>
      </div>
      <div class="flex items-center space-x-3 text-slate-600">
        <!-- Mock Switch Toggle -->
        <button 
          @click="store.setUseMock(!store.useMock)" 
          :class="['px-2 py-1 rounded-full text-[10px] font-bold tracking-wider transition-all duration-300 flex items-center space-x-1 border shadow-sm', 
            store.useMock 
              ? 'bg-amber-500/10 text-amber-600 border-amber-500/20 hover:bg-amber-500/20' 
              : 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20 hover:bg-emerald-500/20']"
        >
          <span class="w-1.5 h-1.5 rounded-full" :class="store.useMock ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500'"></span>
          <span>{{ store.useMock ? 'MOCK 数据' : '实时数据' }}</span>
        </button>
        <Settings :size="24" />
        <Bell :size="24" />
      </div>
    </header>

    <!-- IMEI 切换面板 -->
    <div v-if="showImeiSwitcher" class="fixed top-16 left-4 right-4 bg-white rounded-2xl shadow-2xl border border-slate-100 z-50 p-4">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-sm font-bold text-slate-700">切换设备 IMEI</h3>
        <button @click="showImeiSwitcher = false" class="text-slate-400 text-xs">关闭</button>
      </div>
      <div class="grid grid-cols-2 gap-2">
        <button
          v-for="imei in availableImeis"
          :key="imei"
          @click="switchImei(imei)"
          :class="['p-2 rounded-lg text-xs font-mono transition-colors',
            store.currentImei === imei
              ? 'bg-blue-500 text-white'
              : 'bg-slate-50 text-slate-600 hover:bg-slate-100']"
        >
          {{ imei }}
        </button>
      </div>
      <div class="mt-3 flex items-center space-x-2">
        <input
          v-model="customImei"
          placeholder="输入 IMEI"
          class="flex-1 px-3 py-2 border border-slate-200 rounded-lg text-xs"
        />
        <button @click="switchImei(customImei)" class="px-3 py-2 bg-blue-500 text-white rounded-lg text-xs">确定</button>
      </div>
    </div>

    <!-- Map Section -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-3xl overflow-hidden shadow-md border border-slate-100">
        <!-- Map Container -->
        <div class="h-96 w-full relative bg-slate-100 overflow-hidden">
          <AmapContainer 
            :latitude="store.status?.last_latitude" 
            :longitude="store.status?.last_longitude"
            @address-resolved="onAddressResolved"
          />
        </div>
        
        <!-- Address & Quick Actions Section -->
        <div class="p-5">
          <div class="flex justify-between items-start mb-6">
            <div class="flex-1 pr-4">
              <h3 class="font-bold text-slate-800 text-base leading-snug mb-2" data-testid="address-display">
                {{ resolvedAddress || store.status?.address || (store.status?.last_latitude ? `香港特别行政区 (${store.status.last_latitude.toFixed(4)}, ${store.status.last_longitude.toFixed(4)})` : '正在获取位置...') }}
              </h3>
              <div class="flex items-center space-x-3 text-xs text-slate-400">
                <span class="bg-blue-50 text-blue-500 px-2 py-0.5 rounded-lg uppercase font-bold tracking-tight">{{ store.status?.location_type || 'GPS' }}</span>
                <div class="flex items-center space-x-1.5">
                  <Clock :size="12" />
                  <span>更新: {{ formatTime(store.status?.updated_at || '') }}</span>
                </div>
              </div>
            </div>
            <button class="w-12 h-12 bg-blue-500 rounded-2xl flex items-center justify-center text-white shadow-blue-200 shadow-xl active:scale-95 transition-transform">
              <Navigation :size="24" />
            </button>
          </div>

          <!-- Quick Actions -->
          <div class="grid grid-cols-3 gap-4 pt-4 border-t border-slate-50">
            <div @click="centerMap" class="flex flex-col items-center space-y-2 cursor-pointer group">
              <div class="w-10 h-10 flex items-center justify-center text-blue-500 bg-blue-50 rounded-full group-active:scale-90 transition-transform">
                <LocateFixed :size="22" />
              </div>
              <span class="text-xs font-bold text-slate-700">立即定位</span>
            </div>
            <div class="flex flex-col items-center space-y-2 cursor-pointer group">
              <div class="w-10 h-10 flex items-center justify-center text-blue-500 bg-blue-50 rounded-full group-active:scale-90 transition-transform">
                <History :size="22" />
              </div>
              <span class="text-xs font-bold text-slate-700">历史足迹</span>
            </div>
            <router-link to="/geofence" class="flex flex-col items-center space-y-2 cursor-pointer group">
              <div class="w-10 h-10 flex items-center justify-center text-emerald-600 bg-emerald-50 rounded-full group-active:scale-90 transition-transform shadow-sm">
                <ShieldAlert :size="22" />
              </div>
              <span class="text-xs font-bold text-slate-700">电子围栏</span>
            </router-link>
          </div>
        </div>
      </div>
    </div>

    <!-- Feature Grid -->
    <div class="px-4 mt-6 grid grid-cols-5 gap-2">
      <router-link to="/phonebook" class="flex flex-col items-center space-y-2">
        <div class="w-14 h-14 bg-white rounded-2xl flex items-center justify-center text-blue-500 shadow-sm border border-slate-100">
          <Book :size="24" />
        </div>
        <span class="text-[10px] text-slate-500 font-bold">电话薄</span>
      </router-link>
      <router-link to="/interval" class="flex flex-col items-center space-y-2">
        <div class="w-14 h-14 bg-white rounded-2xl flex items-center justify-center text-blue-400 shadow-sm border border-slate-100">
          <Clock :size="24" />
        </div>
        <span class="text-[10px] text-slate-500 font-bold">间隔设定</span>
      </router-link>
      <router-link to="/reminder" class="flex flex-col items-center space-y-2">
        <div class="w-14 h-14 bg-white rounded-2xl flex items-center justify-center text-blue-500 shadow-sm border border-slate-100">
          <AlarmClock :size="24" />
        </div>
        <span class="text-[10px] text-slate-500 font-bold">提醒设定</span>
      </router-link>
      <router-link to="/params" class="flex flex-col items-center space-y-2">
        <div class="w-14 h-14 bg-white rounded-2xl flex items-center justify-center text-blue-400 shadow-sm border border-slate-100">
          <Settings :size="24" />
        </div>
        <span class="text-[10px] text-slate-500 font-bold">装置参数</span>
      </router-link>
      <router-link to="/find" class="flex flex-col items-center space-y-2">
        <div class="w-14 h-14 bg-white rounded-2xl flex items-center justify-center text-blue-300 shadow-sm border border-slate-100">
          <MoreHorizontal :size="24" />
        </div>
        <span class="text-[10px] text-slate-500 font-bold">寻找装置</span>
      </router-link>
    </div>

    <!-- Health Section -->
    <div class="px-4 mt-8">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-bold text-slate-500 text-sm">健康指标</h2>
      </div>
      
      <div class="grid grid-cols-2 gap-4">
        <!-- Blood Pressure -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-blue-500"><Heart :size="20" /></div>
            <span class="text-sm font-bold text-blue-500">血压</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.blood_pressure || '120/75' }}</span>
            <span class="text-xs text-slate-400">mmHg</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>

        <!-- Heart Rate -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100 cursor-pointer hover:border-red-200 transition-colors" @click="$router.push('/heart-rate')">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-red-500"><Heart :size="20" /></div>
            <span class="text-sm font-bold text-red-500">心率</span>
            <div class="ml-auto"><ChevronRight :size="14" class="text-slate-300" /></div>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800" data-testid="heart-rate-value">{{ store.status?.last_heart_rate || store.health?.heart_rate || '77' }}</span>
            <span class="text-xs text-slate-400">bmp</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>

        <!-- Blood Oxygen -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-blue-400"><Droplets :size="20" /></div>
            <span class="text-sm font-bold text-blue-500">血氧</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.blood_oxygen || '99' }}</span>
            <span class="text-xs text-slate-400">%</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>

        <!-- Temperature -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-orange-500"><Thermometer :size="20" /></div>
            <span class="text-sm font-bold text-orange-500">体温</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.temperature || '36.7' }}</span>
            <span class="text-xs text-slate-400">℃</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>

        <!-- Steps -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-green-500"><Footprints :size="20" /></div>
            <span class="text-sm font-bold text-green-500">运动</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.steps || '5415' }}</span>
            <span class="text-xs text-slate-400">步</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>

        <!-- Sleep -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-purple-500"><Moon :size="20" /></div>
            <span class="text-sm font-bold text-purple-500">睡眠</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.sleep || '--' }}</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ store.health?.updated_at ? formatTime(store.health.updated_at) : '暂无测量数据' }}</span>
        </div>

        <!-- HRV -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-slate-400"><Activity :size="20" /></div>
            <span class="text-sm font-bold text-slate-400 uppercase">HRV</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.hrv || '24' }}</span>
            <span class="text-xs text-slate-400">ms</span>
          </div>
          <span class="text-[10px] text-slate-300">{{ formatTime(store.health?.updated_at || '') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
