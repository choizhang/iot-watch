<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { 
  ChevronLeft, 
  ShieldAlert, 
  Plus, 
  Minus,
  Trash2, 
  MapPin, 
  Check, 
  X, 
  LocateFixed, 
  Layers,
  Edit3,
  AlertTriangle
} from 'lucide-vue-next'
import { useDeviceStore, type GeofenceData } from '../store/device'

/*
// ------------------------------------------------------------------
// 原高德地图 (AMap) 逻辑（暂时注释保留）
// ------------------------------------------------------------------
import AMapLoader from '@amap/amap-jsapi-loader'
const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659'
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca'
if (typeof window !== 'undefined') {
  ;(window as any)._AMapSecurityConfig = {
    securityJsCode: AMAP_SECURITY_CODE,
  }
}
*/

const GOOGLE_MAPS_KEY = 'AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw'

const router = useRouter()
const store = useDeviceStore()

const mapContainer = ref<HTMLDivElement | null>(null)
const loadingMap = ref(true)
const isEditMode = ref(false)
const toastMsg = ref('')

// 编辑模式下的表单状态
const editId = ref<number | null>(null)
const formName = ref('社区防走失安全圈')
const formLat = ref(30.658633)
const formLng = ref(104.064718)
const formRadius = ref(500)
const formType = ref<'IN' | 'OUT'>('IN')
const currentAddress = ref('正在解析地图中心位置...')

let mapInstance: any = null
let googleObj: any = null
let geocoder: any = null
let circleOverlays: any[] = []
let previewCircle: any = null
let deviceMarker: any = null

const zoomIn = () => {
  if (mapInstance) mapInstance.setZoom(mapInstance.getZoom() + 1)
}

const zoomOut = () => {
  if (mapInstance) mapInstance.setZoom(mapInstance.getZoom() - 1)
}

const loadGoogleMaps = (): Promise<any> => {
  if ((window as any).google && (window as any).google.maps) {
    return Promise.resolve((window as any).google.maps)
  }
  return new Promise((resolve, reject) => {
    const scriptId = 'google-maps-geofence-sdk'
    if (document.getElementById(scriptId)) {
      const timer = setInterval(() => {
        if ((window as any).google && (window as any).google.maps) {
          clearInterval(timer);
          resolve((window as any).google.maps);
        }
      }, 100);
      return;
    }
    const script = document.createElement('script')
    script.id = scriptId
    script.src = `https://maps.googleapis.com/maps/api/js?key=${GOOGLE_MAPS_KEY}&language=zh-CN`
    script.async = true
    script.defer = true
    script.onload = () => resolve((window as any).google.maps)
    script.onerror = reject
    document.head.appendChild(script)
  })
}

function createCustomOverlay(googleMaps: any, pos: { lat: number; lng: number }, element: HTMLElement) {
  class CustomOverlay extends googleMaps.OverlayView {
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
        this.element.style.left = point.x - 14 + 'px'
        this.element.style.top = point.y - 14 + 'px'
        this.element.style.position = 'absolute'
      }
    }
    onRemove() {
      if (this.element.parentNode) {
        this.element.parentNode.removeChild(this.element)
      }
    }
  }
  return new CustomOverlay(pos, element)
}

onMounted(async () => {
  await store.fetchStatus()
  await store.fetchGeofences()
  initMap()
})

onUnmounted(() => {
  if (mapInstance) {
    mapInstance = null
  }
})

const showToast = (msg: string) => {
  toastMsg.value = msg
  setTimeout(() => {
    if (toastMsg.value === msg) toastMsg.value = ''
  }, 2500)
}

const initMap = async () => {
  loadingMap.value = true
  try {
    googleObj = await loadGoogleMaps()

    if (!mapContainer.value) return

    geocoder = new googleObj.Geocoder()

    const lat = store.status?.last_latitude || 30.658633
    const lng = store.status?.last_longitude || 104.064718

    formLat.value = lat
    formLng.value = lng

    mapInstance = new googleObj.Map(mapContainer.value, {
      zoom: 15,
      center: { lat, lng },
      disableDefaultUI: true,
      zoomControl: false,
    })

    // 长者当前位置 Marker
    const markerDom = document.createElement('div')
    markerDom.className = 'relative flex items-center justify-center'
    markerDom.innerHTML = `
      <span class="animate-ping absolute inline-flex h-8 w-8 rounded-full bg-emerald-400 opacity-75"></span>
      <div class="w-7 h-7 rounded-full bg-emerald-600 border-2 border-white shadow-xl flex items-center justify-center text-white text-xs font-bold">
        📍
      </div>
    `
    deviceMarker = createCustomOverlay(googleObj, { lat, lng }, markerDom)
    deviceMarker.setMap(mapInstance)

    // 监听地图拖拽移动，实时更新中心点坐标与地址
    mapInstance.addListener('drag', onMapMove)
    mapInstance.addListener('idle', onMapMoveEnd)

    loadingMap.value = false
    renderAllGeofences()
  } catch (err) {
    console.error('围栏地图初始化失败:', err)
    loadingMap.value = false
  }
}

// 地图移动中
const onMapMove = () => {
  if (!isEditMode.value || !mapInstance) return
  const center = mapInstance.getCenter()
  formLat.value = center.lat()
  formLng.value = center.lng()
  updatePreviewCircle()
}

// 地图移动结束：逆地理编码解析地址
const onMapMoveEnd = () => {
  if (!isEditMode.value || !mapInstance || !geocoder) return
  const center = mapInstance.getCenter()
  formLat.value = center.lat()
  formLng.value = center.lng()

  geocoder.geocode({ location: { lat: formLat.value, lng: formLng.value } }, (results: any, status: string) => {
    if (status === 'OK' && results && results[0]) {
      currentAddress.value = results[0].formatted_address || '已知选点位置'
    } else {
      currentAddress.value = `${formLat.value.toFixed(4)}, ${formLng.value.toFixed(4)}`
    }
  })
}

// 渲染列表中所有启用围栏
const renderAllGeofences = () => {
  if (!mapInstance || !googleObj) return

  circleOverlays.forEach(c => c.setMap(null))
  circleOverlays = []

  if (isEditMode.value) return // 编辑模式下只渲染预览圆

  const fenceList = store.geofences || []
  fenceList.forEach(fence => {
    if (!fence.enabled) return
    const isOut = fence.fence_type === 'OUT'
    const circle = new googleObj.Circle({
      map: mapInstance,
      center: { lat: fence.latitude, lng: fence.longitude },
      radius: fence.radius,
      strokeColor: isOut ? '#ef4444' : '#0284c7',
      strokeOpacity: 0.8,
      strokeWeight: 2,
      fillColor: isOut ? '#ef4444' : '#0284c7',
      fillOpacity: 0.2,
    })
    circleOverlays.push(circle)
  })

  if (circleOverlays.length > 0) {
    const bounds = new googleObj.LatLngBounds()
    circleOverlays.forEach(circle => {
      if (circle && typeof circle.getBounds === 'function') {
        const b = circle.getBounds()
        if (b && !b.isEmpty()) bounds.union(b)
      }
    })
    if (!bounds.isEmpty()) {
      mapInstance.fitBounds(bounds)
      const listener = googleObj.event.addListenerOnce(mapInstance, 'bounds_changed', () => {
        if (mapInstance.getZoom() > 15) {
          mapInstance.setZoom(15)
        }
      })
      setTimeout(() => {
        googleObj.event.removeListener(listener)
        if (mapInstance && mapInstance.getZoom() > 15) {
          mapInstance.setZoom(15)
        }
      }, 300)
    }
  }
}

// 编辑模式下预览实时安全圆
const updatePreviewCircle = () => {
  if (!mapInstance || !googleObj || !isEditMode.value) return

  if (previewCircle) {
    previewCircle.setMap(null)
    previewCircle = null
  }

  const isOut = formType.value === 'OUT'
  previewCircle = new googleObj.Circle({
    map: mapInstance,
    center: { lat: formLat.value, lng: formLng.value },
    radius: formRadius.value,
    strokeColor: isOut ? '#ef4444' : '#10b981',
    strokeOpacity: 0.9,
    strokeWeight: 3,
    fillColor: isOut ? '#ef4444' : '#10b981',
    fillOpacity: 0.25,
    strokeStyle: 'solid'
  })
  previewCircle.setMap(mapInstance)
}

// 进入新建/编辑围栏选点模式
const enterEditMode = (fence?: GeofenceData) => {
  isEditMode.value = true
  circleOverlays.forEach(c => c.setMap(null))

  if (fence) {
    editId.value = fence.id
    formName.value = fence.name
    formLat.value = fence.latitude
    formLng.value = fence.longitude
    formRadius.value = fence.radius
    formType.value = fence.fence_type
  } else {
    editId.value = null
    formName.value = '社区防走失安全圈'
    formLat.value = store.status?.last_latitude || 22.371234
    formLng.value = store.status?.last_longitude || 114.115678
    formRadius.value = 500
    formType.value = 'IN'
  }

  if (mapInstance) {
    mapInstance.setZoom(15)
    mapInstance.panTo({ lat: formLat.value, lng: formLng.value })
  }
  updatePreviewCircle()
  onMapMoveEnd()
}

// 退出编辑模式
const exitEditMode = () => {
  isEditMode.value = false
  if (previewCircle) {
    previewCircle.setMap(null)
    previewCircle = null
  }
  renderAllGeofences()
}

// 保存围栏 (新建或更新)
const saveGeofence = async () => {
  if (!formName.value.trim()) {
    showToast('请输入围栏名称')
    return
  }

  if (editId.value) {
    await store.removeGeofence(editId.value)
  }
  await store.addGeofence(formName.value, formLat.value, formLng.value, formRadius.value, formType.value)

  showToast('围栏设置已成功保存！')
  exitEditMode()
}

// 移动地图复位到长者位置
const centerToDevice = () => {
  const lat = store.status?.last_latitude || 22.371234
  const lng = store.status?.last_longitude || 114.115678
  if (mapInstance) {
    mapInstance.setZoom(15)
    mapInstance.panTo({ lat, lng })
  }
}

watch(() => store.geofences, () => {
  if (!isEditMode.value) renderAllGeofences()
}, { deep: true })

watch(() => [formRadius.value, formType.value], () => {
  if (isEditMode.value) updatePreviewCircle()
})
</script>

<template>
  <div class="min-h-screen bg-slate-50 flex flex-col relative overflow-x-hidden select-none">
    
    <!-- Toast 全局轻提示 -->
    <div v-if="toastMsg" class="fixed top-16 left-1/2 -translate-x-1/2 z-50 bg-slate-900/90 text-white text-xs font-bold px-4 py-2 rounded-2xl shadow-xl backdrop-blur animate-in fade-in">
      {{ toastMsg }}
    </div>

    <!-- Header 顶部导航 -->
    <header class="bg-white/90 backdrop-blur px-4 py-3.5 flex items-center justify-between shadow-sm sticky top-0 z-30 border-b border-slate-100">
      <button @click="isEditMode ? exitEditMode() : router.back()" class="text-slate-800 p-1">
        <ChevronLeft :size="24" />
      </button>
      <h1 class="font-extrabold text-slate-800 text-base flex items-center space-x-1.5">
        <ShieldAlert :size="20" class="text-emerald-500" />
        <span>{{ isEditMode ? '拖动地图选择围栏中心' : '电子围栏守护' }}</span>
      </h1>

      <button 
        v-if="!isEditMode" 
        @click="enterEditMode()" 
        class="bg-emerald-500 hover:bg-emerald-600 text-white px-3 py-1.5 rounded-xl font-bold text-xs flex items-center space-x-1 shadow-md shadow-emerald-500/20 active:scale-95 transition"
      >
        <Plus :size="16" />
        <span>新建围栏</span>
      </button>
      <button v-else @click="exitEditMode" class="text-slate-400 font-bold text-xs px-2 py-1">
        取消
      </button>
    </header>

    <!-- 高德地图区 (全屏幕响应) -->
    <div :class="['relative w-full transition-all duration-300', isEditMode ? 'h-[55vh]' : 'h-[45vh]']">
      <div ref="mapContainer" class="w-full h-full"></div>

      <!-- 地图加载 Mask -->
      <div v-if="loadingMap" class="absolute inset-0 bg-white/80 flex items-center justify-center z-20">
        <div class="text-xs text-slate-500 animate-pulse font-bold">正在加载电子围栏地图...</div>
      </div>

      <!-- 编辑模式：屏幕固定准心图标 📍 Pin -->
      <div v-if="isEditMode" class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-20 pointer-events-none flex flex-col items-center">
        <div class="w-8 h-8 rounded-full bg-emerald-500 border-2 border-white shadow-2xl flex items-center justify-center text-white text-sm font-bold animate-bounce">
          📍
        </div>
        <div class="w-2.5 h-1 bg-slate-900/40 rounded-full blur-[1px]"></div>
      </div>

      <!-- 拖拽提示 Badge -->
      <div v-if="isEditMode" class="absolute top-3 left-1/2 -translate-x-1/2 z-20 bg-slate-900/80 text-white text-[11px] font-bold px-3 py-1 rounded-full shadow-lg backdrop-blur flex items-center space-x-1">
        <span>👈 拖动地图调整圈心，准心即为中心点 👉</span>
      </div>

      <!-- H5 统一竖向地图控件组 (统一 Icon，避开重叠) -->
      <div class="absolute bottom-6 right-4 flex flex-col bg-white border border-slate-200/90 rounded-2xl shadow-xl z-20 overflow-hidden text-slate-700 font-sans">
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
          @click="centerToDevice" 
          class="w-10 h-10 flex items-center justify-center hover:bg-slate-50 text-emerald-600 active:bg-emerald-50 transition"
          title="聚焦长者位置"
        >
          <LocateFixed :size="18" />
        </button>
      </div>
    </div>

    <!-- ===== 模式 A: 围栏列表看板视角 (非编辑模式) ===== -->
    <div v-if="!isEditMode" class="flex-1 p-4 space-y-4 bg-slate-50 pb-20">
      <div class="flex items-center justify-between">
        <h2 class="font-bold text-slate-800 text-sm flex items-center space-x-1.5">
          <Layers :size="16" class="text-emerald-500" />
          <span>已生效围栏列表 ({{ (store.geofences || []).length }})</span>
        </h2>
        <span class="text-[11px] text-slate-400">滑动开关实时启用</span>
      </div>

      <!-- 空状态提示 -->
      <div v-if="!store.geofences || store.geofences.length === 0" class="bg-white rounded-3xl p-8 text-center border border-slate-100 shadow-sm">
        <ShieldAlert :size="48" class="mx-auto text-slate-300 mb-3" />
        <p class="text-sm font-bold text-slate-700">暂未设置电子围栏</p>
        <p class="text-xs text-slate-400 mt-1 mb-4">设置防走失圈，长者离开范围自动警报</p>
        <button @click="enterEditMode()" class="px-6 py-2.5 bg-emerald-500 text-white text-xs font-bold rounded-2xl shadow-lg shadow-emerald-500/20">
          立即新建防走失圈
        </button>
      </div>

      <!-- 围栏卡片列表 -->
      <div v-else class="space-y-3">
        <div 
          v-for="fence in store.geofences" 
          :key="fence.id"
          class="bg-white rounded-3xl p-4 shadow-sm border border-slate-100/80 flex flex-col space-y-3"
        >
          <div class="flex items-center justify-between border-b border-slate-100 pb-3">
            <div class="flex items-center space-x-3">
              <div :class="['w-10 h-10 rounded-2xl flex items-center justify-center font-bold text-base',
                fence.fence_type === 'OUT' ? 'bg-rose-50 text-rose-500' : 'bg-emerald-50 text-emerald-600']"
              >
                <ShieldAlert :size="20" />
              </div>
              <div>
                <h3 class="font-bold text-slate-800 text-sm">{{ fence.name }}</h3>
                <p class="text-[11px] text-slate-400 font-mono mt-0.5">
                  {{ fence.fence_type === 'IN' ? '🟢 出界预警 (越出范围报警)' : '🔴 入界预警 (进入危险区报警)' }}
                </p>
              </div>
            </div>

            <!-- Enable 状态切换 Toggle -->
            <button 
              @click="store.toggleGeofence(fence.id)"
              :class="['w-11 h-6 rounded-full transition-colors relative p-0.5 active:scale-95', fence.enabled ? 'bg-emerald-500' : 'bg-slate-300']"
            >
              <div :class="['w-5 h-5 bg-white rounded-full shadow transition-transform', fence.enabled ? 'translate-x-5' : 'translate-x-0']"></div>
            </button>
          </div>

          <!-- 围栏详情信息与控制操作 -->
          <div class="flex items-center justify-between text-xs text-slate-600 pt-1">
            <div class="flex items-center space-x-2 font-mono">
              <span class="bg-slate-100 px-2 py-1 rounded-lg font-bold text-slate-700">保护半径: {{ fence.radius }}m</span>
              <span class="text-slate-400 font-sans">防护中心: 社区安全防护圈</span>
            </div>

            <div class="flex items-center space-x-1">
              <button 
                @click="enterEditMode(fence)"
                class="p-2 text-slate-500 hover:text-emerald-600 hover:bg-emerald-50 rounded-xl transition"
                title="修改编辑"
              >
                <Edit3 :size="16" />
              </button>
              <button 
                @click="store.removeGeofence(fence.id)"
                class="p-2 text-rose-400 hover:text-rose-600 hover:bg-rose-50 rounded-xl transition"
                title="删除围栏"
              >
                <Trash2 :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 模式 B: 交互选点画圈编辑 Drawer (悬浮底栏，完美避开底部 TabBar 遮挡) ===== -->
    <div 
      v-else 
      class="bg-white rounded-t-3xl shadow-2xl border-t border-slate-100 p-5 space-y-4 z-30 animate-in slide-in-from-bottom duration-200 mb-16"
    >
      <!-- 逆地理解析中心点地址提示 -->
      <div class="bg-emerald-50 border border-emerald-200/60 rounded-2xl p-3 flex items-center space-x-2.5">
        <MapPin :size="18" class="text-emerald-600 flex-shrink-0" />
        <div class="text-xs truncate">
          <span class="font-bold text-emerald-900">选点中心地址: </span>
          <span class="text-emerald-700 font-medium">{{ currentAddress }}</span>
        </div>
      </div>

      <!-- 围栏名称 -->
      <div>
        <label class="block text-xs font-bold text-slate-500 mb-1.5">围栏规则名称</label>
        <input 
          v-model="formName" 
          type="text" 
          placeholder="例如：社区防走失防护圈" 
          class="w-full bg-slate-50 border border-slate-200 rounded-2xl px-4 py-2.5 text-xs text-slate-800 font-bold focus:outline-none focus:border-emerald-500"
        />
      </div>

      <!-- 预警规则选择 -->
      <div class="grid grid-cols-2 gap-2">
        <button 
          @click="formType = 'IN'" 
          :class="['py-2.5 px-3 rounded-2xl text-xs font-bold transition flex items-center justify-center space-x-1.5 border',
            formType === 'IN' ? 'bg-emerald-500 border-emerald-500 text-white shadow-md shadow-emerald-500/20' : 'bg-slate-50 border-slate-200 text-slate-600']"
        >
          <ShieldAlert :size="16" />
          <span>出界告警 (越出范围)</span>
        </button>
        <button 
          @click="formType = 'OUT'" 
          :class="['py-2.5 px-3 rounded-2xl text-xs font-bold transition flex items-center justify-center space-x-1.5 border',
            formType === 'OUT' ? 'bg-rose-500 border-rose-500 text-white shadow-md shadow-rose-500/20' : 'bg-slate-50 border-slate-200 text-slate-600']"
        >
          <AlertTriangle :size="16" />
          <span>入界告警 (危险区域)</span>
        </button>
      </div>

      <!-- 半径调节 Slider & 快捷档位 -->
      <div>
        <div class="flex items-center justify-between text-xs font-bold text-slate-600 mb-1.5">
          <span>安全防护半径</span>
          <span class="text-emerald-600 font-mono text-sm">{{ formRadius }} 米</span>
        </div>

        <input 
          v-model.number="formRadius" 
          type="range" 
          min="100" 
          max="3000" 
          step="50"
          class="w-full accent-emerald-500 cursor-pointer h-2 bg-slate-200 rounded-lg"
        />

        <div class="grid grid-cols-4 gap-2 mt-2">
          <button 
            v-for="r in [300, 500, 1000, 2000]" 
            :key="r"
            @click="formRadius = r"
            :class="['py-1 rounded-xl text-[11px] font-mono font-bold transition border',
              formRadius === r ? 'bg-emerald-50 border-emerald-400 text-emerald-600' : 'bg-slate-50 border-slate-200 text-slate-500']"
          >
            {{ r }}m
          </button>
        </div>
      </div>

      <!-- 底部保存提交 -->
      <button 
        @click="saveGeofence" 
        class="w-full py-3.5 bg-emerald-500 hover:bg-emerald-600 active:scale-95 text-white font-extrabold text-sm rounded-2xl shadow-xl shadow-emerald-500/30 transition flex items-center justify-center space-x-2"
      >
        <Check :size="18" />
        <span>保存并启用电子围栏</span>
      </button>
    </div>

  </div>
</template>
