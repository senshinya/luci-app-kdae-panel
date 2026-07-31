import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useBackendStore } from './backend'

function mockHealth(response: unknown, ok = true) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(response), {
    status: ok ? 200 : 503,
    headers: { 'Content-Type': 'application/json' },
  }))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('useBackendStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.unstubAllGlobals()
  })

  it('把 health 的 backend 字段记下来', async () => {
    mockHealth({ status: 'ok', backend: 'procd' })
    const store = useBackendStore()
    await store.ensure()
    expect(store.backend).toBe('procd')
    expect(store.isProcd).toBe(true)
  })

  // 未知取值必须落到 systemd 那套措辞，而不是被当成 procd：
  // procd 分支的文案会承诺"启动脚本不会删除"，在 systemd 上说这句是错的。
  it('backend 缺失或不认识时按 systemd 处理', async () => {
    mockHealth({ status: 'ok' })
    const store = useBackendStore()
    await store.ensure()
    expect(store.backend).toBe('systemd')
    expect(store.isProcd).toBe(false)
  })

  // 后端只影响文案，读不到不该让页面崩掉；下次进页面还要能再试。
  it('请求失败时不抛错，且保留重试机会', async () => {
    mockHealth({ error: { message: '挂了' } }, false)
    const store = useBackendStore()
    await store.ensure()
    expect(store.backend).toBe('')
    expect(store.isProcd).toBe(false)

    const retry = mockHealth({ status: 'ok', backend: 'procd' })
    await store.ensure()
    expect(retry).toHaveBeenCalledTimes(1)
    expect(store.backend).toBe('procd')
  })

  // 一个页面里父子组件各调一次很常见，同一个事实不该查两遍。
  it('并发调用只发一次请求', async () => {
    const fetchMock = mockHealth({ status: 'ok', backend: 'systemd' })
    const store = useBackendStore()
    await Promise.all([store.ensure(), store.ensure(), store.ensure()])
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('已经知道后端时不再请求', async () => {
    mockHealth({ status: 'ok', backend: 'procd' })
    const store = useBackendStore()
    await store.ensure()
    const again = mockHealth({ status: 'ok', backend: 'systemd' })
    await store.ensure()
    expect(again).not.toHaveBeenCalled()
    expect(store.backend).toBe('procd')
  })
})
