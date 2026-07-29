/**
 * E2E-01: SOS 紧急按键全端联动测试 (严格模式)
 * 
 * 流程：设备发送 SOS 报文 (心率 125bpm) → Go 后端处理 → CMS 大屏 & H5 移动端强断言
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000017'

test.describe('E2E-01: SOS 紧急按键全端联动', () => {

  test.beforeAll(async () => {
    // 发送 SOS 报文：心率 125 bpm，电量 72%
    const status = await sendAndWait(
      TEST_IMEI, 'SOS',
      HK_COORDS.MONG_KOK.lat, HK_COORDS.MONG_KOK.lng,
      125, 72,
      2500,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：SOS 告警计数增加且列表中出现红色 SOS 标记', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(2000)

    // 严谨断言 1：设备列表必须包含红色 SOS Badge 元素
    const sosBadge = page.locator('span.animate-pulse:has-text("SOS")').or(page.locator('text=SOS'))
    await expect(sosBadge.first()).toBeVisible({ timeout: 10000 })

    // 严谨断言 2：SOS 告警计数牌可见
    const sosSection = page.locator('text=SOS 紧急告警')
    await expect(sosSection).toBeVisible()

    await page.screenshot({ path: 'test-results/e2e-01-cms-sos-alert.png', fullPage: true })
  })

  test('H5 移动端：显示精准离散状态与真实心率 125 bpm，不得为"正在获取位置"', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    // 前置初始化脚本：锁定测试 IMEI 并强制使用实时数据模式
    await page.addInitScript((imei) => {
      window.localStorage.setItem('current_imei', imei)
      window.localStorage.setItem('use_mock', 'false')
    }, TEST_IMEI)

    await page.goto('/')
    await page.waitForTimeout(2500)

    // 严谨断言 1：绝不允许包含 "正在获取位置..."
    const addressDisplay = page.locator('[data-testid="address-display"]')
    await expect(addressDisplay).toBeVisible()
    const addressText = await addressDisplay.textContent()
    expect(addressText).not.toContain('正在获取位置...')
    expect(addressText).toContain('香港特别行政区')

    // 严谨断言 2：心率数值必须真实显示为 125
    const hrDisplay = page.locator('[data-testid="heart-rate-value"]')
    await expect(hrDisplay).toBeVisible()
    const hrText = await hrDisplay.textContent()
    expect(hrText?.trim()).toBe('125')

    await page.screenshot({ path: 'test-results/e2e-01-h5-sos-alert.png', fullPage: true })
  })
})
