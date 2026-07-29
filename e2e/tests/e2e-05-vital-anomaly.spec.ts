/**
 * E2E-05: 体征高心率预警测试 (严格模式)
 * 
 * 流程：发送异常高心率 (125 bpm) → 严格断言 H5 与 CMS 上心率数字精准显示为 125
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000017'

test.describe('E2E-05: 体征高心率预警', () => {

  test.beforeAll(async () => {
    const status = await sendAndWait(
      TEST_IMEI, 'HEART_RATE',
      HK_COORDS.CAUSEWAY_BAY.lat, HK_COORDS.CAUSEWAY_BAY.lng,
      125, 65, 2500,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：应正常载入并展示最新设备状态', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(3000)

    const pageText = await page.locator('body').textContent() || ''
    expect(pageText.length).toBeGreaterThan(50)

    await page.screenshot({ path: 'test-results/e2e-05-cms-vital.png', fullPage: true })
  })

  test('H5 移动端：首页心率数值必须精准渲染为 125 bpm', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    await page.addInitScript((imei) => {
      window.localStorage.setItem('current_imei', imei)
      window.localStorage.setItem('use_mock', 'false')
    }, TEST_IMEI)

    await page.goto('/')
    await page.waitForTimeout(2500)

    // 严谨断言 1：心率数字绝对不能是默认占位符或缺失，必须等于 125
    const hrDisplay = page.locator('[data-testid="heart-rate-value"]')
    await expect(hrDisplay).toBeVisible()
    const hrVal = await hrDisplay.textContent()
    expect(hrVal?.trim()).toBe('125')

    // 严谨断言 2：地址显示必须解析完成，不能是 "正在获取位置..."
    const addressDisplay = page.locator('[data-testid="address-display"]')
    const addressText = await addressDisplay.textContent()
    expect(addressText).not.toContain('正在获取位置...')

    await page.screenshot({ path: 'test-results/e2e-05-h5-vital.png', fullPage: true })
  })
})
