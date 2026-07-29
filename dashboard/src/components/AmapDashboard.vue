<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { getAMapInstance } from '../utils/amap'
import type { DeviceItem } from '../store/dashboard'

export interface GeofenceProps {
  id: number
  name: string
  latitude: number
  longitude: number
  radius: number
  fence_type: string
  enabled: boolean
}

const props = withDefaults(defineProps<{
  devices: DeviceItem[]
  selectedImei: string
  zoomLevel?: number
  showTrack?: boolean
  trackPoints?: [number, number][]
  runnerPoint?: [number, number] | null
  showHeatmap?: boolean
  heatmapPoints?: { lng: number; lat: number; count: number }[]
  heatmapRadius?: number
  geofences?: GeofenceProps[]
}>(), {
  zoomLevel: undefined,
  showTrack: false,
  trackPoints: () => [],
  runnerPoint: null,
  showHeatmap: false,
  heatmapPoints: () => [],
  heatmapRadius: 35,
  geofences: () => []
})

const emit = defineEmits<{
  (e: 'select-device', imei: string): void
  (e: 'reset-view'): void
}>()

const mapContainer = ref<HTMLDivElement | null>(null)
const loading = ref(true)
const loadError = ref(false)
const currentStyle = ref<'dark' | 'normal' | 'satellite'>('dark')

let mapInstance: any = null
let AMapObj: any = null
let geocoder: any = null
let markers: any[] = []
let trackPolyline: any = null
let trackMarkers: any[] = []
let runnerMarker: any = null
let heatmapObj: any = null
let satelliteLayer: any = null
let fenceCircles: any[] = []

// 防御性安全坐标计算
const getValidCoords = (dev: DeviceItem): [number, number] => {
  let lat = Number(dev.last_latitude)
  let lng = Number(dev.last_longitude)

  if (isNaN(lat) || lat <= 0) {
    lat = dev.imei === '1234567890' ? 22.396428 : dev.imei === '868811000000015' ? 22.371234 : 22.381200
  }
  if (isNaN(lng) || lng <= 0) {
    lng = dev.imei === '1234567890' ? 114.109497 : dev.imei === '868811000000015' ? 114.115678 : 114.189100
  }
  return [lng, lat]
}

const changeMapStyle = (style: 'dark' | 'normal' | 'satellite') => {
  currentStyle.value = style
  if (!mapInstance || !AMapObj) return

  if (style === 'satellite') {
    if (!satelliteLayer) {
      satelliteLayer = new AMapObj.TileLayer.Satellite()
    }
    satelliteLayer.setMap(mapInstance)
  } else {
    if (satelliteLayer) {
      satelliteLayer.setMap(null)
    }
    mapInstance.setMapStyle(`amap://styles/${style}`)
  }
}

const resetMapOverview = () => {
  if (!mapInstance) return
  mapInstance.setPitch(0, false, 300)
  const singleDevice = props.devices.length === 1 ? props.devices[0] : null
  const targetCenter: [number, number] = singleDevice ? getValidCoords(singleDevice) : [114.1694, 22.3193]
  const targetZoom = props.zoomLevel ? props.zoomLevel : (singleDevice ? 17 : 11)
  mapInstance.setZoomAndCenter(targetZoom, targetCenter, false, 300)
  emit('reset-view')
}

const initMap = async () => {
  loading.value = true
  loadError.value = false
  try {
    AMapObj = await getAMapInstance()

    if (!mapContainer.value) return

    geocoder = new AMapObj.Geocoder({ city: '香港特别行政区' })

    const singleDevice = props.devices.length === 1 ? props.devices[0] : null
    const initialCenter = singleDevice 
      ? getValidCoords(singleDevice)
      : [114.1694, 22.3193]

    const initialZoom = props.zoomLevel ? props.zoomLevel : (singleDevice ? 17 : 11)

    mapInstance = new AMapObj.Map(mapContainer.value, {
      viewMode: '2D',
      zoom: initialZoom,
      center: initialCenter,
      pitch: 0,
      mapStyle: `amap://styles/${currentStyle.value}`,
    })

    loading.value = false
    renderMarkers()
    renderTrack()
    renderHeatmap()
    renderGeofences()
    renderRunner()
  } catch (err) {
    console.error('高德地图加载失败:', err)
    loading.value = false
    loadError.value = true
  }
}

const renderMarkers = () => {
  if (!mapInstance || !AMapObj) return

  markers.forEach(m => m.setMap(null))
  markers = []

  props.devices.forEach((dev) => {
    const [lng, lat] = getValidCoords(dev)

    const isSOS = dev.status === 'sos_alert'
    const isOnline = dev.status === 'online'
    const isSelected = dev.imei === props.selectedImei

    const markerDom = document.createElement('div')
    markerDom.style.width = '28px'
    markerDom.style.height = '28px'
    markerDom.style.position = 'relative'
    markerDom.style.cursor = 'pointer'
    markerDom.className = 'group flex flex-col items-center justify-center z-20'

    const pinBg = isSOS 
      ? 'bg-red-600 text-white shadow-lg shadow-red-500/60' 
      : isOnline 
      ? 'bg-emerald-500 text-white shadow-lg shadow-emerald-500/50' 
      : 'bg-slate-700 text-slate-300 shadow-md shadow-slate-900/50'

    const borderStyle = isSelected 
      ? 'ring-4 ring-amber-400 ring-offset-2 ring-offset-slate-950 scale-125 z-50' 
      : 'border-2 border-slate-900'

    const svgIcon = isSOS
      ? `<svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 animate-bounce" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg>`
      : isOnline
      ? `<svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="6"/><rect x="9" y="2" width="6" height="4" rx="1"/><rect x="9" y="18" width="6" height="4" rx="1"/></svg>`
      : `<svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5 opacity-60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="1" y1="1" x2="23" y2="23"/><path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/><path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/></svg>`

    markerDom.innerHTML = `
      <div class="absolute bottom-full mb-2 hidden group-hover:flex ${isSelected ? '!flex' : ''} flex-col items-center pointer-events-none z-50 transition-all duration-200">
        <div class="bg-slate-900/95 border border-slate-700/80 text-white px-3 py-1.5 rounded-xl shadow-2xl backdrop-blur-md text-xs font-bold whitespace-nowrap flex items-center space-x-2">
          <span class="w-2 h-2 rounded-full ${isSOS ? 'bg-red-500 animate-ping' : isOnline ? 'bg-emerald-400' : 'bg-slate-400'}"></span>
          <span>${dev.owner_name}</span>
          ${isSOS ? '<span class="bg-red-600 text-white text-[10px] px-1 rounded animate-pulse">SOS</span>' : ''}
        </div>
        <div class="w-2 h-2 bg-slate-900 rotate-45 -mt-1 border-r border-b border-slate-700/80"></div>
      </div>

      <div class="relative w-7 h-7 flex items-center justify-center">
        ${isSOS ? '<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-500 opacity-75"></span>' : ''}
        <div class="w-7 h-7 rounded-full ${pinBg} ${borderStyle} flex items-center justify-center transition-all duration-200 hover:scale-125">
          ${svgIcon}
        </div>
      </div>
    `

    markerDom.addEventListener('click', () => {
      emit('select-device', dev.imei)
    })

    const marker = new AMapObj.Marker({
      position: [lng, lat],
      content: markerDom,
      offset: new AMapObj.Pixel(-14, -14),
      zIndex: isSelected ? 200 : 100,
    })

    marker.on('click', () => {
      emit('select-device', dev.imei)
    })

    marker.setMap(mapInstance)
    markers.push(marker)
  })
}

const renderGeofences = () => {
  if (!mapInstance || !AMapObj) return

  fenceCircles.forEach(c => c.setMap(null))
  fenceCircles = []

  if (props.geofences && props.geofences.length > 0) {
    props.geofences.forEach(fence => {
      if (!fence.enabled) return
      const isOut = fence.fence_type === 'OUT'
      const circle = new AMapObj.Circle({
        center: [fence.longitude, fence.latitude],
        radius: fence.radius,
        strokeColor: isOut ? '#ef4444' : '#0284c7',
        strokeOpacity: 0.85,
        strokeWeight: 2,
        fillColor: isOut ? '#ef4444' : '#0284c7',
        fillOpacity: 0.2,
        strokeStyle: 'dashed'
      })
      circle.setMap(mapInstance)
      fenceCircles.push(circle)
    })
  }
}

const renderTrack = () => {
  if (!mapInstance || !AMapObj) return

  if (trackPolyline) {
    trackPolyline.setMap(null)
    trackPolyline = null
  }
  trackMarkers.forEach(m => m.setMap(null))
  trackMarkers = []

  if (props.showTrack && props.trackPoints && props.trackPoints.length > 1) {
    trackPolyline = new AMapObj.Polyline({
      path: props.trackPoints,
      isOutline: true,
      outlineColor: '#0284c7',
      borderWeight: 2,
      strokeColor: '#38bdf8',
      strokeOpacity: 0.95,
      strokeWeight: 5,
      strokeStyle: 'solid',
      showDir: true,
    })
    trackPolyline.setMap(mapInstance)

    props.trackPoints.forEach((pt, idx) => {
      const isStart = idx === 0
      const isEnd = idx === props.trackPoints.length - 1

      if (isEnd) return

      const pointDom = document.createElement('div')
      pointDom.className = 'flex items-center justify-center'

      if (isStart) {
        pointDom.innerHTML = `<span class="px-2.5 py-1 bg-emerald-600 text-white font-extrabold text-[10px] rounded-full shadow-lg border border-white whitespace-nowrap">轨迹起点</span>`
      } else {
        pointDom.innerHTML = `<span class="w-2.5 h-2.5 bg-cyan-400 border border-slate-900 rounded-full shadow-md"></span>`
      }

      const tMarker = new AMapObj.Marker({
        position: pt,
        content: pointDom,
        offset: isStart ? new AMapObj.Pixel(-25, -12) : new AMapObj.Pixel(-5, -5),
        zIndex: isStart ? 110 : 80
      })
      tMarker.setMap(mapInstance)
      trackMarkers.push(tMarker)
    })

    if (!props.runnerPoint) {
      mapInstance.setFitView([trackPolyline])
    }
  }
}

// 轨迹回放动态移动 Runner Marker
const renderRunner = () => {
  if (!mapInstance || !AMapObj) return

  if (runnerMarker) {
    runnerMarker.setMap(null)
    runnerMarker = null
  }

  if (props.runnerPoint) {
    const runnerDom = document.createElement('div')
    runnerDom.className = 'relative flex items-center justify-center'
    runnerDom.innerHTML = `
      <span class="animate-ping absolute inline-flex h-9 w-9 rounded-full bg-cyan-400 opacity-75"></span>
      <div class="w-7 h-7 rounded-full bg-cyan-500 border-2 border-white shadow-2xl flex items-center justify-center text-white text-sm font-bold">
        🚶
      </div>
    `
    runnerMarker = new AMapObj.Marker({
      position: props.runnerPoint,
      content: runnerDom,
      offset: new AMapObj.Pixel(-14, -14),
      zIndex: 300
    })
    runnerMarker.setMap(mapInstance)
    mapInstance.panTo(props.runnerPoint)
  }
}

const renderHeatmap = () => {
  if (!mapInstance || !AMapObj) return

  if (heatmapObj) {
    heatmapObj.setMap(null)
    heatmapObj = null
  }

  if (props.showHeatmap && props.heatmapPoints && props.heatmapPoints.length > 0) {
    heatmapObj = new AMapObj.HeatMap(mapInstance, {
      radius: props.heatmapRadius || 35,
      opacity: [0, 0.85],
      gradient: {
        0.4: '#3b82f6',
        0.6: '#06b6d4',
        0.8: '#10b981',
        0.9: '#f59e0b',
        1.0: '#ef4444'
      }
    })
    heatmapObj.setDataSet({
      data: props.heatmapPoints,
      max: 100
    })
  }
}

watch(() => props.devices, () => {
  renderMarkers()
}, { deep: true })

watch(() => props.geofences, () => {
  renderGeofences()
}, { deep: true })

watch(() => [props.showTrack, props.trackPoints], () => {
  renderTrack()
}, { deep: true })

watch(() => props.runnerPoint, () => {
  renderRunner()
})

watch(() => [props.showHeatmap, props.heatmapPoints, props.heatmapRadius], () => {
  renderHeatmap()
}, { deep: true })

watch(() => props.selectedImei, (newImei) => {
  const target = props.devices.find(d => d.imei === newImei)
  if (target && mapInstance) {
    const [targetLng, targetLat] = getValidCoords(target)
    const targetZoom = props.zoomLevel ? props.zoomLevel : 15
    mapInstance.setZoomAndCenter(targetZoom, [targetLng, targetLat], false, 200)
  }
  renderMarkers()
})

onMounted(() => {
  initMap()
})

onUnmounted(() => {
  if (mapInstance) {
    mapInstance.destroy()
    mapInstance = null
  }
})
</script>

<template>
  <div class="relative w-full h-full rounded-2xl overflow-hidden shadow-2xl border border-slate-800 bg-slate-950">
    <div ref="mapContainer" class="w-full h-full min-h-[450px]"></div>

    <!-- 地图皮肤/风格一键切换器 -->
    <div class="absolute top-4 right-4 glass-panel p-1 rounded-xl flex items-center space-x-1.5 z-20 border border-slate-800 shadow-2xl">
      <div class="flex items-center space-x-1 bg-slate-950/80 p-0.5 rounded-lg border border-slate-800/80">
        <button 
          @click="changeMapStyle('dark')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1',
            currentStyle === 'dark' ? 'bg-cyan-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <span>🌙 科技深色</span>
        </button>
        <button 
          @click="changeMapStyle('normal')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1',
            currentStyle === 'normal' ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <span>☀️ 标准彩色</span>
        </button>
        <button 
          @click="changeMapStyle('satellite')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1',
            currentStyle === 'satellite' ? 'bg-indigo-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <span>🛰️ 卫星实景</span>
        </button>
      </div>

      <button 
        @click="resetMapOverview" 
        title="重置缩放镜头"
        class="px-2.5 py-1 bg-slate-900 hover:bg-slate-800 text-cyan-300 hover:text-white border border-slate-700 rounded-lg text-xs font-bold transition flex items-center space-x-1 shadow-md active:scale-95"
      >
        <span>🎯 重置全景</span>
      </button>
    </div>

    <div v-if="loading" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-slate-950/80 backdrop-blur-md">
      <div class="w-10 h-10 border-4 border-cyan-500 border-t-transparent rounded-full animate-spin mb-3"></div>
      <span class="text-xs text-cyan-400 font-bold">正在载入高德地图...</span>
    </div>

    <div v-if="loadError" class="absolute inset-0 z-30 bg-slate-950/90 backdrop-blur-md flex flex-col items-center justify-center p-4">
      <span class="text-red-400 font-bold mb-2">高德地图加载失败</span>
      <button @click="initMap" class="text-xs bg-cyan-600 text-white px-4 py-2 rounded-xl font-bold">重新加载</button>
    </div>
    
    <div class="absolute bottom-4 left-4 glass-panel px-3.5 py-2 rounded-xl flex items-center space-x-4 text-xs font-medium text-slate-300 pointer-events-none z-10 border border-slate-800">
      <div class="flex items-center space-x-2">
        <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 shadow-sm shadow-emerald-500/50"></span>
        <span>在线就绪</span>
      </div>
      <div class="flex items-center space-x-2">
        <span class="w-2.5 h-2.5 rounded-full bg-red-600 animate-pulse shadow-sm shadow-red-600/50"></span>
        <span>SOS 告警</span>
      </div>
      <div v-if="showTrack" class="flex items-center space-x-2 text-cyan-400 border-l border-slate-700 pl-3 font-bold">
        <span class="w-3 h-1 bg-cyan-400 rounded-full"></span>
        <span>GPS 历史轨迹漫游</span>
      </div>
    </div>
  </div>
</template>
