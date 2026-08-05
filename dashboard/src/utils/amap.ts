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

// 谷歌地图 (Google Maps JS API) 配置密钥
export const GOOGLE_MAPS_KEY = 'AIzaSyBOti4mM-6x9WDnZIjIeyEU21OpBXqWBgw'

let googleMapsPromise: Promise<any> | null = null

// 单例 Google Maps API 预加载与缓存
export const getGoogleMapsInstance = (): Promise<any> => {
  if (typeof window === 'undefined') return Promise.reject('window undefined')
  if ((window as any).google && (window as any).google.maps) {
    return Promise.resolve((window as any).google.maps)
  }
  if (googleMapsPromise) return googleMapsPromise

  googleMapsPromise = new Promise((resolve, reject) => {
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

    const script = document.createElement('script')
    script.id = scriptId
    script.src = `https://maps.googleapis.com/maps/api/js?key=${GOOGLE_MAPS_KEY}&libraries=places,visualization&language=zh-CN`
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

