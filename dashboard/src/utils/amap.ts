import AMapLoader from '@amap/amap-jsapi-loader'

const AMAP_KEY = '51f8039ddd81720e5acd9cf08f4da659'
const AMAP_SECURITY_CODE = '1bc91108fb1a45315f635b3671d2ffca'

if (typeof window !== 'undefined') {
  ;(window as any)._AMapSecurityConfig = {
    securityJsCode: AMAP_SECURITY_CODE,
  }
}

let amapPromise: Promise<any> | null = null

// 单例高德 JS API 预加载与全局 Promise 缓存 (全局仅初始化加载一次)
export const getAMapInstance = (): Promise<any> => {
  if (amapPromise) return amapPromise

  amapPromise = AMapLoader.load({
    key: AMAP_KEY,
    version: '2.0',
    plugins: [
      'AMap.Geocoder', 
      'AMap.Marker', 
      'AMap.Pixel', 
      'AMap.Polyline', 
      'AMap.HeatMap', 
      'AMap.TileLayer'
    ],
  }).catch((err) => {
    amapPromise = null // 失败重置，允许重试
    throw err
  })

  return amapPromise
}
