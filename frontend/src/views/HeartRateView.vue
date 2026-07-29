<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, Heart, TrendingUp, TrendingDown, Clock, Activity } from 'lucide-vue-next'
import { useDeviceStore } from '../store/device'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import axios from 'axios'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

interface HeartRatePoint {
  time: string
  heart_rate: number
  battery?: number
  latitude?: number
  longitude?: number
}

const route = useRoute()
const imei = route.params.imei as string || localStorage.getItem('current_imei') || '1234567890'
const store = useDeviceStore()

const points = ref<HeartRatePoint[]>([])
const loading = ref(true)
const selectedHours = ref(24)

const stats = computed(() => {
  if (!points.value.length) return { min: 0, max: 0, avg: 0, count: 0 }
  const hrs = points.value.map(p => p.heart_rate)
  return {
    min: Math.min(...hrs),
    max: Math.max(...hrs),
    avg: Math.round(hrs.reduce((a, b) => a + b, 0) / hrs.length),
    count: hrs.length
  }
})

const chartData = computed(() => {
  const labels = points.value.map(p => {
    const d = new Date(p.time)
    return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
  })
  const data = points.value.map(p => p.heart_rate)

  return {
    labels,
    datasets: [{
      label: '心率 (bpm)',
      data,
      borderColor: '#ef4444',
      backgroundColor: 'rgba(239, 68, 68, 0.08)',
      borderWidth: 2,
      pointRadius: points.value.length > 50 ? 0 : 3,
      pointBackgroundColor: '#ef4444',
      pointBorderColor: '#fff',
      pointBorderWidth: 1,
      fill: true,
      tension: 0.4,
    }]
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: '#1e293b',
      titleColor: '#94a3b8',
      bodyColor: '#fff',
      padding: 10,
      callbacks: {
        title: (items: any[]) => {
          if (!items[0]) return ''
          return items[0].label
        },
        label: (item: any) => ` 心率: ${item.raw} bpm`
      }
    }
  },
  scales: {
    x: {
      grid: { color: 'rgba(0,0,0,0.04)' },
      ticks: {
        maxTicksLimit: 8,
        color: '#94a3b8',
        font: { size: 10 }
      }
    },
    y: {
      min: 40,
      max: 140,
      grid: { color: 'rgba(0,0,0,0.04)' },
      ticks: {
        stepSize: 20,
        color: '#94a3b8',
        font: { size: 10 }
      }
    }
  }
}

const fetchHistory = async (hours: number) => {
  loading.value = true
  selectedHours.value = hours
  try {
    if (store.useMock) {
      const mockPoints = []
      const now = Date.now()
      const count = hours * 12 // 12 points per hour (every 5m)
      for (let i = count; i >= 0; i--) {
        const timeVal = new Date(now - i * 5 * 60 * 1000)
        const hr = 70 + Math.floor(Math.sin(i / 10) * 12) + Math.floor(Math.random() * 6)
        mockPoints.push({
          time: timeVal.toISOString(),
          heart_rate: hr,
          battery: 88 - Math.floor((count - i) / 50),
          latitude: 22.396428 + Math.sin(i / 100) * 0.005,
          longitude: 114.109497 + Math.cos(i / 100) * 0.005
        })
      }
      points.value = mockPoints
      return
    }

    const res = await axios.get(`http://localhost:8080/api/v1/device/${imei}/heart-rate/history?hours=${hours}`)
    points.value = res.data.points || []
  } catch (e) {
    console.error('获取心率历史失败', e)
    points.value = []
  } finally {
    loading.value = false
  }
}

watch(() => store.useMock, () => {
  fetchHistory(selectedHours.value)
})

onMounted(() => {
  fetchHistory(24)
})
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <!-- Header -->
    <header class="bg-white px-4 py-3 flex items-center justify-between shadow-sm sticky top-0 z-10">
      <div class="flex items-center space-x-3">
        <button @click="$router.back()" class="text-slate-600">
          <ArrowLeft :size="22" />
        </button>
        <div class="flex items-center space-x-2">
          <div class="text-red-500"><Heart :size="20" /></div>
          <span class="font-bold text-slate-800">心率详情</span>
        </div>
      </div>
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
    </header>

    <!-- Time Range Tabs -->
    <div class="bg-white px-4 py-3 flex items-center justify-between border-b border-slate-100">
      <span class="text-xs text-slate-500">IMEI: {{ imei }}</span>
      <div class="flex items-center space-x-1 bg-slate-100 rounded-lg p-1">
        <button
          v-for="h in [6, 12, 24, 72]"
          :key="h"
          @click="fetchHistory(h)"
          :class="['px-3 py-1 rounded-md text-xs font-medium transition-colors',
            selectedHours === h ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']"
        >
          {{ h < 24 ? `${h}h` : `${h/24}天` }}
        </button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="px-4 mt-4 grid grid-cols-4 gap-3">
      <div class="bg-white rounded-2xl p-4 text-center shadow-sm border border-slate-100">
        <div class="text-xs text-slate-400 mb-1">最高</div>
        <div class="text-xl font-bold text-red-500">{{ stats.max }}</div>
        <div class="text-[10px] text-slate-300 mt-1 flex items-center justify-center space-x-0.5">
          <TrendingUp :size="10" /> <span>bpm</span>
        </div>
      </div>
      <div class="bg-white rounded-2xl p-4 text-center shadow-sm border border-slate-100">
        <div class="text-xs text-slate-400 mb-1">最低</div>
        <div class="text-xl font-bold text-blue-500">{{ stats.min }}</div>
        <div class="text-[10px] text-slate-300 mt-1 flex items-center justify-center space-x-0.5">
          <TrendingDown :size="10" /> <span>bpm</span>
        </div>
      </div>
      <div class="bg-white rounded-2xl p-4 text-center shadow-sm border border-slate-100">
        <div class="text-xs text-slate-400 mb-1">平均</div>
        <div class="text-xl font-bold text-slate-700">{{ stats.avg }}</div>
        <div class="text-[10px] text-slate-300 mt-1 flex items-center justify-center space-x-0.5">
          <Activity :size="10" /> <span>bpm</span>
        </div>
      </div>
      <div class="bg-white rounded-2xl p-4 text-center shadow-sm border border-slate-100">
        <div class="text-xs text-slate-400 mb-1">记录</div>
        <div class="text-xl font-bold text-slate-700">{{ stats.count }}</div>
        <div class="text-[10px] text-slate-300 mt-1 flex items-center justify-center space-x-0.5">
          <Clock :size="10" /> <span>条</span>
        </div>
      </div>
    </div>

    <!-- Heart Rate Chart -->
    <div class="px-4 mt-4">
      <div class="bg-white rounded-2xl p-4 shadow-sm border border-slate-100">
        <h3 class="text-sm font-bold text-slate-700 mb-4">心率趋势</h3>
        
        <div v-if="loading" class="h-64 flex items-center justify-center">
          <div class="w-8 h-8 border-2 border-red-500 border-t-transparent rounded-full animate-spin"></div>
        </div>
        
        <div v-else-if="!points.length" class="h-64 flex flex-col items-center justify-center text-slate-400">
          <Heart :size="40" class="mb-3 opacity-30" />
          <span class="text-sm">暂无心率数据</span>
          <span class="text-xs mt-1">请确保设备已上报过定位/心跳数据</span>
        </div>

        <div v-else class="h-64">
          <Line :data="chartData" :options="chartOptions" />
        </div>
      </div>
    </div>

    <!-- Data List -->
    <div v-if="!loading && points.length > 0" class="px-4 mt-4 pb-8">
      <div class="bg-white rounded-2xl shadow-sm border border-slate-100 overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center justify-between">
          <h3 class="text-sm font-bold text-slate-700">数据明细</h3>
          <span class="text-xs text-slate-400">共 {{ points.length }} 条</span>
        </div>
        
        <div class="max-h-80 overflow-y-auto">
          <div
            v-for="(p, i) in points.slice().reverse().slice(0, 50)"
            :key="i"
            class="px-4 py-3 flex items-center justify-between border-b border-slate-50 last:border-0"
          >
            <div class="flex items-center space-x-3">
              <div :class="['w-2 h-2 rounded-full',
                p.heart_rate > 100 ? 'bg-red-500' : p.heart_rate < 60 ? 'bg-blue-400' : 'bg-green-500']"></div>
              <div>
                <div class="text-sm font-bold text-slate-800">{{ p.heart_rate }} <span class="text-xs text-slate-400 font-normal">bpm</span></div>
                <div class="text-[10px] text-slate-400">{{ new Date(p.time).toLocaleString() }}</div>
              </div>
            </div>
            <div :class="['text-xs px-2 py-0.5 rounded-full',
              p.heart_rate > 100 ? 'bg-red-50 text-red-500' :
              p.heart_rate < 60 ? 'bg-blue-50 text-blue-500' : 'bg-green-50 text-green-600']">
              {{ p.heart_rate > 100 ? '偏快' : p.heart_rate < 60 ? '偏慢' : '正常' }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
