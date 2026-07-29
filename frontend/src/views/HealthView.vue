<script setup lang="ts">
import { onMounted } from 'vue'
import { useDeviceStore } from '../store/device'
import { 
  Heart, 
  Droplets, 
  Thermometer, 
  Footprints, 
  Moon, 
  Activity,
  ChevronDown,
  Settings,
  Bell
} from 'lucide-vue-next'

const store = useDeviceStore()

onMounted(() => {
  store.fetchHealth()
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 pb-20">
    <!-- Header (Same as Home) -->
    <header class="bg-white px-4 py-3 flex items-center justify-between shadow-sm sticky top-0 z-10">
      <div class="flex items-center space-x-3">
        <div class="w-10 h-10 bg-slate-100 rounded-full flex items-center justify-center overflow-hidden">
          <img src="https://coresg-normal.trae.ai/api/ide/v1/text_to_image?prompt=smart%20watch%20for%20elderly%20modern%20design&image_size=square" alt="watch" class="w-full h-full object-cover" />
        </div>
        <div>
          <div class="flex items-center space-x-1">
            <span class="font-bold text-slate-800">CC</span>
            <ChevronDown :size="16" class="text-slate-500" />
          </div>
          <div class="flex items-center space-x-1">
            <div :class="['w-2 h-2 rounded-full', store.status?.status === 'online' ? 'bg-green-500' : 'bg-gray-400']"></div>
            <span class="text-xs text-slate-500">{{ store.status?.status === 'online' ? '在线' : '离线' }}</span>
          </div>
        </div>
      </div>
      <div class="flex items-center space-x-4 text-slate-600">
        <Settings :size="24" />
        <Bell :size="24" />
      </div>
    </header>

    <!-- Sub Nav -->
    <div class="bg-white flex px-4 py-4 space-x-8 border-t border-slate-50">
      <span class="text-slate-400 font-medium">微聊</span>
      <span class="text-slate-400 font-medium">间隔设定</span>
      <span class="text-slate-400 font-medium">电话薄</span>
      <span class="text-slate-400 font-medium">提醒设定</span>
      <span class="text-slate-800 font-bold border-b-2 border-blue-500 pb-1">更多</span>
    </div>

    <div class="p-4">
      <h2 class="text-slate-500 font-bold text-sm mb-4">健康指标</h2>

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
          <span class="text-[10px] text-slate-300">2026-06-11 16:13:54</span>
        </div>

        <!-- Heart Rate -->
        <div class="bg-white p-5 rounded-3xl shadow-sm border border-slate-100">
          <div class="flex items-center space-x-2 mb-3">
            <div class="text-red-500"><Heart :size="20" /></div>
            <span class="text-sm font-bold text-red-500">心率</span>
          </div>
          <div class="flex items-baseline space-x-1 mb-1">
            <span class="text-2xl font-bold text-slate-800">{{ store.health?.heart_rate || '77' }}</span>
            <span class="text-xs text-slate-400">bmp</span>
          </div>
          <span class="text-[10px] text-slate-300">2026-06-11 16:10:59</span>
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
          <span class="text-[10px] text-slate-300">2026-06-11 16:11:35</span>
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
          <span class="text-[10px] text-slate-300">2026-06-11 16:14:29</span>
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
          <span class="text-[10px] text-slate-300">2026-06-11 16:09:28</span>
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
          <span class="text-[10px] text-slate-300">暂无测量数据</span>
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
          <span class="text-[10px] text-slate-300">2026-06-11 16:13:51</span>
        </div>
      </div>
    </div>
  </div>
</template>
