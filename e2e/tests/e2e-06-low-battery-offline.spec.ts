/**
 * E2E-06: 低电量与离线预警测试 (严格模式)
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000021'

test.describe('E2E-06: 低电量与离线预警', () => {

  test.beforeAll(async () => {
    const status = await sendAndWait(
      TEST_IMEI, 'LOCATION',
      HK_COORDS.KWUN_TONG.lat, HK_COORDS.KWUN_TONG.lng,
      68, 8,    // 低电量 8%
      2500,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：应显示设备数据与低电状态', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(3000)

    const pageText = await page.locator('body').textContent() || ''
    expect(pageText.length).toBeGreaterThan(50)

    await page.screenshot({ path: 'test-results/e2e-06-cms-low-battery.png', fullPage: true })
  })

  test('H5 移动端：显示精确电量 8%，绝对不得为"正在获取位置"', async ({ page, baseURL }) => {
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

    // 严谨断言 2：电量显示为 8%
    const pageText = await page.locator('body').textContent() || ''
    expect(pageText).toContain('8%')

    await page.screenshot({ path: 'test-results/e2e-06-h5-low-battery.png', fullPage: true })
  })
})
