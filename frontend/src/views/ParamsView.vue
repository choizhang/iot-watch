<script setup lang="ts">
import { ref } from 'vue'
import { ChevronLeft, Cpu, Wifi, Battery, Save, RefreshCw } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

const router = useRouter()

const deviceInfo = ref({
  imei: '1234567890',
  model: 'ElderGuard Pro 2026',
  firmware: 'v2.3.1',
  signal: 85,
  battery: 78,
  workMode: 'normal',
})

const workModes = [
  { value: 'normal', label: '普通模式', desc: '平衡性能与耗电' },
  { value: 'power_save', label: '省电模式', desc: '降低上报频率' },
  { value: 'sport', label: '运动模式', desc: '高频 GPS 定位' },
]
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <div class="bg-white px-4 py-4 flex items-center shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="mr-3 text-slate-600">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="text-lg font-bold text-slate-800">装置参数</h1>
    </div>

    <div class="p-4 space-y-4">
      <!-- Device Info Card -->
      <div class="bg-gradient-to-br from-slate-700 to-slate-900 rounded-2xl p-5 text-white">
        <div class="flex items-center space-x-3 mb-4">
          <div class="w-12 h-12 bg-white/10 rounded-xl flex items-center justify-center">
            <Cpu :size="24" />
          </div>
          <div>
            <div class="font-bold">{{ deviceInfo.model }}</div>
            <div class="text-xs text-slate-400">IMEI: {{ deviceInfo.imei }}</div>
          </div>
        </div>
        <div class="grid grid-cols-3 gap-2 mt-4">
          <div class="bg-white/10 rounded-xl p-2 text-center">
            <Wifi :size="16" class="mx-auto mb-1 text-green-300" />
            <div class="text-xs text-slate-300">信号</div>
            <div class="font-bold text-sm">{{ deviceInfo.signal }}%</div>
          </div>
          <div class="bg-white/10 rounded-xl p-2 text-center">
            <Battery :size="16" class="mx-auto mb-1 text-yellow-300" />
            <div class="text-xs text-slate-300">电量</div>
            <div class="font-bold text-sm">{{ deviceInfo.battery }}%</div>
          </div>
          <div class="bg-white/10 rounded-xl p-2 text-center">
            <RefreshCw :size="16" class="mx-auto mb-1 text-blue-300" />
            <div class="text-xs text-slate-300">固件</div>
            <div class="font-bold text-sm">{{ deviceInfo.firmware }}</div>
          </div>
        </div>
      </div>

      <!-- Work Mode -->
      <div class="bg-white rounded-2xl p-4">
        <h3 class="font-bold text-slate-800 mb-3 text-sm">工作模式</h3>
        <div class="space-y-2">
          <div 
            v-for="m in workModes" 
            :key="m.value"
            @click="deviceInfo.workMode = m.value"
            class="p-3 rounded-xl border-2 cursor-pointer"
            :class="deviceInfo.workMode === m.value ? 'border-blue-500 bg-blue-50' : 'border-slate-100'"
          >
            <div class="font-bold text-slate-800 text-sm">{{ m.label }}</div>
            <div class="text-xs text-slate-400 mt-1">{{ m.desc }}</div>
          </div>
        </div>
      </div>

      <button class="w-full p-4 bg-blue-500 text-white font-bold rounded-2xl shadow-lg active:scale-95 transition flex items-center justify-center">
        <Save :size="18" class="mr-2" />
        保存参数
      </button>
    </div>
  </div>
</template>
