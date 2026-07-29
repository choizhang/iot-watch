<script setup lang="ts">
import { onMounted, onUnmounted, watch, nextTick, ref, computed } from 'vue';
import AMapLoader from '@amap/amap-jsapi-loader';

// 高德地图安全密钥配置 (请替换为您自己的 Key)
const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659';
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca';

(window as any)._AMapSecurityConfig = {
  securityJsCode: AMAP_SECURITY_CODE,
};

const props = defineProps<{
  latitude?: number;
  longitude?: number;
  zoom?: number;
}>();

const emit = defineEmits<{
  (e: 'address-resolved', address: string): void;
}>();

const loading = ref(true);
const loadError = ref(false);
let map: any = null;
let marker: any = null;
let geocoder: any = null;

// 计算经纬度是否有效 (0 / NaN / 越界 都视为无效)
const isValidCoord = (lat: any, lng: any) => {
  const nlat = Number(lat);
  const nlng = Number(lng);
  return !isNaN(nlat) && !isNaN(nlng) && nlat !== 0 && nlng !== 0 && nlat < 90 && nlat > -90 && nlng < 180 && nlng > -180;
};

// 是否拥有有效坐标（控制 Marker 是否渲染）
const hasValidPosition = computed(() => isValidCoord(props.latitude, props.longitude));

// Marker 渲染器
const renderMarker = (pos: [number, number]) => {
  const AMap = (window as any).AMap;
  if (!AMap || !map) return;
  
  // 清除旧的 marker
  if (marker) {
    map.remove(marker);
    marker = null;
  }
  
  // 高德原生点标记
  marker = new AMap.Marker({
    position: pos,
    map: map,
    icon: new AMap.Icon({
      size: new AMap.Size(30, 40),
      image: 'data:image/svg+xml;base64,' + btoa(`
        <svg xmlns="http://www.w3.org/2000/svg" width="30" height="40" viewBox="0 0 30 40">
          <defs>
            <filter id="shadow" x="-50%" y="-50%" width="200%" height="200%">
              <feDropShadow dx="0" dy="2" stdDeviation="2" flood-color="#000" flood-opacity="0.3"/>
            </filter>
          </defs>
          <path filter="url(#shadow)" d="M15 0 C6.7 0 0 6.7 0 15 c0 10.5 15 25 15 25 s15 -14.5 15 -25 C30 6.7 23.3 0 15 0z" fill="#3b82f6"/>
          <circle cx="15" cy="15" r="6" fill="white"/>
        </svg>
      `),
      imageSize: new AMap.Size(30, 40),
      imageOffset: new AMap.Pixel(0, 0)
    }),
    offset: new AMap.Pixel(-15, -40),
    zIndex: 100,
    title: '当前位置'
  });
};

const initMap = async () => {
  await nextTick();
  loading.value = true;
  loadError.value = false;
  
  try {
    const AMap = await AMapLoader.load({
      key: AMAP_KEY,
      version: '2.0',
      plugins: ['AMap.Geocoder', 'AMap.Marker', 'AMap.Icon', 'AMap.Size', 'AMap.Pixel', 'AMap.Circle'],
    });

    // 地图初始中心：如果有有效坐标则用真实坐标，否则用空地图（无中心）
    const hasPos = isValidCoord(props.latitude, props.longitude);
    const center: [number, number] | undefined = hasPos
      ? [Number(props.longitude), Number(props.latitude)]
      : undefined;

    console.log('[Amap] 初始化，是否有有效坐标:', hasPos, '中心:', center);

    map = new AMap.Map('amap-container', {
      viewMode: '2D',
      zoom: props.zoom || (hasPos ? 16 : 11),
      center: center,
      dragEnable: true,
      zoomEnable: true,
      resizeEnable: true,
    });

    map.on('error', (err: any) => {
      console.error('地图运行错误:', err);
    });

    geocoder = new AMap.Geocoder({ city: '全国' });

    // 仅在有有效坐标时才渲染 Marker
    if (hasPos && center) {
      renderMarker(center);
      resolveAddress();
    } else {
      console.log('[Amap] 无有效坐标，不渲染 Marker');
    }
    
    loading.value = false;
  } catch (e) {
    console.error('高德地图加载失败', e);
    loading.value = false;
    loadError.value = true;
  }
};

const resolveAddress = () => {
  if (!geocoder) return;
  if (!isValidCoord(props.latitude, props.longitude)) return;
  
  const lat = Number(props.latitude);
  const lng = Number(props.longitude);
  
  geocoder.getAddress([lng, lat], (status: string, result: any) => {
    if (status === 'complete' && result.regeocode) {
      emit('address-resolved', result.regeocode.formattedAddress);
    }
  });
};

watch(() => [props.latitude, props.longitude], ([newLat, newLng]) => {
  const hasPos = isValidCoord(newLat, newLng);

  if (map && hasPos) {
    const pos: [number, number] = [Number(newLng), Number(newLat)];
    console.log('[Amap] 更新位置:', pos);
    
    // 平滑移动中心
    map.setCenter(pos, false, 500);
    
    if (marker) {
      marker.setPosition(pos);
    } else {
      renderMarker(pos);
    }
    resolveAddress();
  } else if (map && marker) {
    // 坐标失效，移除 Marker
    map.remove(marker);
    marker = null;
  }
}, { immediate: true });

onMounted(() => {
  initMap();
});

onUnmounted(() => {
  if (map) {
    map.destroy();
    map = null;
    marker = null;
  }
});
</script>

<template>
  <div class="relative w-full h-full bg-slate-100">
    <div id="amap-container" class="w-full h-full"></div>
    
    <div v-if="loading" class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-white/80 backdrop-blur-sm">
      <div class="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mb-3"></div>
      <span class="text-xs text-slate-500">正在加载地图...</span>
    </div>

    <div v-if="loadError" class="absolute inset-0 z-20 bg-white flex flex-col items-center justify-center p-4">
      <span class="text-red-500 font-bold mb-2">地图加载失败</span>
      <span class="text-[10px] text-slate-400 text-center max-w-[200px] mb-3">请检查高德 Key 是否正确</span>
      <button @click="initMap" class="text-xs bg-blue-500 text-white px-4 py-1.5 rounded-lg">重试</button>
    </div>
  </div>
</template>

<style scoped>
#amap-container {
  width: 100%;
  height: 100%;
  min-height: 200px;
  display: block;
}
</style>
