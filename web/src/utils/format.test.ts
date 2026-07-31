import { describe, expect, it } from 'vitest'
import { formatUptime } from './format'

describe('formatUptime', () => {
  it('缺失或非法输入显示破折号而不是 0 秒', () => {
    // procd 读不到 /proc 时报 0，systemd 后端则根本不填这个字段。
    // 两者都必须显示"不知道"，"0 秒"会被读成"刚刚重启过"。
    expect(formatUptime(undefined)).toBe('—')
    expect(formatUptime(Number.NaN)).toBe('—')
    expect(formatUptime(-1)).toBe('—')
  })

  it('按量级选单位，天数不会被折算成上千小时', () => {
    expect(formatUptime(42)).toBe('42 秒')
    expect(formatUptime(90)).toBe('1 分')
    expect(formatUptime(3600 + 120)).toBe('1 小时 2 分')
    expect(formatUptime(86400 * 3 + 3600 * 5)).toBe('3 天 5 小时')
  })
})
