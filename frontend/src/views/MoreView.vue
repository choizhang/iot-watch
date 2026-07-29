<script setup lang="ts">
import { ChevronLeft, MessageCircle, Clock, Book, AlarmClock, Monitor, Search, Footprints, ShieldAlert } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <!-- Header -->
    <header class="bg-white px-4 py-4 flex items-center justify-between shadow-sm sticky top-0 z-10">
      <button @click="router.back()" class="text-slate-800">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="font-bold text-slate-800 text-lg">更多功能</h1>
      <div class="w-6"></div> <!-- Spacer for centering -->
    </header>

    <!-- Grid Container -->
    <div class="p-4 grid grid-cols-2 gap-4">
      <router-link to="/chat" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <MessageCircle :size="24" />
        </div>
        <span class="font-bold text-slate-700">微聊</span>
      </router-link>

      <router-link to="/interval" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <Clock :size="24" />
        </div>
        <span class="font-bold text-slate-700 text-sm">间隔设定</span>
      </router-link>

      <router-link to="/phonebook" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <Book :size="24" />
        </div>
        <span class="font-bold text-slate-700">电话薄</span>
      </router-link>

      <router-link to="/reminder" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <AlarmClock :size="24" />
        </div>
        <span class="font-bold text-slate-700 text-sm">提醒设定</span>
      </router-link>

      <router-link to="/params" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <Monitor :size="24" />
        </div>
        <span class="font-bold text-slate-700 text-sm">装置参数</span>
      </router-link>

      <router-link to="/find" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <Search :size="24" />
        </div>
        <span class="font-bold text-slate-700 text-sm">寻找装置</span>
      </router-link>

      <router-link to="/sport-reminder" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4">
        <div class="w-12 h-12 bg-blue-50 rounded-2xl flex items-center justify-center text-blue-500">
          <Footprints :size="24" />
        </div>
        <span class="font-bold text-slate-700 text-sm">运动提醒</span>
      </router-link>

      <router-link to="/geofence" class="bg-white p-6 rounded-3xl shadow-sm flex items-center space-x-4 border-2 border-emerald-100 hover:border-emerald-300 transition">
        <div class="w-12 h-12 bg-emerald-50 rounded-2xl flex items-center justify-center text-emerald-500">
          <ShieldAlert :size="24" />
        </div>
        <div class="flex flex-col">
          <span class="font-bold text-slate-800 text-sm">电子围栏</span>
          <span class="text-[10px] text-slate-400">出界/入界告警</span>
        </div>
      </router-link>
    </div>
  </div>
</template>
