<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { getGoogleMapsInstance } from '../utils/amap'
import type { DeviceItem } from '../store/dashboard'
import { Plus, Minus, Crosshair, MapPin, Moon, Sun, Globe } from 'lucide-vue-next'

/*
// ------------------------------------------------------------------
// 原高德地图 (AMap) 逻辑（暂时注释保留）
// ------------------------------------------------------------------
import { getAMapInstance } from '../utils/amap'
let AMapObj: any = null
let geocoder: any = null
let heatmapObj: any = null
let satelliteLayer: any = null
...
*/

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
let googleObj: any = null
let markers: any[] = []
let trackPolyline: any = null
let trackMarkers: any[] = []
let runnerMarker: any = null
let heatmapObj: any = null
let fenceCircles: any[] = []

// 谷歌地图暗黑模式风格 Style JSON
const DARK_MAP_STYLE = [
  { elementType: 'geometry', stylers: [{ color: '#1d2c4d' }] },
  { elementType: 'labels.text.stroke', stylers: [{ color: '#1a3646' }] },
  { elementType: 'labels.text.fill', stylers: [{ color: '#8ec3b9' }] },
  { featureType: 'administrative.country', elementType: 'geometry.stroke', stylers: [{ color: '#4b6878' }] },
  { featureType: 'administrative.province', elementType: 'geometry.stroke', stylers: [{ color: '#4b6878' }] },
  { featureType: 'landscape.man_made', elementType: 'geometry.stroke', stylers: [{ color: '#334e68' }] },
  { featureType: 'landscape.natural', elementType: 'geometry', stylers: [{ color: '#021019' }] },
  { featureType: 'poi', elementType: 'geometry', stylers: [{ color: '#283d6a' }] },
  { featureType: 'poi', elementType: 'labels.text.fill', stylers: [{ color: '#6f9ba5' }] },
  { featureType: 'poi.park', elementType: 'geometry.fill', stylers: [{ color: '#023e58' }] },
  { featureType: 'road', elementType: 'geometry', stylers: [{ color: '#304a7d' }] },
  { featureType: 'road', elementType: 'labels.text.fill', stylers: [{ color: '#98a5be' }] },
  { featureType: 'road.highway', elementType: 'geometry', stylers: [{ color: '#2c4595' }] },
  { featureType: 'road.highway', elementType: 'geometry.stroke', stylers: [{ color: '#1f2835' }] },
  { featureType: 'transit', elementType: 'labels.text.fill', stylers: [{ color: '#98a5be' }] },
  { featureType: 'water', elementType: 'geometry', stylers: [{ color: '#0e1626' }] },
  { featureType: 'water', elementType: 'labels.text.fill', stylers: [{ color: '#4e6d96' }] }
]

// 防御性安全坐标计算
const getValidCoords = (dev: DeviceItem): [number, number] => {
  let lat = Number(dev.last_latitude)
  let lng = Number(dev.last_longitude)

  if (isNaN(lat) || lat <= 0) {
    lat = dev.imei === '1234567890' ? 22.396428 : dev.imei === '868811000000015' ? 22.371234 : 30.658633
  }
  if (isNaN(lng) || lng <= 0) {
    lng = dev.imei === '1234567890' ? 114.109497 : dev.imei === '868811000000015' ? 114.115678 : 104.064718
  }
  return [lng, lat]
}

const changeMapStyle = (style: 'dark' | 'normal' | 'satellite') => {
  currentStyle.value = style
  if (!mapInstance || !googleObj) return

  if (style === 'satellite') {
    mapInstance.setMapTypeId(googleObj.MapTypeId.HYBRID)
  } else if (style === 'dark') {
    mapInstance.setMapTypeId(googleObj.MapTypeId.ROADMAP)
    mapInstance.setOptions({ styles: DARK_MAP_STYLE })
  } else {
    mapInstance.setMapTypeId(googleObj.MapTypeId.ROADMAP)
    mapInstance.setOptions({ styles: [] })
  }
}

const zoomIn = () => {
  if (mapInstance) mapInstance.setZoom(mapInstance.getZoom() + 1)
}

const zoomOut = () => {
  if (mapInstance) mapInstance.setZoom(mapInstance.getZoom() - 1)
}

const resetMapOverview = () => {
  if (!mapInstance || !googleObj) return
  const singleDevice = props.devices.length === 1 ? props.devices[0] : null
  const [lng, lat] = singleDevice ? getValidCoords(singleDevice) : [104.064718, 30.658633]
  const targetZoom = props.zoomLevel ? props.zoomLevel : (singleDevice ? 17 : 12)
  mapInstance.setZoom(targetZoom)
  mapInstance.panTo({ lat, lng })
  emit('reset-view')
}

// 定义用于 Google Maps 的 HTML 自定义 Marker 覆盖物
function createCustomHTMLOverlay(googleMaps: any, pos: { lat: number; lng: number }, element: HTMLElement) {
  class CustomHTMLOverlay extends googleMaps.OverlayView {
    position: any
    element: HTMLElement
    constructor(p: { lat: number; lng: number }, el: HTMLElement) {
      super()
      this.position = new googleMaps.LatLng(p.lat, p.lng)
      this.element = el
    }
    onAdd() {
      const panes = this.getPanes()
      panes?.overlayMouseTarget.appendChild(this.element)
    }
    draw() {
      const overlayProjection = this.getProjection()
      if (!overlayProjection) return
      const point = overlayProjection.fromLatLngToDivPixel(this.position)
      if (point) {
        this.element.style.left = point.x + 'px'
        this.element.style.top = point.y + 'px'
        this.element.style.transform = 'translate(-50%, -50%)'
        this.element.style.position = 'absolute'
      }
    }
    onRemove() {
      if (this.element.parentNode) {
        this.element.parentNode.removeChild(this.element)
      }
    }
  }
  return new CustomHTMLOverlay(pos, element)
}

const initMap = async () => {
  loading.value = true
  loadError.value = false
  try {
    googleObj = await getGoogleMapsInstance()

    if (!mapContainer.value) return

    const singleDevice = props.devices.length === 1 ? props.devices[0] : null
    const [lng, lat] = singleDevice 
      ? getValidCoords(singleDevice)
      : [104.064718, 30.658633]

    const initialZoom = props.zoomLevel ? props.zoomLevel : (singleDevice ? 17 : 12)

    mapInstance = new googleObj.Map(mapContainer.value, {
      zoom: initialZoom,
      center: { lat, lng },
      mapTypeId: currentStyle.value === 'satellite' ? googleObj.MapTypeId.HYBRID : googleObj.MapTypeId.ROADMAP,
      styles: currentStyle.value === 'dark' ? DARK_MAP_STYLE : [],
      disableDefaultUI: true,
      zoomControl: false,
    })

    loading.value = false
    renderMarkers()
    renderTrack()
    renderHeatmap()
    renderGeofences()
    renderRunner()
  } catch (err) {
    console.error('谷歌地图 (Google Maps) 加载失败:', err)
    loading.value = false
    loadError.value = true
  }
}

let accuracyCircles: any[] = []

const renderMarkers = () => {
  if (!mapInstance || !googleObj) return

  markers.forEach(m => m.setMap(null))
  markers = []
  accuracyCircles.forEach(c => c.setMap(null))
  accuracyCircles = []

  const bounds = new googleObj.LatLngBounds()

  props.devices.forEach((dev) => {
    const [lng, lat] = getValidCoords(dev)
    bounds.extend({ lat, lng })

    const isSOS = dev.status === 'sos_alert'
    const isOnline = dev.status === 'online'
    const isSelected = dev.imei === props.selectedImei
    const accuracyMeter = dev.accuracy || 18

    // 绘制定位精度与误差米数范围圈 (Accuracy Circle)
    const accuracyColor = isSOS ? '#ef4444' : isOnline ? '#10b981' : '#f59e0b'
    const accuracyCircle = new googleObj.Circle({
      map: mapInstance,
      center: { lat, lng },
      radius: accuracyMeter,
      strokeColor: accuracyColor,
      strokeOpacity: 0.8,
      strokeWeight: 1.5,
      fillColor: accuracyColor,
      fillOpacity: 0.08,
      clickable: false,
    })
    accuracyCircles.push(accuracyCircle)

    const markerDom = document.createElement('div')
    markerDom.style.position = 'relative'
    markerDom.style.cursor = 'pointer'
    markerDom.className = 'z-20'

    // 状态颜色划分：在线 (翡翠绿 #10b981), 告警 (警示红 #ef4444), 离线 (冷灰 #94a3b8)
    const iconColor = isSOS ? '#ef4444' : isOnline ? '#10b981' : '#94a3b8'
    const strapColor = isSOS ? '#dc2626' : isOnline ? '#059669' : '#64748b'
    const iconBgFill = isSOS ? '#fee2e2' : isOnline ? '#ecfdf5' : '#f8fafc'

    const displayName = dev.owner_name || dev.device_name || (dev.imei ? '长者设备 #' + dev.imei.slice(-4) : '手环设备')

    const watchSvg = `<svg xmlns="http://www.w3.org/2000/svg" width="34" height="34" viewBox="0 0 24 24" fill="none" stroke="${iconColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="filter drop-shadow-md">
      <!-- 上下表带 -->
      <path d="M9 1h6v5H9z" fill="${strapColor}" stroke="${iconColor}" stroke-width="1.2"/>
      <path d="M9 18h6v5H9z" fill="${strapColor}" stroke="${iconColor}" stroke-width="1.2"/>
      <!-- 表盘外壳 -->
      <rect x="6" y="6" width="12" height="12" rx="3.5" fill="${iconBgFill}" stroke="${iconColor}" stroke-width="2"/>
      <!-- 表盘心圈与时间指示点 -->
      <circle cx="12" cy="12" r="2" fill="none" stroke="${iconColor}" stroke-width="1.5" />
      <!-- 右侧表冠按键 -->
      <path d="M18.5 10v4" stroke="${iconColor}" stroke-width="2" stroke-linecap="round" />
    </svg>`

    markerDom.innerHTML = `
      <div class="relative flex items-center justify-center select-none group cursor-pointer z-30">
        <!-- 1. 顶部名称卡片: absolute bottom-full 脱标，保证不侵占文档流高度与几何中点 -->
        <div class="absolute bottom-full mb-1.5 px-2.5 py-0.5 bg-slate-900/95 border border-slate-700/80 text-white text-xs font-bold rounded-lg shadow-xl whitespace-nowrap flex items-center space-x-1.5 backdrop-blur-md transition-transform group-hover:scale-110 pointer-events-none">
          <span class="w-2 h-2 rounded-full ${isSOS ? 'bg-red-500 animate-ping' : isOnline ? 'bg-emerald-400' : 'bg-slate-400'}"></span>
          <span>${displayName}</span>
          <span class="text-[9px] px-1 py-0.2 rounded font-mono text-cyan-300 bg-slate-800 border border-slate-700">±${accuracyMeter}m</span>
          ${isSOS ? '<span class="bg-red-600 text-white text-[9px] px-1 rounded animate-pulse">SOS</span>' : ''}
        </div>

        <!-- 2. 纯粹智能手表 Icon (几何像素中心点 100% 死锁误差圆圈的圆心) -->
        <div class="relative flex items-center justify-center transition-transform duration-200 hover:scale-125 ${isSelected ? 'scale-125 z-40' : ''}">
          ${isSOS ? '<span class="animate-ping absolute inline-flex h-8 w-8 rounded-full bg-red-500 opacity-60"></span>' : ''}
          ${watchSvg}
        </div>
      </div>
    `

    markerDom.addEventListener('click', () => {
      emit('select-device', dev.imei)
    })

    const overlay = createCustomHTMLOverlay(googleObj, { lat, lng }, markerDom)
    overlay.setMap(mapInstance)
    markers.push(overlay)
  })

  if (props.devices.length === 1) {
    const singleDevice = props.devices[0]
    if (singleDevice) {
      const [lng, lat] = getValidCoords(singleDevice)
      const targetZoom = props.zoomLevel ? props.zoomLevel : 15
      mapInstance.setZoom(targetZoom)
      mapInstance.panTo({ lat, lng })
    }
  } else if (props.devices.length > 1 && !bounds.isEmpty()) {
    mapInstance.fitBounds(bounds)
  }
}

const renderGeofences = () => {
  if (!mapInstance || !googleObj) return

  fenceCircles.forEach(c => c.setMap(null))
  fenceCircles = []

  if (props.geofences && props.geofences.length > 0) {
    props.geofences.forEach(fence => {
      if (!fence.enabled) return
      const isOut = fence.fence_type === 'OUT'
      const circle = new googleObj.Circle({
        map: mapInstance,
        center: { lat: fence.latitude, lng: fence.longitude },
        radius: fence.radius,
        strokeColor: isOut ? '#ef4444' : '#0284c7',
        strokeOpacity: 0.85,
        strokeWeight: 2,
        fillColor: isOut ? '#ef4444' : '#0284c7',
        fillOpacity: 0.2,
      })
      fenceCircles.push(circle)
    })
  }
}

const renderTrack = () => {
  if (!mapInstance || !googleObj) return

  if (trackPolyline) {
    trackPolyline.setMap(null)
    trackPolyline = null
  }
  trackMarkers.forEach(m => m.setMap(null))
  trackMarkers = []

  if (props.showTrack && props.trackPoints && props.trackPoints.length > 1) {
    const path = props.trackPoints.map(([lng, lat]) => ({ lat, lng }))
    trackPolyline = new googleObj.Polyline({
      path,
      geodesic: true,
      strokeColor: '#38bdf8',
      strokeOpacity: 0.95,
      strokeWeight: 5,
      map: mapInstance,
    })

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

      const tMarker = createCustomHTMLOverlay(googleObj, { lat: pt[1], lng: pt[0] }, pointDom)
      tMarker.setMap(mapInstance)
      trackMarkers.push(tMarker)
    })

    if (!props.runnerPoint) {
      const bounds = new googleObj.LatLngBounds()
      path.forEach(p => bounds.extend(p))
      mapInstance.fitBounds(bounds)
    }
  }
}

// 轨迹回放动态移动 Runner Marker
const renderRunner = () => {
  if (!mapInstance || !googleObj) return

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
    runnerMarker = createCustomHTMLOverlay(googleObj, { lat: props.runnerPoint[1], lng: props.runnerPoint[0] }, runnerDom)
    runnerMarker.setMap(mapInstance)
    mapInstance.panTo({ lat: props.runnerPoint[1], lng: props.runnerPoint[0] })
  }
}

const renderHeatmap = () => {
  if (!mapInstance || !googleObj) return

  if (heatmapObj) {
    heatmapObj.setMap(null)
    heatmapObj = null
  }

  if (props.showHeatmap && props.heatmapPoints && props.heatmapPoints.length > 0 && googleObj.visualization) {
    const heatmapData = props.heatmapPoints.map(p => ({
      location: new googleObj.LatLng(p.lat, p.lng),
      weight: p.count
    }))
    heatmapObj = new googleObj.visualization.HeatmapLayer({
      data: heatmapData,
      radius: props.heatmapRadius || 35,
      map: mapInstance
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
    mapInstance.setZoom(targetZoom)
    mapInstance.panTo({ lat: targetLat, lng: targetLng })
  }
  renderMarkers()
})

let userLocationMarker: any = null

const locateUserPosition = () => {
  if (!navigator.geolocation) {
    resetMapOverview()
    return
  }
  navigator.geolocation.getCurrentPosition(
    (pos) => {
      const uLat = pos.coords.latitude
      const uLng = pos.coords.longitude
      if (mapInstance && googleObj) {
        mapInstance.panTo({ lat: uLat, lng: uLng })
        mapInstance.setZoom(16)

        if (userLocationMarker) userLocationMarker.setMap(null)
        userLocationMarker = new googleObj.Marker({
          position: { lat: uLat, lng: uLng },
          map: mapInstance,
          title: '我的当前位置',
          icon: {
            path: googleObj.SymbolPath.CIRCLE,
            scale: 8,
            fillColor: '#38bdf8',
            fillOpacity: 1,
            strokeColor: '#ffffff',
            strokeWeight: 2,
          }
        })
      }
    },
    (err) => {
      console.warn('获取当前定位失败:', err)
      resetMapOverview()
    },
    { enableHighAccuracy: true, timeout: 5000 }
  )
}

onMounted(() => {
  initMap()
})

onUnmounted(() => {
  if (userLocationMarker) {
    userLocationMarker.setMap(null)
    userLocationMarker = null
  }
  if (mapInstance) {
    mapInstance = null
  }
})
</script>

<template>
  <div class="relative w-full h-full rounded-2xl overflow-hidden shadow-2xl border border-slate-800 bg-slate-950">
    <div ref="mapContainer" class="w-full h-full min-h-[450px]"></div>

    <!-- 1. 顶部地图皮肤/风格切换器 -->
    <div class="absolute top-4 right-4 glass-panel p-1 rounded-xl flex items-center z-20 border border-slate-800 shadow-2xl">
      <div class="flex items-center space-x-1 bg-slate-950/80 p-0.5 rounded-lg border border-slate-800/80">
        <button 
          @click="changeMapStyle('dark')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1.5',
            currentStyle === 'dark' ? 'bg-cyan-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <Moon :size="13" />
          <span>科技深色</span>
        </button>
        <button 
          @click="changeMapStyle('normal')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1.5',
            currentStyle === 'normal' ? 'bg-blue-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <Sun :size="13" />
          <span>标准彩色</span>
        </button>
        <button 
          @click="changeMapStyle('satellite')"
          :class="['px-2.5 py-1 rounded-md text-xs font-bold transition flex items-center space-x-1.5',
            currentStyle === 'satellite' ? 'bg-indigo-600 text-white shadow-md' : 'text-slate-400 hover:text-white']"
        >
          <Globe :size="13" />
          <span>卫星实景</span>
        </button>
      </div>
    </div>

    <!-- 2. 右侧统一竖向地图定位与缩放面板 -->
    <div class="absolute top-16 right-4 flex flex-col bg-slate-900/90 border border-slate-800 backdrop-blur-md rounded-2xl shadow-2xl z-20 overflow-hidden text-slate-300 font-sans">
      <button 
        @click="zoomIn"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-800/80 hover:bg-slate-800 text-slate-300 hover:text-white active:scale-95 transition"
        title="放大地图"
      >
        <Plus :size="18" />
      </button>
      <button 
        @click="zoomOut"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-800/80 hover:bg-slate-800 text-slate-300 hover:text-white active:scale-95 transition"
        title="缩小地图"
      >
        <Minus :size="18" />
      </button>
      <button 
        @click="resetMapOverview"
        class="w-10 h-10 flex items-center justify-center border-b border-slate-800/80 hover:bg-slate-800 text-cyan-400 hover:text-cyan-300 active:scale-95 transition"
        title="聚焦设备位置"
      >
        <Crosshair :size="18" />
      </button>
      <button 
        @click="locateUserPosition"
        class="w-10 h-10 flex items-center justify-center hover:bg-slate-800 text-blue-400 hover:text-blue-300 active:scale-95 transition"
        title="我的当前位置"
      >
        <MapPin :size="18" />
      </button>
    </div>

    <div v-if="loading" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-slate-950/80 backdrop-blur-md">
      <div class="w-10 h-10 border-4 border-cyan-500 border-t-transparent rounded-full animate-spin mb-3"></div>
      <span class="text-xs text-cyan-400 font-bold">正在载入谷歌地图 (Google Maps)...</span>
    </div>

    <div v-if="loadError" class="absolute inset-0 z-30 bg-slate-950/90 backdrop-blur-md flex flex-col items-center justify-center p-4">
      <span class="text-red-400 font-bold mb-2">谷歌地图 (Google Maps) 加载失败</span>
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
        <span>历史轨迹漫游</span>
      </div>
    </div>
  </div>
</template>
