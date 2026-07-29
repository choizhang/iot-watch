/**
 * E2E-03: 电子围栏越界告警测试 (严格模式)
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000019'

test.describe('E2E-03: 电子围栏越界告警', () => {

  test.beforeAll(async () => {
    const status = await sendAndWait(
      TEST_IMEI, 'LOCATION',
      HK_COORDS.OUT_OF_BOUNDS.lat, HK_COORDS.OUT_OF_BOUNDS.lng,
      82, 60, 2500,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：设备列表应正常加载该设备', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(3000)

    const deviceCards = page.locator('[class*="cursor-pointer"]')
    expect(await deviceCards.count()).toBeGreaterThan(0)

    await page.screenshot({ path: 'test-results/e2e-03-cms-geofence.png', fullPage: true })
  })

  test('H5 移动端：显示精准离散状态，绝对不得为"正在获取位置"', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    await page.addInitScript((imei) => {
      window.localStorage.setItem('current_imei', imei)
      window.localStorage.setItem('use_mock', 'false')
    }, TEST_IMEI)

    await page.goto('/')
    await page.waitForTimeout(2500)

    const addressDisplay = page.locator('[data-testid="address-display"]')
    await expect(addressDisplay).toBeVisible()
    const addressText = await addressDisplay.textContent()
    expect(addressText).not.toContain('正在获取位置...')
    expect(addressText).toContain('香港特别行政区')

    await page.screenshot({ path: 'test-results/e2e-03-h5-geofence.png', fullPage: true })
  })
})
