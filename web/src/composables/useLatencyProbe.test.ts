import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { MessageApi } from 'naive-ui'
import { postJSON } from '../api/client'
import type { LatencyResult } from '../types/api'
import { probeTarget, useLatencyProbe } from './useLatencyProbe'

vi.mock('../api/client', () => ({ postJSON: vi.fn() }))

function fakeMessage(): MessageApi {
  return {
    warning: vi.fn(),
    error: vi.fn(),
    success: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
    destroyAll: vi.fn(),
  } as unknown as MessageApi
}

describe('probeTarget', () => {
  it('接受合法的主机与端口', () => {
    expect(probeTarget({ host: 'example.com', port: 443 })).toEqual({ host: 'example.com', port: 443 })
  })

  it('拒绝空值、非法端口与含空白/斜杠的主机（对齐后端 netprobe.Target.validate）', () => {
    expect(probeTarget(null)).toBeNull()
    expect(probeTarget(undefined)).toBeNull()
    expect(probeTarget({ host: '', port: 443 })).toBeNull()
    expect(probeTarget({ host: 'example.com', port: null })).toBeNull()
    expect(probeTarget({ host: 'example.com', port: 0 })).toBeNull()
    expect(probeTarget({ host: 'example.com', port: 65536 })).toBeNull()
    expect(probeTarget({ host: 'example.com', port: 1.5 })).toBeNull()
    expect(probeTarget({ host: ' example.com', port: 443 })).toBeNull()
    expect(probeTarget({ host: 'a b', port: 443 })).toBeNull()
    expect(probeTarget({ host: 'a/b', port: 443 })).toBeNull()
    expect(probeTarget({ host: 'a\\b', port: 443 })).toBeNull()
    expect(probeTarget({ host: 'a'.repeat(254), port: 443 })).toBeNull()
  })
})

describe('useLatencyProbe', () => {
  beforeEach(() => {
    vi.mocked(postJSON).mockReset()
  })

  it('未测过时文案为未测，类型为 default，tooltip 为空', () => {
    const { label, type, title } = useLatencyProbe(fakeMessage())
    const source = { host: 'a.example.com', port: 443 }
    expect(label(source)).toBe('未测')
    expect(type(source)).toBe('default')
    expect(title(source)).toBe('')
  })

  it('目标本身无法测量（host/port 非法）时文案为 —，类型为 default', () => {
    const { label, type } = useLatencyProbe(fakeMessage())
    expect(label(null)).toBe('—')
    expect(type({ host: '', port: null })).toBe('default')
    expect(label({ host: 'example.com', port: null })).toBe('—')
  })

  it('阈值分档：<100ms 成功，[100,300)ms 警告，>=300ms 错误', async () => {
    const results: LatencyResult[] = [
      { host: 'fast.example.com', port: 443, reachable: true, latencyMs: 5.2 },
      { host: 'mid.example.com', port: 443, reachable: true, latencyMs: 99 },
      { host: 'warn.example.com', port: 443, reachable: true, latencyMs: 100 },
      { host: 'edge.example.com', port: 443, reachable: true, latencyMs: 299 },
      { host: 'slow.example.com', port: 443, reachable: true, latencyMs: 300 },
    ]
    vi.mocked(postJSON).mockResolvedValueOnce({ results })
    const { probe, label, type } = useLatencyProbe(fakeMessage())
    await probe(results.map((result) => ({ host: result.host, port: result.port })))

    expect(type({ host: 'fast.example.com', port: 443 })).toBe('success')
    expect(label({ host: 'fast.example.com', port: 443 })).toBe('5.2 ms')
    expect(type({ host: 'mid.example.com', port: 443 })).toBe('success')
    expect(type({ host: 'warn.example.com', port: 443 })).toBe('warning')
    expect(label({ host: 'warn.example.com', port: 443 })).toBe('100 ms')
    expect(type({ host: 'edge.example.com', port: 443 })).toBe('warning')
    expect(type({ host: 'slow.example.com', port: 443 })).toBe('error')
    expect(label({ host: 'slow.example.com', port: 443 })).toBe('300 ms')
  })

  it('不可达：tcp 显示不可达，icmp 显示无法测量，类型都是 error', async () => {
    vi.mocked(postJSON).mockResolvedValueOnce({
      results: [
        { host: 'tcp.example.com', port: 443, reachable: false, method: 'tcp', error: '连接被拒绝' },
        { host: 'icmp.example.com', port: 443, reachable: false, method: 'icmp' },
      ] satisfies LatencyResult[],
    })
    const { probe, label, type, title } = useLatencyProbe(fakeMessage())
    await probe([
      { host: 'tcp.example.com', port: 443 },
      { host: 'icmp.example.com', port: 443 },
    ])

    expect(label({ host: 'tcp.example.com', port: 443 })).toBe('不可达')
    expect(type({ host: 'tcp.example.com', port: 443 })).toBe('error')
    expect(title({ host: 'tcp.example.com', port: 443 })).toBe('连接被拒绝')

    expect(label({ host: 'icmp.example.com', port: 443 })).toBe('无法测量')
    expect(type({ host: 'icmp.example.com', port: 443 })).toBe('error')
    expect(title({ host: 'icmp.example.com', port: 443 })).toBe('')
  })

  it('可达时 tooltip 文案区分 ICMP/TCP 并附带解析 IP', async () => {
    vi.mocked(postJSON).mockResolvedValueOnce({
      results: [
        { host: 'icmp.example.com', port: 443, reachable: true, latencyMs: 10, method: 'icmp', resolvedIp: '1.2.3.4' },
        { host: 'tcp.example.com', port: 443, reachable: true, latencyMs: 10, method: 'tcp' },
      ] satisfies LatencyResult[],
    })
    const { probe, title } = useLatencyProbe(fakeMessage())
    await probe([
      { host: 'icmp.example.com', port: 443 },
      { host: 'tcp.example.com', port: 443 },
    ])
    expect(title({ host: 'icmp.example.com', port: 443 })).toBe('ICMP 网络延迟 · 1.2.3.4')
    expect(title({ host: 'tcp.example.com', port: 443 })).toBe('TCP 握手延迟')
  })

  it('没有可探测目标时提示且不发请求，probing 保持 false', async () => {
    const message = fakeMessage()
    const { probe, probing } = useLatencyProbe(message)
    await probe([null, undefined, { host: '', port: null }])
    expect(message.warning).toHaveBeenCalledWith('没有可探测的节点(需要能解析出服务器与端口)')
    expect(postJSON).not.toHaveBeenCalled()
    expect(probing.value).toBe(false)
  })

  it('探测期间 probing 为 true，结束后恢复 false', async () => {
    const message = fakeMessage()
    let release: (value: { results: LatencyResult[] }) => void = () => {}
    vi.mocked(postJSON).mockReturnValueOnce(new Promise((resolve) => { release = resolve }))
    const { probe, probing } = useLatencyProbe(message)
    const pending = probe([{ host: 'a.example.com', port: 443 }])
    await Promise.resolve()
    await Promise.resolve()
    expect(probing.value).toBe(true)
    release({ results: [] })
    await pending
    expect(probing.value).toBe(false)
  })

  it('超过 64 个目标时分两批发送，且请求前会按 host:port 去重', async () => {
    const message = fakeMessage()
    const sources = Array.from({ length: 100 }, (_, index) => ({ host: `h${index}.example.com`, port: 443 }))
    sources.push({ host: 'h0.example.com', port: 443 }) // 重复目标不应多发一份
    vi.mocked(postJSON).mockResolvedValue({ results: [] })
    const { probe } = useLatencyProbe(message)
    await probe(sources)

    expect(postJSON).toHaveBeenCalledTimes(2)
    const [, firstBody] = vi.mocked(postJSON).mock.calls[0]
    const [, secondBody] = vi.mocked(postJSON).mock.calls[1]
    expect((firstBody as { targets: unknown[] }).targets).toHaveLength(64)
    expect((secondBody as { targets: unknown[] }).targets).toHaveLength(36)
  })

  it('逐批发布结果：某一批失败时前面的结果仍然可读到', async () => {
    const message = fakeMessage()
    const sources = Array.from({ length: 65 }, (_, index) => ({ host: `h${index}.example.com`, port: 443 }))
    vi.mocked(postJSON)
      .mockResolvedValueOnce({ results: [{ host: 'h0.example.com', port: 443, reachable: true, latencyMs: 10 }] })
      .mockRejectedValueOnce(new Error('第二批失败'))
    const { probe, label } = useLatencyProbe(message)
    await probe(sources)

    expect(label({ host: 'h0.example.com', port: 443 })).toBe('10 ms')
    expect(message.error).toHaveBeenCalledWith('第二批失败')
  })

  it('请求失败且非 Error 异常时使用默认文案', async () => {
    const message = fakeMessage()
    vi.mocked(postJSON).mockRejectedValueOnce('oops')
    const { probe, probing } = useLatencyProbe(message)
    await probe([{ host: 'a.example.com', port: 443 }])
    expect(message.error).toHaveBeenCalledWith('延迟探测失败')
    expect(probing.value).toBe(false)
  })
})
