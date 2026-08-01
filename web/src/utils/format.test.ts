import { describe, expect, it } from 'vitest'
import { formatBytes, formatElapsedSince } from './format'

describe('formatBytes', () => {
  it('允许运行指标保留固定精度', () => {
    expect(formatBytes(150.74 * 1024 * 1024)).toBe('151 MiB')
    expect(formatBytes(150.74 * 1024 * 1024, 1)).toBe('150.7 MiB')
  })
})

describe('formatElapsedSince', () => {
  it('按本次启动时间计算持续时长', () => {
    const startedAt = '2026-08-01T10:00:00Z'
    expect(formatElapsedSince(startedAt, Date.parse('2026-08-01T10:00:09Z'))).toBe('9 秒')
    expect(formatElapsedSince(startedAt, Date.parse('2026-08-01T11:02:03Z'))).toBe('1 小时 2 分')
    expect(formatElapsedSince(startedAt, Date.parse('2026-08-03T13:00:00Z'))).toBe('2 天 3 小时')
  })

  it('拒绝缺失、非法或未来的启动时间', () => {
    const now = Date.parse('2026-08-01T10:00:00Z')
    expect(formatElapsedSince(undefined, now)).toBe('—')
    expect(formatElapsedSince('不是时间', now)).toBe('—')
    expect(formatElapsedSince('2026-08-01T10:00:01Z', now)).toBe('—')
  })
})
