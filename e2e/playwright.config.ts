import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,       // 串行执行，确保设备发包顺序
  retries: 0,
  workers: 1,
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
  ],
  use: {
    headless: true,
    screenshot: 'on',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'cms-dashboard',
      use: {
        baseURL: 'http://localhost:5174',
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'h5-mobile',
      use: {
        baseURL: 'http://localhost:5173',
        viewport: { width: 390, height: 844 },   // iPhone 14 尺寸
      },
    },
  ],
})
