<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { getAMapInstance } from './utils/amap'

onMounted(() => {
  // 应用启动时后台异步预加载高德 SDK & 插件 (全港区域预热)
  getAMapInstance().catch(err => {
    console.warn('高德 SDK 预加载提示:', err)
  })
})
</script>

<template>
  <RouterView v-slot="{ Component }">
    <KeepAlive :max="10">
      <component :is="Component" />
    </KeepAlive>
  </RouterView>
</template>
