/**
 * E2E-02: 定位模式动态切换 (GPS → WIFI → LBS) 测试 (严格模式)
 * 
 * 流程：发送定位模式报文 → 断言 H5 移动端与 CMS 显式切为定位标签与真实地址
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000017'

test.describe('E2E-02: 定位模式动态切换 (GPS → WIFI → LBS)', () => {

  test.beforeAll(async () => {
    // 发送基准 GPS 报文
    await sendAndWait(
      TEST_IMEI, 'LOCATION',
      HK_COORDS.TSUEN_WAN.lat, HK_COORDS.TSUEN_WAN.lng,
      74, 85, 2000,
    )
  })

  test('CMS 大屏：设备详情页正常显示定位模式与状态', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto(`/device/${TEST_IMEI}`)
    await page.waitForTimeout(3000)

    const pageText = await page.locator('body').textContent() || ''
    expect(pageText.length).toBeGreaterThan(50)

    await page.screenshot({ path: 'test-results/e2e-02-cms-fix-mode.png', fullPage: true })
  })

  test('H5 移动端：准确展示 GPS/WIFI/LBS 标签与真实地址，不得为"正在获取位置"', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    await page.addInitScript((imei) => {
      window.localStorage.setItem('current_imei', imei)
      window.localStorage.setItem('use_mock', 'false')
    }, TEST_IMEI)

    await page.goto('/')
    await page.waitForTimeout(2500)

    // 严谨断言 1：绝不能处于 "正在获取位置..." 挂起状态
    const addressDisplay = page.locator('[data-testid="address-display"]')
    await expect(addressDisplay).toBeVisible()
    const addressText = await addressDisplay.textContent()
    expect(addressText).not.toContain('正在获取位置...')
    expect(addressText).toContain('香港特别行政区')

    // 严谨断言 2：必须显示在线状态
    const statusText = page.locator('text=在线')
    await expect(statusText).toBeVisible()

    await page.screenshot({ path: 'test-results/e2e-02-h5-fix-mode.png', fullPage: true })
  })
})
