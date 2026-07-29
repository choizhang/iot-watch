/**
 * E2E-04: 跌倒告警测试
 * 
 * 流程：设备发送 FALL 报文 → 验证 CMS 大屏出现告警标记
 */
import { test, expect } from '@playwright/test'
import { sendAndWait, HK_COORDS } from '../helpers/device-simulator'

const TEST_IMEI = '359000000000020'

test.describe('E2E-04: 跌倒告警检测', () => {

  test.beforeAll(async () => {
    const status = await sendAndWait(
      TEST_IMEI, 'FALL',
      HK_COORDS.CENTRAL.lat, HK_COORDS.CENTRAL.lng,
      95, 70, 3000,
    )
    expect(status).toBe(200)
  })

  test('CMS 大屏：设备列表应显示告警/异常状态标记', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5174'), '仅在 CMS 大屏项目运行')

    await page.goto('/')
    await page.waitForTimeout(3000)

    // 断言：页面中存在告警相关的视觉元素（SOS/告警文字或红色指示灯）
    const pageText = await page.locator('body').textContent()
    const hasAlertUI = pageText?.includes('SOS') ||
      pageText?.includes('告警') ||
      pageText?.includes('紧急')
    expect(hasAlertUI).toBeTruthy()

    await page.screenshot({ path: 'test-results/e2e-04-cms-fall.png', fullPage: true })
  })

  test('H5 移动端：页面应正常加载', async ({ page, baseURL }) => {
    test.skip(!baseURL?.includes('5173'), '仅在 H5 移动端项目运行')

    await page.goto('/')
    await page.waitForTimeout(2000)

    const body = await page.locator('body').textContent()
    expect(body?.length).toBeGreaterThan(0)

    await page.screenshot({ path: 'test-results/e2e-04-h5-fall.png', fullPage: true })
  })
})
