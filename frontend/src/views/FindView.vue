<script setup lang="ts">
import { ref } from 'vue'
import { ChevronLeft, Search, Volume2, Vibrate, CheckCircle, Loader } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

import { useDeviceStore } from '../store/device'

const router = useRouter()
const store = useDeviceStore()

const status = ref<'idle' | 'searching' | 'found'>('idle')
const countdown = ref(0)
const errMsg = ref('')

const startSearch = async () => {
  errMsg.value = ''
  try {
    await store.sendCommand('FIND')
  } catch (err: any) {
    errMsg.value = err.message || '指令下发失败'
  }

  status.value = 'searching'
  countdown.value = 30
  
  const timer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(timer)
      status.value = 'found'
    }
  }, 1000)
}
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <div class="bg-white px-4 py-4 flex items-center shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="mr-3 text-slate-600">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="text-lg font-bold text-slate-800">寻找装置</h1>
    </div>

    <div class="p-6 flex flex-col items-center justify-center min-h-[60vh]">
      <!-- Status Display -->
      <div v-if="status === 'idle'" class="text-center">
        <div class="w-32 h-32 mx-auto bg-blue-50 rounded-full flex items-center justify-center mb-6">
          <Search :size="64" class="text-blue-500" />
        </div>
        <h2 class="text-xl font-bold text-slate-800 mb-2">找不到手环了？</h2>
        <p class="text-sm text-slate-400 mb-8">点击下方按钮让手环发出声音和震动</p>
        <button @click="startSearch" class="px-12 py-4 bg-blue-500 text-white font-bold rounded-2xl shadow-lg shadow-blue-200 active:scale-95 transition">
          开始寻找
        </button>
      </div>

      <div v-else-if="status === 'searching'" class="text-center">
        <div class="w-32 h-32 mx-auto bg-blue-500 rounded-full flex items-center justify-center mb-6 animate-pulse">
          <Loader :size="64" class="text-white animate-spin" />
        </div>
        <h2 class="text-xl font-bold text-slate-800 mb-2">正在搜索设备...</h2>
        <p class="text-sm text-slate-400">剩余 {{ countdown }} 秒</p>
        <div class="mt-6 flex items-center justify-center space-x-6">
          <div class="flex flex-col items-center">
            <Volume2 :size="28" class="text-blue-500" />
            <span class="text-xs text-slate-400 mt-1">正在响铃</span>
          </div>
          <div class="flex flex-col items-center">
            <Vibrate :size="28" class="text-blue-500" />
            <span class="text-xs text-slate-400 mt-1">正在震动</span>
          </div>
        </div>
      </div>

      <div v-else class="text-center">
        <div class="w-32 h-32 mx-auto bg-green-50 rounded-full flex items-center justify-center mb-6">
          <CheckCircle :size="64" class="text-green-500" />
        </div>
        <h2 class="text-xl font-bold text-slate-800 mb-2">设备已响应</h2>
        <p class="text-sm text-slate-400 mb-8">手环发出了声音和震动信号</p>
        <button @click="status = 'idle'" class="px-12 py-3 bg-slate-100 text-slate-600 font-bold rounded-2xl">
          重新寻找
        </button>
      </div>
    </div>
  </div>
</template>
