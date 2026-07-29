import { defineStore } from 'pinia'
import axios from 'axios'

export interface DeviceStatus {
  imei: string
  status: 'online' | 'offline' | 'sos_alert'
  battery: number
  last_heart_rate: number
  last_latitude: number
  last_longitude: number
  address: string
  updated_at: number
  location_type?: 'GPS' | 'WIFI' | 'LBS'
}

export interface HealthData {
  blood_pressure: string
  heart_rate: number
  blood_oxygen: number
  temperature: number
  steps: number
  sleep: string
  hrv: number
  updated_at: string
}

export interface GeofenceData {
  id: number
  imei: string
  name: string
  latitude: number
  longitude: number
  radius: number
  fence_type: 'IN' | 'OUT'
  enabled: boolean
}

const API_BASE = 'http://localhost:8080/api/v1'

export const useDeviceStore = defineStore('device', {
  state: () => ({
    currentImei: localStorage.getItem('current_imei') || '1234567890',
    status: null as DeviceStatus | null,
    health: null as HealthData | null,
    loading: false,
    error: null as string | null,
    useMock: localStorage.getItem('use_mock') === 'true',
    contacts: [] as any[],
    settings: { interval: 300 } as { interval: number },
    reminders: [] as any[],
    geofences: [] as GeofenceData[],
  }),

  actions: {
    async fetchStatus() {
      this.loading = true
      try {
        if (this.useMock) {
          this.status = {
            imei: this.currentImei,
            status: 'online',
            battery: 88,
            last_heart_rate: 76,
            last_latitude: 22.396428,
            last_longitude: 114.109497,
            address: '香港葵青區葵涌興芳路 223 號新都會廣場',
            updated_at: Math.floor(Date.now() / 1000),
            location_type: 'GPS'
          }
          this.error = null
          return
        }

        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/status`)
        this.status = res.data
        this.error = null
      } catch (err) {
        this.error = '获取设备状态失败'
        console.error(err)
      } finally {
        this.loading = false
      }
    },

    async fetchHealth() {
      try {
        if (this.useMock) {
          this.health = {
            blood_pressure: '120/78',
            heart_rate: 75,
            blood_oxygen: 99,
            temperature: 36.6,
            steps: 8432,
            sleep: '7小时45分钟',
            hrv: 48,
            updated_at: new Date().toISOString()
          }
          return
        }

        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/health`)
        this.health = res.data
      } catch (err) {
        console.error('获取健康数据失败', err)
      }
    },

    setImei(imei: string) {
      this.currentImei = imei
      localStorage.setItem('current_imei', imei)
      this.fetchStatus()
      this.fetchHealth()
      this.fetchContacts()
      this.fetchSettings()
      this.fetchReminders()
    },

    setUseMock(val: boolean) {
      this.useMock = val
      localStorage.setItem('use_mock', val ? 'true' : 'false')
      this.fetchStatus()
      this.fetchHealth()
      this.fetchContacts()
      this.fetchSettings()
      this.fetchReminders()
    },

    async fetchContacts() {
      if (this.useMock) {
        const cached = localStorage.getItem(`contacts_${this.currentImei}`)
        if (cached) {
          this.contacts = JSON.parse(cached)
        } else {
          this.contacts = [
            { id: 1, name: '女儿小美', phone: '13800138000', relation: '家属' },
            { id: 2, name: '社区医生', phone: '0755-2345678', relation: '医疗' },
          ]
          localStorage.setItem(`contacts_${this.currentImei}`, JSON.stringify(this.contacts))
        }
        return
      }

      try {
        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/contacts`)
        this.contacts = res.data
      } catch (err) {
        console.error('获取联系人失败', err)
      }
    },

    async addContact(name: string, phone: string, relation: string) {
      if (this.useMock) {
        const contact = { id: Date.now(), name, phone, relation }
        this.contacts.push(contact)
        localStorage.setItem(`contacts_${this.currentImei}`, JSON.stringify(this.contacts))
        return
      }

      try {
        const res = await axios.post(`${API_BASE}/device/${this.currentImei}/contacts`, { name, phone, relation })
        this.contacts.push(res.data)
      } catch (err) {
        console.error('添加联系人失败', err)
      }
    },

    async removeContact(id: number) {
      if (this.useMock) {
        this.contacts = this.contacts.filter(c => c.id !== id)
        localStorage.setItem(`contacts_${this.currentImei}`, JSON.stringify(this.contacts))
        return
      }

      try {
        await axios.delete(`${API_BASE}/device/${this.currentImei}/contacts/${id}`)
        this.contacts = this.contacts.filter(c => c.id !== id)
      } catch (err) {
        console.error('删除联系人失败', err)
      }
    },

    async fetchSettings() {
      if (this.useMock) {
        const cached = localStorage.getItem(`settings_${this.currentImei}`)
        if (cached) {
          this.settings = JSON.parse(cached)
        } else {
          this.settings = { interval: 300 }
          localStorage.setItem(`settings_${this.currentImei}`, JSON.stringify(this.settings))
        }
        return
      }

      try {
        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/settings`)
        this.settings = res.data
      } catch (err) {
        console.error('获取设置失败', err)
      }
    },

    async updateSettings(interval: number) {
      if (this.useMock) {
        this.settings = { interval }
        localStorage.setItem(`settings_${this.currentImei}`, JSON.stringify(this.settings))
        return
      }

      try {
        const res = await axios.post(`${API_BASE}/device/${this.currentImei}/settings`, { interval })
        this.settings = res.data
      } catch (err) {
        console.error('更新设置失败', err)
      }
    },

    async fetchReminders() {
      if (this.useMock) {
        const cached = localStorage.getItem(`reminders_${this.currentImei}`)
        if (cached) {
          this.reminders = JSON.parse(cached)
        } else {
          this.reminders = [
            { id: 1, time: '08:00', label: '早晨吃药', enabled: true },
            { id: 2, time: '12:00', label: '午餐时间', enabled: true },
            { id: 3, time: '18:00', label: '傍晚散步', enabled: false },
          ]
          localStorage.setItem(`reminders_${this.currentImei}`, JSON.stringify(this.reminders))
        }
        return
      }

      try {
        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/reminders`)
        this.reminders = res.data
      } catch (err) {
        console.error('获取提醒失败', err)
      }
    },

    async addReminder(time: string, label: string) {
      if (this.useMock) {
        const reminder = { id: Date.now(), time, label, enabled: true }
        this.reminders.push(reminder)
        localStorage.setItem(`reminders_${this.currentImei}`, JSON.stringify(this.reminders))
        return
      }

      try {
        const res = await axios.post(`${API_BASE}/device/${this.currentImei}/reminders`, { time, label })
        this.reminders.push(res.data)
      } catch (err) {
        console.error('添加提醒失败', err)
      }
    },

    async removeReminder(id: number) {
      if (this.useMock) {
        this.reminders = this.reminders.filter(r => r.id !== id)
        localStorage.setItem(`reminders_${this.currentImei}`, JSON.stringify(this.reminders))
        return
      }

      try {
        await axios.delete(`${API_BASE}/device/${this.currentImei}/reminders/${id}`)
        this.reminders = this.reminders.filter(r => r.id !== id)
      } catch (err) {
        console.error('删除提醒失败', err)
      }
    },

    async sendCommand(command: string) {
      if (this.useMock) {
        return { code: 200, message: '模拟指令下发成功', command }
      }

      try {
        const res = await axios.post(`${API_BASE}/device/${this.currentImei}/command`, { command })
        return res.data
      } catch (err: any) {
        console.error('下发指令失败', err)
        throw new Error(err.response?.data?.message || '下发指令失败，设备可能离线')
      }
    },

    async fetchGeofences() {
      if (this.useMock) {
        const cached = localStorage.getItem(`geofences_${this.currentImei}`)
        if (cached) {
          this.geofences = JSON.parse(cached)
        } else {
          this.geofences = [
            {
              id: 1,
              imei: this.currentImei,
              name: '荃湾社区安全防走失围栏',
              latitude: 22.371234,
              longitude: 114.115678,
              radius: 500,
              fence_type: 'IN',
              enabled: true,
            }
          ]
          localStorage.setItem(`geofences_${this.currentImei}`, JSON.stringify(this.geofences))
        }
        return
      }

      try {
        const res = await axios.get(`${API_BASE}/device/${this.currentImei}/geofences`)
        this.geofences = res.data
      } catch (err) {
        console.error('获取电子围栏失败', err)
      }
    },

    async addGeofence(name: string, latitude: number, longitude: number, radius: number, fence_type: 'IN' | 'OUT') {
      if (this.useMock) {
        const fence: GeofenceData = {
          id: Date.now(),
          imei: this.currentImei,
          name,
          latitude,
          longitude,
          radius,
          fence_type,
          enabled: true,
        }
        this.geofences.push(fence)
        localStorage.setItem(`geofences_${this.currentImei}`, JSON.stringify(this.geofences))
        return
      }

      try {
        const res = await axios.post(`${API_BASE}/device/${this.currentImei}/geofences`, {
          name,
          latitude,
          longitude,
          radius,
          fence_type,
        })
        this.geofences.push(res.data)
      } catch (err) {
        console.error('创建电子围栏失败', err)
      }
    },

    async removeGeofence(id: number) {
      if (this.useMock) {
        this.geofences = this.geofences.filter(g => g.id !== id)
        localStorage.setItem(`geofences_${this.currentImei}`, JSON.stringify(this.geofences))
        return
      }

      try {
        await axios.delete(`${API_BASE}/device/${this.currentImei}/geofences/${id}`)
        this.geofences = this.geofences.filter(g => g.id !== id)
      } catch (err) {
        console.error('删除电子围栏失败', err)
      }
    },

    async toggleGeofence(id: number) {
      if (this.useMock) {
        const item = this.geofences.find(g => g.id === id)
        if (item) {
          item.enabled = !item.enabled
          localStorage.setItem(`geofences_${this.currentImei}`, JSON.stringify(this.geofences))
        }
        return
      }

      try {
        const res = await axios.put(`${API_BASE}/device/${this.currentImei}/geofences/${id}/toggle`)
        const idx = this.geofences.findIndex(g => g.id === id)
        if (idx !== -1) {
          this.geofences[idx] = res.data
        }
      } catch (err) {
        console.error('切换围栏使能状态失败', err)
      }
    }
  }
})
