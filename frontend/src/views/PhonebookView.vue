<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ChevronLeft, Plus, Trash2, Phone, User, CheckCircle, AlertTriangle } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { useDeviceStore } from '../store/device'

const router = useRouter()
const store = useDeviceStore()

const showAddDialog = ref(false)
const newContact = ref({ name: '', phone: '', relation: '家属' })
const toast = ref({ show: false, message: '' })
const confirmDialog = ref({ show: false, title: '', message: '', onConfirm: () => {} })

const showToast = (msg: string) => {
  toast.value.message = msg
  toast.value.show = true
  setTimeout(() => {
    toast.value.show = false
  }, 2000)
}

const addContact = async () => {
  if (!newContact.value.name || !newContact.value.phone) return
  await store.addContact(newContact.value.name, newContact.value.phone, newContact.value.relation)
  newContact.value = { name: '', phone: '', relation: '家属' }
  showAddDialog.value = false
  showToast('联系人添加成功')
}

const removeContact = (id: number, name: string) => {
  confirmDialog.value = {
    show: true,
    title: '确认删除',
    message: `您确定要删除联系人“${name}”吗？此操作无法撤销。`,
    onConfirm: async () => {
      confirmDialog.value.show = false
      await store.removeContact(id)
      showToast('联系人已删除')
    }
  }
}

onMounted(() => {
  store.fetchContacts()
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 relative pb-16">
    <!-- Header -->
    <div class="bg-white px-4 py-4 flex items-center shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="mr-3 text-slate-600">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="text-lg font-bold text-slate-800">电话薄</h1>
    </div>

    <!-- Contact List -->
    <div class="p-4 space-y-3">
      <div v-for="contact in store.contacts" :key="contact.id" class="bg-white rounded-2xl p-4 flex items-center shadow-sm">
        <div class="w-12 h-12 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white mr-3">
          <User :size="20" />
        </div>
        <div class="flex-1">
          <div class="flex items-center space-x-2">
            <h3 class="font-bold text-slate-800">{{ contact.name }}</h3>
            <span class="text-[10px] bg-blue-50 text-blue-500 px-2 py-0.5 rounded">{{ contact.relation }}</span>
          </div>
          <div class="flex items-center text-slate-500 text-sm mt-1">
            <Phone :size="12" class="mr-1" />
            <span>{{ contact.phone }}</span>
          </div>
        </div>
        <button @click="removeContact(contact.id, contact.name)" class="text-red-400 p-2 active:scale-90 transition-transform">
          <Trash2 :size="18" />
        </button>
      </div>

      <!-- Add Button -->
      <button @click="showAddDialog = true" class="w-full bg-white border-2 border-dashed border-blue-200 rounded-2xl p-4 flex items-center justify-center text-blue-500 font-bold active:scale-98 transition-transform">
        <Plus :size="20" class="mr-2" />
        添加联系人
      </button>
    </div>

    <!-- Add Dialog (Drawer, elevated z-index to cover bottom nav) -->
    <div v-if="showAddDialog" class="fixed inset-0 z-[100] bg-black/40 flex items-end" @click.self="showAddDialog = false">
      <div class="w-full bg-white rounded-t-3xl p-6 space-y-4 pb-12 shadow-2xl">
        <h3 class="text-lg font-bold text-slate-800">新增联系人</h3>
        <input v-model="newContact.name" placeholder="姓名" class="w-full p-3 bg-slate-50 rounded-xl outline-none" />
        <input v-model="newContact.phone" placeholder="手机号" class="w-full p-3 bg-slate-50 rounded-xl outline-none" />
        <select v-model="newContact.relation" class="w-full p-3 bg-slate-50 rounded-xl outline-none">
          <option>家属</option>
          <option>医疗</option>
          <option>社区</option>
        </select>
        <div class="flex gap-3">
          <button @click="showAddDialog = false" class="flex-1 p-3 bg-slate-100 rounded-xl text-slate-600">取消</button>
          <button @click="addContact" class="flex-1 p-3 bg-blue-500 rounded-xl text-white font-bold">保存</button>
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
