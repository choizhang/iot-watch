<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { ChevronLeft, Clock, Check, CheckCircle } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useDeviceStore } from '../store/device'

const router = useRouter()
const store = useDeviceStore()

const intervals = [
  { label: '1 分钟', value: 60, desc: '耗电最高' },
  { label: '5 分钟', value: 300, desc: '推荐' },
  { label: '10 分钟', value: 600, desc: '均衡' },
  { label: '30 分钟', value: 1800, desc: '省电' },
  { label: '1 小时', value: 3600, desc: '最省电' },
]

const currentInterval = ref(300)
const toast = ref({ show: false, message: '' })

const showToast = (msg: string) => {
  toast.value.message = msg
  toast.value.show = true
  setTimeout(() => {
    toast.value.show = false
  }, 1500)
}

const saveSettings = async () => {
  await store.updateSettings(currentInterval.value)
  showToast('设置保存成功')
  setTimeout(() => {
    router.back()
  }, 1500)
}

onMounted(async () => {
  await store.fetchSettings()
  currentInterval.value = store.settings?.interval || 300
})

watch(() => store.settings?.interval, (val) => {
  if (val) {
    currentInterval.value = val
  }
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 relative">
    <div class="bg-white px-4 py-4 flex items-center shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="mr-3 text-slate-600">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="text-lg font-bold text-slate-800">定位间隔设定</h1>
    </div>

    <div class="p-4">
      <div class="bg-blue-50 rounded-2xl p-4 mb-4 flex items-start space-x-3">
        <Clock :size="20" class="text-blue-500 mt-0.5" />
        <div class="text-xs text-slate-600">
          定位间隔决定了设备多久向服务器上报一次位置。间隔越短越精准，但耗电越快。
        </div>
      </div>

      <div class="space-y-3">
        <div 
          v-for="item in intervals" 
          :key="item.value"
          @click="currentInterval = item.value"
          class="bg-white rounded-2xl p-4 flex items-center justify-between cursor-pointer"
          :class="currentInterval === item.value ? 'ring-2 ring-blue-500' : ''"
        >
          <div>
            <div class="font-bold text-slate-800">{{ item.label }}</div>
            <div class="text-xs text-slate-400 mt-1">{{ item.desc }}</div>
          </div>
          <div v-if="currentInterval === item.value" class="w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center">
            <Check :size="14" class="text-white" />
          </div>
          <div v-else class="w-6 h-6 border-2 border-slate-200 rounded-full"></div>
        </div>
      </div>

      <button @click="saveSettings" class="w-full mt-6 p-4 bg-blue-500 text-white font-bold rounded-2xl shadow-lg active:scale-95 transition">
        保存设置
      </button>
    </div>

    <!-- Toast Notification -->
    <Transition name="fade">
      <div v-if="toast.show" class="fixed top-24 left-1/2 -translate-x-1/2 bg-slate-800 text-white px-4 py-2.5 rounded-full text-xs font-bold shadow-2xl flex items-center space-x-2 z-[999] transition-all">
        <CheckCircle :size="14" class="text-green-400" />
        <span>{{ toast.message }}</span>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
