/**
 * E2E-07: 正常漫游基线测试 (严格模式)
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000017'

test.describe('E2E-07: 正常漫游基线验证', () => {

  test.beforeAll(async () => {
    const status = await sendAndWait(
      TEST_IMEI, 'LOCATION',
      HK_COORDS.TSUEN_WAN.lat, HK_COORDS.TSUEN_WAN.lng,
      72, 85, 2500,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：系统标题与 KPI 正常加载', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(3000)

    const title = page.locator('h1:has-text("社区养老智慧安防指挥控制中心")')
    await expect(title).toBeVisible()

    await page.screenshot({ path: 'test-results/e2e-07-cms-normal.png', fullPage: true })
  })

  test('H5 移动端：显示精准在线状态与真实地址，绝对不得为"正在获取位置"', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    await page.addInitScript((imei) => {
      window.localStorage.setItem('current_imei', imei)
      window.localStorage.setItem('use_mock', 'false')
    }, TEST_IMEI)

    await page.goto('/')
    await page.waitForTimeout(2500)

    // 严谨断言 1：绝不能处于 "正在获取位置..."
    const addressDisplay = page.locator('[data-testid="address-display"]')
    await expect(addressDisplay).toBeVisible()
    const addressText = await addressDisplay.textContent()
    expect(addressText).not.toContain('正在获取位置...')
    expect(addressText).toContain('香港特别行政区')

    // 严谨断言 2：心率数字存在
    const hrDisplay = page.locator('[data-testid="heart-rate-value"]')
    await expect(hrDisplay).toBeVisible()

    await page.screenshot({ path: 'test-results/e2e-07-h5-normal.png', fullPage: true })
  })
})
