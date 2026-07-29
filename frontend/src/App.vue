<script setup lang="ts">
import { Home, Sparkles, User } from 'lucide-vue-next'
</script>

<template>
  <router-view />

  <!-- Bottom Navigation -->
  <nav class="fixed bottom-0 left-0 right-0 bg-white border-t border-slate-100 flex items-center justify-around py-2 z-50">
    <router-link to="/" class="flex flex-col items-center space-y-1" active-class="text-blue-500">
      <Home :size="24" />
      <span class="text-[10px] font-bold">首頁</span>
    </router-link>
    <div class="flex flex-col items-center space-y-1 text-slate-400">
      <Sparkles :size="24" />
      <span class="text-[10px] font-bold">AI 助手</span>
    </div>
    <div class="flex flex-col items-center space-y-1 text-slate-400">
      <User :size="24" />
      <span class="text-[10px] font-bold">我的</span>
    </div>
  </nav>
</template>

<style>
.router-link-active {
  color: #3b82f6;
}
</style>
