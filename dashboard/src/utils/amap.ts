/*
// 高德地图 JS API 密钥与配置（暂时注释）
import AMapLoader from '@amap/amap-jsapi-loader'
const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659'
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca'

if (typeof window !== 'undefined') {
  ;(window as any)._AMapSecurityConfig = {
    securityJsCode: AMAP_SECURITY_CODE,
  }
}
*/

let googleMapsPromise: Promise<any> | null = null

// Fetch Google Maps Key from backend config API
export const fetchGoogleMapsKey = async (): Promise<string> => {
  try {
    const API_URL = (import.meta as any).env?.VITE_API_BASE_URL || '/api/v1'
    const resp = await fetch(`${API_URL}/device/config/maps-key`)
    if (resp.ok) {
      const data = await resp.json()
      if (data.maps_key) {
        return data.maps_key
      }
    }
  } catch (e) {
    console.error('Failed to fetch maps key:', e)
  }
  return ''
}

// 单例 Google Maps API 预加载与缓存
export const getGoogleMapsInstance = (): Promise<any> => {
  if (typeof window === 'undefined') return Promise.reject('window undefined')
  if ((window as any).google && (window as any).google.maps) {
    return Promise.resolve((window as any).google.maps)
  }
  if (googleMapsPromise) return googleMapsPromise

  googleMapsPromise = new Promise(async (resolve, reject) => {
    const scriptId = 'google-maps-js-sdk'
    if (document.getElementById(scriptId)) {
      const checkInterval = setInterval(() => {
        if ((window as any).google && (window as any).google.maps) {
          clearInterval(checkInterval)
          resolve((window as any).google.maps)
        }
      }, 100)
      return
    }

    const key = await fetchGoogleMapsKey()
    if (!key) {
      reject(new Error('无法获取谷歌地图 API 密钥'))
      googleMapsPromise = null
      return
    }

    const script = document.createElement('script')
    script.id = scriptId
    script.src = `https://maps.googleapis.com/maps/api/js?key=${key}&libraries=places,visualization&language=zh-CN`
    script.async = true
    script.defer = true
    script.onload = () => {
      if ((window as any).google && (window as any).google.maps) {
        resolve((window as any).google.maps)
      } else {
        reject(new Error('Google Maps SDK 加载失败'))
      }
    }
    script.onerror = (err) => {
      googleMapsPromise = null
      reject(err)
    }
    document.head.appendChild(script)
  })

  return googleMapsPromise
}

// 别名方法，确保兼容调用的地方
export const getAMapInstance = getGoogleMapsInstance

