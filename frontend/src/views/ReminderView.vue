<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ChevronLeft, Plus, AlarmClock, Trash2, Bell, CheckCircle, AlertTriangle } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useDeviceStore } from '../store/device'

const router = useRouter()
const store = useDeviceStore()

const showAdd = ref(false)
const newReminder = ref({ time: '08:00', label: '' })
const toast = ref({ show: false, message: '' })
const confirmDialog = ref({ show: false, title: '', message: '', onConfirm: () => {} })

const showToast = (msg: string) => {
  toast.value.message = msg
  toast.value.show = true
  setTimeout(() => {
    toast.value.show = false
  }, 2000)
}

const addReminder = async () => {
  if (!newReminder.value.label) return
  await store.addReminder(newReminder.value.time, newReminder.value.label)
  newReminder.value = { time: '08:00', label: '' }
  showAdd.value = false
  showToast('提醒保存成功')
}

const removeReminder = (id: number, label: string) => {
  confirmDialog.value = {
    show: true,
    title: '确认删除',
    message: `您确定要删除“${label}”的提醒吗？此操作无法撤销。`,
    onConfirm: async () => {
      confirmDialog.value.show = false
      await store.removeReminder(id)
      showToast('提醒已删除')
    }
  }
}

onMounted(() => {
  store.fetchReminders()
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 relative pb-16">
    <div class="bg-white px-4 py-4 flex items-center shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="mr-3 text-slate-600">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="text-lg font-bold text-slate-800">提醒设定</h1>
    </div>

    <div class="p-4 space-y-3">
      <div v-for="r in store.reminders" :key="r.id" class="bg-white rounded-2xl p-4 flex items-center shadow-sm">
        <div class="w-12 h-12 rounded-full bg-blue-50 flex items-center justify-center text-blue-500 mr-3">
          <AlarmClock :size="20" />
        </div>
        <div class="flex-1">
          <div class="flex items-center space-x-2">
            <span class="text-xl font-bold text-slate-800">{{ r.time }}</span>
            <span class="text-slate-500">·</span>
            <span class="text-slate-600 text-sm">{{ r.label }}</span>
          </div>
        </div>
        <button @click="removeReminder(r.id, r.label)" class="text-red-400 p-2 active:scale-90 transition-transform">
          <Trash2 :size="18" />
        </button>
      </div>

      <button @click="showAdd = true" class="w-full bg-white border-2 border-dashed border-blue-200 rounded-2xl p-4 flex items-center justify-center text-blue-500 font-bold active:scale-98 transition-transform">
        <Plus :size="20" class="mr-2" />
        新增提醒
      </button>
    </div>

    <!-- Add Dialog (Drawer, elevated z-index to cover bottom nav) -->
    <div v-if="showAdd" class="fixed inset-0 z-[100] bg-black/40 flex items-end" @click.self="showAdd = false">
      <div class="w-full bg-white rounded-t-3xl p-6 space-y-4 pb-12 shadow-2xl">
        <h3 class="text-lg font-bold text-slate-800">新增提醒</h3>
        <div class="flex items-center space-x-3">
          <Bell :size="20" class="text-blue-500" />
          <input v-model="newReminder.time" type="time" class="flex-1 p-3 bg-slate-50 rounded-xl outline-none" />
        </div>
        <input v-model="newReminder.label" placeholder="提醒内容 (如: 吃药)" class="w-full p-3 bg-slate-50 rounded-xl outline-none" />
        <div class="flex gap-3">
          <button @click="showAdd = false" class="flex-1 p-3 bg-slate-100 rounded-xl text-slate-600">取消</button>
          <button @click="addReminder" class="flex-1 p-3 bg-blue-500 rounded-xl text-white font-bold">保存</button>
        </div>
      </div>
    </div>

    <!-- Confirm Dialog (z-[100] to sit above nav) -->
    <div v-if="confirmDialog.show" class="fixed inset-0 z-[100] bg-black/40 flex items-center justify-center p-6" @click.self="confirmDialog.show = false">
      <div class="bg-white rounded-3xl p-6 w-full max-w-sm space-y-4 shadow-2xl text-center">
        <div class="w-12 h-12 bg-red-50 text-red-500 rounded-full flex items-center justify-center mx-auto">
          <AlertTriangle :size="24" />
        </div>
        <div class="space-y-1">
          <h3 class="text-base font-bold text-slate-800">{{ confirmDialog.title }}</h3>
          <p class="text-xs text-slate-500 leading-relaxed">{{ confirmDialog.message }}</p>
        </div>
        <div class="flex gap-3 pt-2">
          <button @click="confirmDialog.show = false" class="flex-1 p-3 bg-slate-100 rounded-xl text-slate-600 text-xs font-bold">取消</button>
          <button @click="confirmDialog.onConfirm" class="flex-1 p-3 bg-red-500 rounded-xl text-white text-xs font-bold shadow-lg shadow-red-100">确认删除</button>
        </div>
      </div>
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
