import { defineStore } from 'pinia'
import axios from 'axios'

export interface DeviceItem {
  imei: string
  owner_name: string
  owner_phone: string
  device_name?: string
  status: 'online' | 'offline' | 'sos_alert'
  battery: number
  last_heart_rate: number
  last_latitude: number
  last_longitude: number
  address: string
  updated_at: number
  fix_mode: 'GPS' | 'WIFI' | 'LBS'
  location_type?: string
  last_location_type?: string
  satellites: number
  accuracy: number
  rssi: number
  bp: string
  spo2: number
  temperature: number
  steps: number
  city?: '成都市' | '香港'
}

export interface AlarmOrder {
  id: number
  alarm_id: string
  device_imei: string
  alert_type: string
  category: 'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE'
  trigger_time: string
  latitude: number
  longitude: number
  heart_rate: number
  status: 'UNHANDLED' | 'HANDLING' | 'COMPLETED'
  handler_name?: string
  handler_notes?: string
}

const API_BASE = 'http://localhost:8080/api/v1'

// ─── 成都市行政区 ────────────────────────────────────────────────────────────
const CHENGDU_DISTRICTS = [
  { district: '武侯区', center: [30.642, 104.043], street: '人民南路四段' },
  { district: '锦江区', center: [30.655, 104.083], street: '春熙路红星路三段' },
  { district: '青羊区', center: [30.665, 104.053], street: '天府广场宽窄巷子旁' },
  { district: '金牛区', center: [30.692, 104.051], street: '一环路北二段' },
  { district: '成华区', center: [30.672, 104.103], street: '建设路万达广场旁' },
  { district: '高新区', center: [30.582, 104.063], street: '天府大道中段软件园' },
  { district: '双流区', center: [30.572, 103.923], street: '双流机场航站楼附近' },
  { district: '温江区', center: [30.682, 103.833], street: '光华大道三段' },
]

// ─── 香港 18 区 ───────────────────────────────────────────────────────────────
const HK_DISTRICTS = [
  { district: '葵青区', center: [22.358, 114.128], street: '葵涌兴芳路' },
  { district: '荃湾区', center: [22.372, 114.115], street: '荃湾街市街' },
  { district: '沙田区', center: [22.382, 114.190], street: '沙田正街' },
  { district: '油尖旺区', center: [22.312, 114.170], street: '旺角弥敦道' },
  { district: '深水埗区', center: [22.332, 114.160], street: '长沙湾道' },
  { district: '九龙城区', center: [22.325, 114.190], street: '红磡马头围道' },
  { district: '黄大仙区', center: [22.342, 114.195], street: '龙翔道' },
  { district: '观塘区',   center: [22.310, 114.225], street: '九龙湾宏照道' },
  { district: '屯门区',   center: [22.395, 113.975], street: '屯门屯喜路' },
  { district: '元朗区',   center: [22.445, 114.025], street: '元朗青山公路' },
  { district: '北区',     center: [22.500, 114.130], street: '上水符兴街' },
  { district: '大埔区',   center: [22.450, 114.165], street: '大埔安慈路' },
  { district: '西贡区',   center: [22.315, 114.260], street: '将军澳重华路' },
  { district: '中西区',   center: [22.281, 114.155], street: '坚尼地城吉席街' },
  { district: '湾仔区',   center: [22.275, 114.180], street: '铜锣湾轩尼诗道' },
  { district: '东区',     center: [22.285, 114.215], street: '北角英皇道' },
  { district: '南区',     center: [22.245, 114.155], street: '香港仔大道' },
  { district: '离岛区',   center: [22.288, 113.940], street: '东涌文东路' },
]

const SURNAMES     = ['张','李','王','赵','陈','林','黄','周','吴','徐','孙','胡','朱','高','郭','罗','梁','宋','郑','谢']
const GIVEN_TITLES = ['老伯','奶奶','大爷','阿婆','伯伯','婆婆','婶婶','阿叔']

// ─── Mock 数据生成（成都市 50 台 + 香港 50 台，共 100 台）────────────────────
export const generateMockData = (): { devices: DeviceItem[]; alarms: AlarmOrder[] } => {
  const devices: DeviceItem[] = []
  const alarms: AlarmOrder[]  = []
  const now = Math.floor(Date.now() / 1000)

  const makeDevices = (
    city: '成都市' | '香港',
    districts: typeof CHENGDU_DISTRICTS,
    count: number,
    startIdx: number
  ) => {
    for (let i = 0; i < count; i++) {
      const idx      = startIdx + i
      const imei     = `869${city === '成都市' ? '1' : '2'}${String(idx).padStart(11, '0')}`
      const surname  = SURNAMES[idx % SURNAMES.length]
      const title    = GIVEN_TITLES[idx % GIVEN_TITLES.length]
      const distObj  = districts[idx % districts.length]
      const name     = `${surname}${title} (${distObj.district})`

      let status: 'online' | 'offline' | 'sos_alert' = 'online'
      if (idx % 10 === 0) status = 'sos_alert'
      else if (idx % 5 === 0) status = 'offline'

      let fix_mode: 'GPS'|'WIFI'|'LBS' = 'GPS'
      let satellites = Math.floor(Math.random() * 8) + 8
      let accuracy   = Math.floor(Math.random() * 6) + 3
      let rssi       = -60 - Math.floor(Math.random() * 25)

      if (status === 'offline') { fix_mode = 'LBS'; satellites = 0; accuracy = 180; rssi = -112 }
      else if (idx % 7 === 0)  { fix_mode = 'LBS'; satellites = 0; accuracy = 250; rssi = -95 }
      else if (idx % 4 === 0)  { fix_mode = 'WIFI'; satellites = 3; accuracy = 20; rssi = -78 }

      const latOffset = (Math.random() - 0.5) * 0.045
      const lngOffset = (Math.random() - 0.5) * 0.055
      const lat = Number((distObj.center[0] + latOffset).toFixed(6))
      const lng = Number((distObj.center[1] + lngOffset).toFixed(6))

      const battery  = status === 'offline' ? Math.floor(Math.random() * 12) : Math.floor(Math.random() * 70) + 30
      const hr       = status === 'offline' ? 0 : status === 'sos_alert' ? Math.floor(Math.random() * 35) + 110 : Math.floor(Math.random() * 30) + 65
      const sys      = Math.floor(Math.random() * 25) + 115
      const dia      = Math.floor(Math.random() * 15) + 72
      const spo2     = status === 'sos_alert' ? 94 + Math.floor(Math.random() * 3) : 97 + Math.floor(Math.random() * 3)
      const temp     = Number((36.2 + Math.random() * 0.9).toFixed(1))
      const steps    = Math.floor(Math.random() * 9000) + 800
      const timeAgo  = status === 'offline' ? Math.floor(Math.random() * 7200) + 1800 : Math.floor(Math.random() * 300)
      const prefix   = city === '成都市' ? city : '香港'

      devices.push({
        imei, owner_name: name, owner_phone: `13${Math.floor(Math.random() * 900000009 + 100000000)}`,
        status, battery, last_heart_rate: hr, last_latitude: lat, last_longitude: lng,
        address: `${prefix}${distObj.district}${distObj.street}${Math.floor(Math.random() * 120 + 1)}号`,
        updated_at: now - timeAgo, fix_mode, satellites, accuracy, rssi,
        bp: `${sys}/${dia}`, spo2, temperature: temp, steps, city
      })

      if (status === 'sos_alert') {
        const alertConfigs: { type: string; category: 'SOS'|'FALL'|'VITAL'|'GEOFENCE' }[] = [
          { type: 'SOS 紧急按键求救', category: 'SOS' },
          { type: '摔倒姿态异常告警', category: 'FALL' },
          { type: '静息心率过高 (128bpm)', category: 'VITAL' },
          { type: '越界离开社区生活圈', category: 'GEOFENCE' },
        ]
        const cfg = alertConfigs[alarms.length % alertConfigs.length]
        alarms.push({
          id: alarms.length + 1,
          alarm_id: `ALARM-${city === '成都市' ? 'CD' : 'HK'}-${String(alarms.length + 1).padStart(3, '0')}`,
          device_imei: imei, alert_type: cfg.type, category: cfg.category,
          trigger_time: `今天 ${new Date((now - timeAgo) * 1000).toTimeString().split(' ')[0]}`,
          latitude: lat, longitude: lng, heart_rate: hr, status: 'UNHANDLED'
        })
      }
    }
  }

  makeDevices('成都市', CHENGDU_DISTRICTS, 50, 0)
  makeDevices('香港',   HK_DISTRICTS,      50, 50)

  return { devices, alarms }
}

// ─── Store ────────────────────────────────────────────────────────────────────
export const useDashboardStore = defineStore('dashboard', {
  state: () => ({
    devices:         [] as DeviceItem[],
    selectedImei:    '',
    alarmOrders:     [] as AlarmOrder[],
    loading:         false,
    soundEnabled:    true,
    autoRefresh:     true,
    lastRefreshTime: new Date(),
    mockMode:        false,   // false = 实时 TCP 直连模式
    toastMessage:    '',
  }),

  getters: {
    onlineCount:          (state) => state.devices.filter(d => d.status === 'online').length,
    offlineCount:         (state) => state.devices.filter(d => d.status === 'offline').length,
    sosCount:             (state) => state.devices.filter(d => d.status === 'sos_alert').length,
    totalCount:           (state) => state.devices.length,
    onlineRate:           (state) => Math.round((state.devices.filter(d => d.status === 'online').length / (state.devices.length || 1)) * 100),
    unhandledAlarmsCount: (state) => state.alarmOrders.filter(a => a.status === 'UNHANDLED').length,
    criticalAlarmsCount:  (state) => state.alarmOrders.filter(a => a.category === 'SOS' || a.category === 'FALL').length,
    warningAlarmsCount:   (state) => state.alarmOrders.filter(a => a.category === 'VITAL' || a.category === 'GEOFENCE').length,
    countSOS:             (state) => state.alarmOrders.filter(a => a.category === 'SOS' || a.category === 'FALL').length,
    countFALL:            (state) => state.alarmOrders.filter(a => a.category === 'FALL').length,
    countVITAL:           (state) => state.alarmOrders.filter(a => a.category === 'VITAL').length,
    countGEOFENCE:        (state) => state.alarmOrders.filter(a => a.category === 'GEOFENCE').length,
    selectedDevice:       (state) => state.devices.find(d => d.imei === state.selectedImei) || null,
  },

  actions: {
    // ── 拉取数据（根据模式自动选择 mock / 真实后端）──────────────────────────
    async fetchAllDevices() {
      this.loading = true
      try {
        if (this.mockMode) {
          // Mock 模式：生成双城全量 mock 数据
          const data = generateMockData()
          this.devices     = data.devices
          this.alarmOrders = data.alarms
        } else {
          // TCP 直连模式：只从后端 API 拉取真实注册设备，无 mock 注入
          const res = await axios.get(`${API_BASE}/devices`)
          if (res.data && Array.isArray(res.data) && res.data.length > 0) {
            // 后端数据默认标记为成都市（当前真实设备在成都）
            this.devices = (res.data as DeviceItem[]).map(d => ({
              ...d,
              city: (d.city ?? '成都市') as '成都市' | '香港',
            }))
          } else {
            this.devices = []
          }
          // 告警数据也从后端拉取（如果后端无数据则清空）
          try {
            const alarmRes = await axios.get(`${API_BASE}/alarms`)
            if (alarmRes.data && Array.isArray(alarmRes.data)) {
              this.alarmOrders = alarmRes.data
            } else {
              this.alarmOrders = []
            }
          } catch {
            this.alarmOrders = []
          }
        }
        this.lastRefreshTime = new Date()
      } catch (err) {
        console.error('设备数据拉取失败:', err)
      } finally {
        this.loading = false
      }
    },

    // ── 切换 Mock / TCP 直连模式 ───────────────────────────────────────────────
    toggleMockMode() {
      this.mockMode = !this.mockMode
      if (this.mockMode) {
        this.showToast('已切换为 [全场景 Mock 模式] — 成都市 + 香港双城演示数据')
      } else {
        this.showToast('已切换为 [实时 TCP 直连模式] — 仅显示真实注册设备')
      }
      this.fetchAllDevices()
    },

    showToast(msg: string) {
      this.toastMessage = msg
      setTimeout(() => { if (this.toastMessage === msg) this.toastMessage = '' }, 3500)
    },

    async sendDeviceCommand(imei: string, command: string) {
      try {
        if (this.mockMode) {
          return { status: 'success', message: `[Mock] 指令 ${command} 已下发至手环 #${imei.slice(-4)}` }
        }
        const res = await axios.post(`${API_BASE}/device/${imei}/command`, { command })
        return res.data
      } catch (err: any) {
        throw new Error(err.response?.data?.message || '指令发送失败，设备可能掉线')
      }
    },

    selectDevice(imei: string) {
      this.selectedImei = imei
    },

    clearSelection() {
      this.selectedImei = ''
    },

    toggleSound() {
      this.soundEnabled = !this.soundEnabled
    },

    toggleAutoRefresh() {
      this.autoRefresh = !this.autoRefresh
    },
  }
})
