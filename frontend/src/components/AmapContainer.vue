<script setup lang="ts">
import { onMounted, onUnmounted, watch, nextTick, ref, computed } from 'vue';
import { Plus, Minus, Crosshair, MapPin } from 'lucide-vue-next';

/*
// ------------------------------------------------------------------
// 原高德地图 (AMap) 逻辑（暂时注释保留）
// ------------------------------------------------------------------
import AMapLoader from '@amap/amap-jsapi-loader';
const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659';
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca';
(window as any)._AMapSecurityConfig = {
  securityJsCode: AMAP_SECURITY_CODE,
};
*/

const GOOGLE_MAPS_KEY = 'AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw';

const props = defineProps<{
  latitude?: number;
  longitude?: number;
  zoom?: number;
  status?: string;
  accuracy?: number;
}>();

const emit = defineEmits<{
  (e: 'address-resolved', address: string): void;
}>();

const loading = ref(true);
const loadError = ref(false);
let map: any = null;
let marker: any = null;
let accuracyCircle: any = null;
let geocoder: any = null;
let googleObj: any = null;

// 计算经纬度是否有效 (0 / NaN / 越界 都视为无效)
const isValidCoord = (lat: any, lng: any) => {
  const nlat = Number(lat);
  const nlng = Number(lng);
  return !isNaN(nlat) && !isNaN(nlng) && nlat !== 0 && nlng !== 0 && nlat < 90 && nlat > -90 && nlng < 180 && nlng > -180;
};

// 是否拥有有效坐标（控制 Marker 是否渲染）
const hasValidPosition = computed(() => isValidCoord(props.latitude, props.longitude));

// 谷歌地图异步单例加载方法
const loadGoogleMaps = () => {
  return new Promise<any>((resolve, reject) => {
    if ((window as any).google && (window as any).google.maps) {
      return resolve((window as any).google.maps);
    }
    const scriptId = 'google-maps-script';
    if (document.getElementById(scriptId)) {
      let interval = setInterval(() => {
        if ((window as any).google && (window as any).google.maps) {
          clearInterval(interval);
          resolve((window as any).google.maps);
        }
      }, 100);
      return;
    }
    const script = document.createElement('script');
    script.id = scriptId;
    script.src = `https://maps.googleapis.com/maps/api/js?key=${GOOGLE_MAPS_KEY}&language=zh-CN`;
    script.async = true;
    script.defer = true;
    script.onload = () => {
      if ((window as any).google && (window as any).google.maps) {
        resolve((window as any).google.maps);
      } else {
        reject(new Error('Google Maps script loaded but google.maps unavailable'));
      }
    };
    script.onerror = (err) => reject(err);
    document.head.appendChild(script);
  });
};

// Marker 渲染器
const renderMarker = (pos: { lat: number; lng: number }) => {
  if (!googleObj || !map) return;
  
  if (marker) {
    marker.setMap(null);
    marker = null;
  }
  if (accuracyCircle) {
    accuracyCircle.setMap(null);
    accuracyCircle = null;
  }

  const isSOS = props.status === 'sos_alert';
  const isOffline = props.status === 'offline';

  // 离线冷灰 / 告警警示红 / 在线翡翠绿
  const iconColor = isSOS ? '#ef4444' : isOffline ? '#64748b' : '#10b981';
  const strapColor = isSOS ? '#dc2626' : isOffline ? '#475569' : '#059669';
  const iconBgFill = isSOS ? '#fee2e2' : isOffline ? '#f1f5f9' : '#ecfdf5';

  const accuracyMeter = props.accuracy || 18.5;

  // 绘制 1:1 地理物理半径定位误差范围圈
  accuracyCircle = new googleObj.Circle({
    map: map,
    center: pos,
    radius: accuracyMeter,
    strokeColor: iconColor,
    strokeOpacity: 0.8,
    strokeWeight: 1.5,
    fillColor: iconColor,
    fillOpacity: 0.08,
    clickable: false,
  });
  
  marker = new googleObj.Marker({
    position: pos,
    map: map,
    title: '手环当前位置',
    icon: {
      url: 'data:image/svg+xml;base64,' + btoa(`
        <svg xmlns="http://www.w3.org/2000/svg" width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="${iconColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 1h6v5H9z" fill="${strapColor}" stroke="${iconColor}" stroke-width="1.2"/>
          <path d="M9 18h6v5H9z" fill="${strapColor}" stroke="${iconColor}" stroke-width="1.2"/>
          <rect x="6" y="6" width="12" height="12" rx="3.5" fill="${iconBgFill}" stroke="${iconColor}" stroke-width="2"/>
          <circle cx="12" cy="12" r="2" fill="none" stroke="${iconColor}" stroke-width="1.5" />
          <path d="M18.5 10v4" stroke="${iconColor}" stroke-width="2" stroke-linecap="round" />
        </svg>
      `),
      scaledSize: new googleObj.Size(34, 34),
      anchor: new googleObj.Point(17, 17)
    }
  });
};

const initMap = async () => {
  await nextTick();
  loading.value = true;
  loadError.value = false;
  
  try {
    googleObj = await loadGoogleMaps();

    const hasPos = isValidCoord(props.latitude, props.longitude);
    const center = hasPos
      ? { lat: Number(props.latitude), lng: Number(props.longitude) }
      : { lat: 30.658633, lng: 104.064718 };

    const mapEl = document.getElementById('amap-container');
    if (!mapEl) return;

    map = new googleObj.Map(mapEl, {
      zoom: props.zoom || (hasPos ? 16 : 11),
      center: center,
      disableDefaultUI: true,
      zoomControl: false,
    });

    geocoder = new googleObj.Geocoder();

    if (hasPos && center) {
      renderMarker(center);
      resolveAddress();
    }
    
    loading.value = false;
  } catch (e) {
    console.error('谷歌地图 (Google Maps) 加载失败', e);
    loading.value = false;
    loadError.value = true;
  }
};

const resolveAddress = () => {
  if (!geocoder) return;
  if (!isValidCoord(props.latitude, props.longitude)) return;
  
  const lat = Number(props.latitude);
  const lng = Number(props.longitude);
  
  geocoder.geocode({ location: { lat, lng } }, (results: any, status: string) => {
    if (status === 'OK' && results && results[0]) {
      emit('address-resolved', results[0].formatted_address);
    }
  });
};

watch(() => [props.latitude, props.longitude], ([newLat, newLng]) => {
  const hasPos = isValidCoord(newLat, newLng);

  if (map && hasPos) {
    const pos = { lat: Number(newLat), lng: Number(newLng) };
    map.panTo(pos);
    
    if (marker) {
      marker.setPosition(pos);
    } else {
      renderMarker(pos);
    }
    resolveAddress();
  } else if (map && marker) {
    marker.setMap(null);
    marker = null;
  }
}, { immediate: true });

let userLocationMarker: any = null;

const zoomIn = () => {
  if (map) map.setZoom(map.getZoom() + 1);
};

const zoomOut = () => {
  if (map) map.setZoom(map.getZoom() - 1);
};

const centerOnDevice = () => {
  if (map && hasValidPosition.value) {
    map.panTo({ lat: Number(props.latitude), lng: Number(props.longitude) });
    map.setZoom(17);
  }
};

const locateUserPosition = () => {
  if (!navigator.geolocation) {
    centerOnDevice();
    return;
  }
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      const uLat = pos.coords.latitude;
      const uLng = pos.coords.longitude;
      if (map && googleObj) {
        map.panTo({ lat: uLat, lng: uLng });
        map.setZoom(16);

        if (userLocationMarker) userLocationMarker.setMap(null);
        userLocationMarker = new googleObj.Marker({
          position: { lat: uLat, lng: uLng },
          map: map,
          title: '我的当前位置',
          icon: {
            path: googleObj.SymbolPath.CIRCLE,
            scale: 8,
            fillColor: '#3b82f6',
            fillOpacity: 1,
            strokeColor: '#ffffff',
            strokeWeight: 2,
          }
        });
      }
    },
    (err) => {
      console.warn('获取我的当前定位失败:', err);
      centerOnDevice();
    },
    { enableHighAccuracy: true, timeout: 5000 }
  );
};

onMounted(() => {
  initMap();
});

onUnmounted(() => {
  if (userLocationMarker) {
    userLocationMarker.setMap(null);
    userLocationMarker = null;
  }
  if (map) {
    map = null;
    marker = null;
  }
});
</script>

<template>
  <div class="relative w-full h-full bg-slate-100">
    <div id="amap-container" class="w-full h-full"></div>
    
    <!-- H5 统一竖向地图控件组（避开底部与放大缩小冲突） -->
    <div class="absolute bottom-20 right-4 flex flex-col bg-white border border-slate-200/90 rounded-2xl shadow-xl z-20 overflow-hidden text-slate-700 font-sans">
      <button 
        @click="zoomIn"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-100 hover:bg-slate-50 text-slate-700 active:bg-slate-100 transition"
        title="放大"
      >
        <Plus :size="18" />
      </button>
      <button 
        @click="zoomOut"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-100 hover:bg-slate-50 text-slate-700 active:bg-slate-100 transition"
        title="缩小"
      >
        <Minus :size="18" />
      </button>
      <button 
        @click="locateUserPosition"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-100 hover:bg-slate-50 text-blue-600 active:bg-blue-50 transition"
        title="我的当前位置"
      >
        <Crosshair :size="18" />
      </button>
      <button 
        @click="centerOnDevice"
        class="w-10 h-10 flex items-center justify-center hover:bg-slate-50 text-emerald-600 active:bg-emerald-50 transition"
        title="聚焦设备位置"
      >
        <MapPin :size="18" />
      </button>
    </div>

    <div v-if="loading" class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-white/80 backdrop-blur-sm">
      <div class="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mb-3"></div>
      <span class="text-xs text-slate-500">正在加载谷歌地图 (Google Maps)...</span>
    </div>

    <div v-if="loadError" class="absolute inset-0 z-20 bg-white flex flex-col items-center justify-center p-4">
      <span class="text-red-500 font-bold mb-2">谷歌地图 (Google Maps) 加载失败</span>
      <span class="text-[10px] text-slate-400 text-center max-w-[200px] mb-3">请检查谷歌 API Key 是否有效</span>
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
