import { defineStore } from 'pinia'
import axios from 'axios'

export interface DeviceItem {
  imei: string
  owner_name: string
  owner_phone: string
  status: 'online' | 'offline' | 'sos_alert'
  battery: number
  last_heart_rate: number
  last_latitude: number
  last_longitude: number
  address: string
  updated_at: number
  fix_mode: 'GPS' | 'WIFI' | 'LBS' | 'LKP'
  satellites: number
  accuracy: number
  rssi: number
  bp: string
  spo2: number
  temperature: number
  steps: number
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

// 香港全境 18 个行政区散布词库与中心点
const HK_18_DISTRICTS = [
  { district: '中西区', center: [22.281, 114.155], street: '坚尼地城吉席街' },
  { district: '湾仔区', center: [22.275, 114.180], street: '铜锣湾轩尼诗道' },
  { district: '东区', center: [22.285, 114.215], street: '北角英皇道' },
  { district: '南区', center: [22.245, 114.155], street: '香港仔大道安老中心' },
  { district: '油尖旺区', center: [22.312, 114.170], street: '旺角弥敦道' },
  { district: '深水埗区', center: [22.332, 114.160], street: '长沙湾道' },
  { district: '九龙城区', center: [22.325, 114.190], street: '红磡马头围道' },
  { district: '黄大仙区', center: [22.342, 114.195], street: '龙翔道黄大仙祠旁' },
  { district: '观塘区', center: [22.310, 114.225], street: '九龙湾宏照道' },
  { district: '葵青区', center: [22.358, 114.128], street: '葵涌兴芳路' },
  { district: '荃湾区', center: [22.372, 114.115], street: '荃湾街市街安老院' },
  { district: '屯门区', center: [22.395, 113.975], street: '屯门屯喜路' },
  { district: '元朗区', center: [22.445, 114.025], street: '元朗青山公路' },
  { district: '北区', center: [22.500, 114.130], street: '上水符兴街' },
  { district: '大埔区', center: [22.450, 114.165], street: '大埔安慈路' },
  { district: '沙田区', center: [22.382, 114.190], street: '沙田正街' },
  { district: '西贡区', center: [22.315, 114.260], street: '将军澳重华路' },
  { district: '离岛区', center: [22.288, 113.940], street: '东涌文东路' },
]

const SURNAMES = ['张', '李', '王', '赵', '陈', '林', '黄', '周', '吴', '徐', '孙', '胡', '朱', '高', '郭', '罗', '梁', '宋', '郑', '谢']
const GIVEN_TITLES = ['老伯', '奶奶', '大爷', '阿婆', '伯伯', '婆婆', '婶婶', '阿叔']

// 生成 100 台分散全港、全场景覆盖的 Mock 设备与同步 SOS 告警风暴列表
export const generateMockData = (): { devices: DeviceItem[]; alarms: AlarmOrder[] } => {
  const devices: DeviceItem[] = []
  const alarms: AlarmOrder[] = []
  const now = Math.floor(Date.now() / 1000)

  // 固定的 3 台基础锚点设备
  devices.push(
    {
      imei: '1234567890',
      owner_name: '张老伯 (葵涌社区)',
      owner_phone: '13800138000',
      status: 'online',
      battery: 88,
      last_heart_rate: 76,
      last_latitude: 22.396428,
      last_longitude: 114.109497,
      address: '葵涌兴芳路 223 号新都会广场',
      updated_at: now,
      fix_mode: 'GPS',
      satellites: 14,
      accuracy: 4,
      rssi: -68,
      bp: '120/78',
      spo2: 99,
      temperature: 36.6,
      steps: 6420
    },
    {
      imei: '868811000000015',
      owner_name: '李奶奶 (荃湾社区)',
      owner_phone: '13900139000',
      status: 'sos_alert',
      battery: 45,
      last_heart_rate: 118,
      last_latitude: 22.371234,
      last_longitude: 114.115678,
      address: '荃湾街市街 55 号安老院',
      updated_at: now - 25,
      fix_mode: 'GPS',
      satellites: 12,
      accuracy: 5,
      rssi: -72,
      bp: '142/92',
      spo2: 96,
      temperature: 37.1,
      steps: 3200
    },
    {
      imei: '359000000000017',
      owner_name: '王大爷 (沙田分中心)',
      owner_phone: '13700137000',
      status: 'offline',
      battery: 12,
      last_heart_rate: 0,
      last_latitude: 22.381200,
      last_longitude: 114.189100,
      address: '沙田正街 18 号新城市广场',
      updated_at: now - 1800,
      fix_mode: 'LKP',
      satellites: 0,
      accuracy: 150,
      rssi: -110,
      bp: '118/75',
      spo2: 98,
      temperature: 36.5,
      steps: 1250
    }
  )

  // 李奶奶固定产生起 SOS 告警
  alarms.push({
    id: 1,
    alarm_id: 'ALARM-HK-20260727-001',
    device_imei: '868811000000015',
    alert_type: 'SOS 紧急手动求救',
    category: 'SOS',
    trigger_time: '今天 10:25:30',
    latitude: 22.371234,
    longitude: 114.115678,
    heart_rate: 118,
    status: 'UNHANDLED'
  })

  // 生成剩余 97 台高度分散在香港全境的设备
  for (let i = 4; i <= 100; i++) {
    const imei = `8690000000${i < 10 ? '0' + i : i}`
    const surname = SURNAMES[i % SURNAMES.length]
    const title = GIVEN_TITLES[i % GIVEN_TITLES.length]
    const distObj = HK_18_DISTRICTS[i % HK_18_DISTRICTS.length]
    const name = `${surname}${title} (${distObj.district})`

    // 状态分布: 10% SOS告警, 20% 离线失联, 70% 在线
    let status: 'online' | 'offline' | 'sos_alert' = 'online'
    if (i % 10 === 0) {
      status = 'sos_alert'
    } else if (i % 5 === 0) {
      status = 'offline'
    }

    // 定位技术分布: 60% GPS, 25% WIFI, 10% LBS, 5% LKP
    let fix_mode: 'GPS' | 'WIFI' | 'LBS' | 'LKP' = 'GPS'
    let satellites = Math.floor(Math.random() * 8) + 8
    let accuracy = Math.floor(Math.random() * 6) + 3
    let rssi = -60 - Math.floor(Math.random() * 25)

    if (status === 'offline') {
      fix_mode = 'LKP'
      satellites = 0
      accuracy = 180
      rssi = -112
    } else if (i % 7 === 0) {
      fix_mode = 'LBS'
      satellites = 0
      accuracy = 250
      rssi = -95
    } else if (i % 4 === 0) {
      fix_mode = 'WIFI'
      satellites = 3
      accuracy = 20
      rssi = -78
    }

    // 坐标随机散布算法 (在大行政区中心加上 ±0.035 度范围内的自然随机偏移)
    const latOffset = (Math.random() - 0.5) * 0.045
    const lngOffset = (Math.random() - 0.5) * 0.055
    const lat = Number((distObj.center[0] + latOffset).toFixed(6))
    const lng = Number((distObj.center[1] + lngOffset).toFixed(6))

    const battery = status === 'offline' ? Math.floor(Math.random() * 12) : Math.floor(Math.random() * 70) + 30
    const hr = status === 'offline' ? 0 : status === 'sos_alert' ? Math.floor(Math.random() * 35) + 110 : Math.floor(Math.random() * 30) + 65
    const sys = Math.floor(Math.random() * 25) + 115
    const dia = Math.floor(Math.random() * 15) + 72
    const spo2 = status === 'sos_alert' ? 94 + Math.floor(Math.random() * 3) : 97 + Math.floor(Math.random() * 3)
    const temp = Number((36.2 + Math.random() * 0.9).toFixed(1))
    const steps = Math.floor(Math.random() * 9000) + 800
    const timeAgo = status === 'offline' ? Math.floor(Math.random() * 7200) + 1800 : Math.floor(Math.random() * 300)

    devices.push({
      imei,
      owner_name: name,
      owner_phone: `13${Math.floor(Math.random() * 900000009 + 100000000)}`,
      status,
      battery,
      last_heart_rate: hr,
      last_latitude: lat,
      last_longitude: lng,
      address: `香港${distObj.district}${distObj.street}${Math.floor(Math.random() * 120 + 1)}号`,
      updated_at: now - timeAgo,
      fix_mode,
      satellites,
      accuracy,
      rssi,
      bp: `${sys}/${dia}`,
      spo2,
      temperature: temp,
      steps
    })

    // 如果是 SOS 设备，自动同步生成 1 条对应的实时 SOS 告警单
    if (status === 'sos_alert') {
      const alertConfigs: { type: string; category: 'SOS' | 'FALL' | 'VITAL' | 'GEOFENCE' }[] = [
        { type: 'SOS 紧急按键求救', category: 'SOS' },
        { type: '摔倒姿态异常告警', category: 'FALL' },
        { type: '静息心率过高告警 (128bpm)', category: 'VITAL' },
        { type: '越界离开社区生活圈', category: 'GEOFENCE' }
      ]
      const cfg = alertConfigs[alarms.length % alertConfigs.length]
      alarms.push({
        id: alarms.length + 1,
        alarm_id: `ALARM-HK-20260727-${alarms.length + 1 < 10 ? '00' + (alarms.length + 1) : '0' + (alarms.length + 1)}`,
        device_imei: imei,
        alert_type: cfg.type,
        category: cfg.category,
        trigger_time: `今天 ${new Date((now - timeAgo) * 1000).toTimeString().split(' ')[0]}`,
        latitude: lat,
        longitude: lng,
        heart_rate: hr,
        status: 'UNHANDLED'
      })
    }
  }

  return { devices, alarms }
}

const mockInit = generateMockData()

export const useDashboardStore = defineStore('dashboard', {
  state: () => ({
    devices: mockInit.devices,
    selectedImei: '', // 默认进入大屏时不选中任何设备，展现全港全景
    alarmOrders: mockInit.alarms,
    loading: false,
    soundEnabled: true,
    autoRefresh: true,
    lastRefreshTime: new Date(),
    mockMode: true, // 默认开启全场景 Mock 模式
    toastMessage: '',
  }),

  getters: {
    onlineCount: (state) => state.devices.filter(d => d.status === 'online').length,
    offlineCount: (state) => state.devices.filter(d => d.status === 'offline').length,
    sosCount: (state) => state.devices.filter(d => d.status === 'sos_alert').length,
    totalCount: (state) => state.devices.length,
    onlineRate: (state) => Math.round((state.devices.filter(d => d.status === 'online').length / (state.devices.length || 1)) * 100),
    unhandledAlarmsCount: (state) => state.alarmOrders.filter(a => a.status === 'UNHANDLED').length,
    criticalAlarmsCount: (state) => state.alarmOrders.filter(a => a.category === 'SOS' || a.category === 'FALL').length,
    warningAlarmsCount: (state) => state.alarmOrders.filter(a => a.category === 'VITAL' || a.category === 'GEOFENCE').length,
    countSOS: (state) => state.alarmOrders.filter(a => a.category === 'SOS' || a.category === 'FALL').length,
    countFALL: (state) => state.alarmOrders.filter(a => a.category === 'FALL').length,
    countVITAL: (state) => state.alarmOrders.filter(a => a.category === 'VITAL').length,
    countGEOFENCE: (state) => state.alarmOrders.filter(a => a.category === 'GEOFENCE').length,
    selectedDevice: (state) => state.devices.find(d => d.imei === state.selectedImei) || null,
  },

  actions: {
    async fetchAllDevices() {
      this.loading = true
      try {
        if (this.mockMode) {
          this.lastRefreshTime = new Date()
          return
        }

        // 真实 TCP直连模式：如果没选中设备则查默认第一个设备 status
        const targetImei = this.selectedImei || '868811000000015'
        const res = await axios.get(`${API_BASE}/device/${targetImei}/status`)
        const realImei = res.data.imei || targetImei

        const idx = this.devices.findIndex(d => d.imei === realImei)
        if (idx !== -1) {
          const validLat = Number(res.data.last_latitude) > 0 ? Number(res.data.last_latitude) : this.devices[idx].last_latitude
          const validLng = Number(res.data.last_longitude) > 0 ? Number(res.data.last_longitude) : this.devices[idx].last_longitude

          this.devices[idx] = {
            ...this.devices[idx],
            status: res.data.status || this.devices[idx].status,
            battery: res.data.battery ?? this.devices[idx].battery,
            last_heart_rate: res.data.last_heart_rate ?? this.devices[idx].last_heart_rate,
            last_latitude: validLat,
            last_longitude: validLng,
            updated_at: res.data.updated_at || Math.floor(Date.now() / 1000)
          }
        }
        this.lastRefreshTime = new Date()
      } catch (err) {
        console.error('获取大屏设备数据失败', err)
      } finally {
        this.loading = false
      }
    },

    // 显式切换 Mock 模式与 真实 TCP 直连模式
    toggleMockMode() {
      this.mockMode = !this.mockMode
      if (this.mockMode) {
        const data = generateMockData()
        this.devices = data.devices
        this.alarmOrders = data.alarms
        this.showToast('已成功切换为 [100台全场景 MOCK 模式]！')
      } else {
        // 恢复真实 TCP 模式下的 3 台基础设备
        this.devices = [
          {
            imei: '1234567890',
            owner_name: '张老伯 (葵涌社区)',
            owner_phone: '13800138000',
            status: 'online',
            battery: 88,
            last_heart_rate: 76,
            last_latitude: 22.396428,
            last_longitude: 114.109497,
            address: '葵涌兴芳路 223 号新都会广场',
            updated_at: Math.floor(Date.now() / 1000),
            fix_mode: 'GPS',
            satellites: 14,
            accuracy: 4,
            rssi: -68,
            bp: '120/78',
            spo2: 99,
            temperature: 36.6,
            steps: 6420
          },
          {
            imei: '868811000000015',
            owner_name: '李奶奶 (荃湾社区)',
            owner_phone: '13900139000',
            status: 'sos_alert',
            battery: 45,
            last_heart_rate: 118,
            last_latitude: 22.371234,
            last_longitude: 114.115678,
            address: '荃湾街市街 55 号安老院',
            updated_at: Math.floor(Date.now() / 1000) - 25,
            fix_mode: 'GPS',
            satellites: 12,
            accuracy: 5,
            rssi: -72,
            bp: '142/92',
            spo2: 96,
            temperature: 37.1,
            steps: 3200
          },
          {
            imei: '359000000000017',
            owner_name: '王大爷 (沙田分中心)',
            owner_phone: '13700137000',
            status: 'offline',
            battery: 12,
            last_heart_rate: 0,
            last_latitude: 22.381200,
            last_longitude: 114.189100,
            address: '沙田正街 18 号新城市广场',
            updated_at: Math.floor(Date.now() / 1000) - 1800,
            fix_mode: 'LKP',
            satellites: 0,
            accuracy: 150,
            rssi: -110,
            bp: '118/75',
            spo2: 98,
            temperature: 36.5,
            steps: 1250
          }
        ]
        this.alarmOrders = [
          {
            id: 1,
            alarm_id: 'ALARM-HK-20260727-001',
            device_imei: '868811000000015',
            alert_type: 'SOS 紧急求救',
            category: 'SOS',
            trigger_time: '今天 10:25:30',
            latitude: 22.371234,
            longitude: 114.115678,
            heart_rate: 118,
            status: 'UNHANDLED'
          }
        ]
        this.showToast('已切换为 [真实后端 TCP 直连模式]！')
        this.fetchAllDevices()
      }
    },

    showToast(msg: string) {
      this.toastMessage = msg
      setTimeout(() => {
        if (this.toastMessage === msg) this.toastMessage = ''
      }, 3000)
    },

    async sendDeviceCommand(imei: string, command: string) {
      try {
        if (this.mockMode) {
          return { status: 'success', message: `[Mock模式] 指令 ${command} 已成功下发至手环 #${imei.slice(-4)}` }
        }
        const res = await axios.post(`${API_BASE}/device/${imei}/command`, { command })
        return res.data
      } catch (err: any) {
        throw new Error(err.response?.data?.message || '指令发送失败，设备可能掉线')
      }
    },

    selectDevice(imei: string) {
      this.selectedImei = imei
      this.fetchAllDevices()
    },

    clearSelection() {
      this.selectedImei = ''
    },

    toggleSound() {
      this.soundEnabled = !this.soundEnabled
    },

    toggleAutoRefresh() {
      this.autoRefresh = !this.autoRefresh
    }
  }
})
