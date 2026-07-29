/**
 * IoT 设备发包模拟器 (Device Simulator)
 * 
 * 封装向 Go 后端发送 EMQX Webhook JSON 报文的能力，
 * 供 Playwright E2E 测试用例在断言 UI 前调用。
 */

const BACKEND_URL = 'http://localhost:8080/api/v1/device/raw-tcp'

/**
 * 将十进制度数转为 DDMM.MMMM 格式（手环协议标准）
 */
function degreesToDDMM(deg: number, isLng: boolean): string {
  const d = Math.floor(deg)
  const m = (deg - d) * 60
  const pad = isLng ? 3 : 2
  return d.toString().padStart(pad, '0') + m.toFixed(4).padStart(7, '0')
}

/**
 * 构造标准手环 *HQ 协议 ASCII 报文
 */
export function buildHQFrame(
  imei: string,
  msgType: string,
  lat: number,
  lng: number,
  heartRate: number,
  battery: number,
): string {
  const now = new Date()
  const timeStr =
    now.getHours().toString().padStart(2, '0') +
    now.getMinutes().toString().padStart(2, '0') +
    now.getSeconds().toString().padStart(2, '0')

  const latStr = degreesToDDMM(lat, false)
  const lngStr = degreesToDDMM(lng, true)

  return `*HQ,${imei},${msgType},${timeStr},A,${latStr},N,${lngStr},E,${heartRate},${battery}#`
}

/**
 * 向 Go 后端发送一帧设备报文（经 EMQX Webhook JSON 封装）
 * 返回 HTTP 状态码
 */
export async function sendDeviceFrame(
  imei: string,
  msgType: string,
  lat: number,
  lng: number,
  heartRate: number,
  battery: number,
): Promise<number> {
  const payload = buildHQFrame(imei, msgType, lat, lng, heartRate, battery)

  const body = JSON.stringify({
    clientid: imei,
    topic: 'device/raw-tcp',
    payload,
  })

  const res = await fetch(BACKEND_URL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  })

  return res.status
}

/**
 * 发送报文后等待一段时间让后端处理 + 前端轮询刷新
 */
export async function sendAndWait(
  imei: string,
  msgType: string,
  lat: number,
  lng: number,
  heartRate: number,
  battery: number,
  waitMs = 2000,
): Promise<number> {
  const status = await sendDeviceFrame(imei, msgType, lat, lng, heartRate, battery)
  await new Promise((r) => setTimeout(r, waitMs))
  return status
}

// 香港常用坐标
export const HK_COORDS = {
  TSUEN_WAN:    { lat: 22.396428, lng: 114.109497 },   // 荃湾
  MONG_KOK:     { lat: 22.319300, lng: 114.169400 },   // 旺角
  CENTRAL:      { lat: 22.281900, lng: 114.158800 },   // 中环
  CAUSEWAY_BAY: { lat: 22.280000, lng: 114.184000 },   // 铜锣湾
  KWUN_TONG:    { lat: 22.313000, lng: 114.226000 },   // 观塘
  // 围栏越界用 — 偏离荃湾中心 >1km
  OUT_OF_BOUNDS: { lat: 22.450000, lng: 114.200000 },
}
