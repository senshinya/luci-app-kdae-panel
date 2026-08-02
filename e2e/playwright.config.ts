import { defineConfig } from '@playwright/test'

// 端口避开面板默认的 2023：本地开着开发实例也能跑 E2E。
const PORT = 21323

export default defineConfig({
  testDir: '.',
  testMatch: 'panel.spec.ts',
  timeout: 60_000,
  // 步骤间共享同一份磁盘状态（账号、配置文件），并行毫无意义
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    viewport: { width: 1600, height: 900 },
    // Windows 默认复用系统 Edge，Linux CI 继续使用流水线安装的 Chromium。
    channel: process.env.E2E_BROWSER_CHANNEL || (process.platform === 'win32' ? 'msedge' : undefined),
  },
  webServer: {
    command: 'node start-panel.mjs',
    url: `http://127.0.0.1:${PORT}/api/v1/health`,
    reuseExistingServer: false,
    timeout: 30_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
})
