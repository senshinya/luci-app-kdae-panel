import { describe, expect, it } from 'vitest'
import { setSectionBody } from './daeconf'
import {
  buildDNSBody,
  defaultDNSDraft,
  defaultConfiguration,
  readDNSCapabilities,
  readDNSState,
  withDefaultDNS,
} from './dns'

const COMPLETE = `global {}

dns {
  ipversion_prefer: 4
  fixed_domain_ttl {
    ddns.example.org: 10
    'test.example.org': 3600
  }
  optimistic_cache: true
  optimistic_cache_ttl: 60
  max_cache_size: 65536
  bind: 'tcp+udp://127.0.0.1:5353'
  upstream {
    alidns: 'udp://223.5.5.5:53'
    googledns: 'tcp+udp://8.8.8.8:53'
  }
  routing {
    request {
      qname(geosite:cn) -> alidns
      fallback: googledns
    }
    response {
      upstream(googledns) -> accept
      ip(geoip:private) && !qname(geosite:cn) -> googledns
      fallback: accept
    }
  }
}
`

describe('DNS 配置解析与生成', () => {
  it('覆盖 dae 的全部公开 DNS 字段并可按语义往返', () => {
    const state = readDNSState(COMPLETE)
    expect(state.simpleSafe).toBe(true)
    expect(state.draft).toMatchObject({
      ipVersionPrefer: '4',
      bind: 'tcp+udp://127.0.0.1:5353',
      optimisticCache: true,
      optimisticCacheTTL: 60,
      maxCacheSize: 65536,
    })
    expect(state.draft.fixedTTLs.map(({ domain, ttl }) => ({ domain, ttl }))).toEqual([
      { domain: 'ddns.example.org', ttl: 10 },
      { domain: 'test.example.org', ttl: 3600 },
    ])
    expect(state.draft.upstreams.map(({ name, url }) => ({ name, url }))).toEqual([
      { name: 'alidns', url: 'udp://223.5.5.5:53' },
      { name: 'googledns', url: 'tcp+udp://8.8.8.8:53' },
    ])
    expect(state.draft.requestRules).toMatchObject([
      { matcher: 'qname(geosite:cn)', target: 'alidns', fallback: false },
      { matcher: '', target: 'googledns', fallback: true },
    ])
    expect(state.draft.responseRules).toHaveLength(3)

    const rebuilt = setSectionBody(COMPLETE, 'dns', buildDNSBody(state.draft))
    expect(rebuilt).toContain('    ddns.example.org: 10')
    expect(rebuilt).not.toContain("'ddns.example.org': 10")
    const reparsed = readDNSState(rebuilt)
    expect(reparsed.simpleSafe).toBe(true)
    expect(buildDNSBody(reparsed.draft)).toBe(buildDNSBody(state.draft))
  })

  it('无 DNS 时提供 daed 默认草稿，但不伪装成已经配置', () => {
    const state = readDNSState('global {}\n')
    expect(state.present).toBe(false)
    expect(state.draft.upstreams).toHaveLength(0)
    const draft = defaultDNSDraft()
    expect(draft.upstreams.map((item) => item.name)).toEqual(['alidns', 'googledns'])
    expect(draft.responseRules).toHaveLength(2)
    expect(buildDNSBody(draft)).toContain('qname(geosite:cn) -> alidns')
  })

  it('旧配置缺少 DNS 时生成待保存默认草稿，重复处理不追加第二份', () => {
    const old = 'global {}\n\nrouting {}\n'
    const migrated = withDefaultDNS(old)
    expect(migrated).toContain('dns {')
    expect(readDNSState(migrated).draft.upstreams).toHaveLength(2)
    expect(withDefaultDNS(migrated)).toBe(migrated)

    const fresh = defaultConfiguration()
    expect(fresh.indexOf('global {}')).toBeLessThan(fresh.indexOf('dns {'))
    expect(fresh.indexOf('dns {')).toBeLessThan(fresh.lastIndexOf('routing {}'))
  })

  it('注释、未知字段与未知块会降级到进阶模式', () => {
    const state = readDNSState(`dns {
  # 必须保留
  future_cache: true
  custom_block { value: one }
  upstream { a: 'udp://1.1.1.1:53' }
}`)
    expect(state.simpleSafe).toBe(false)
    expect(state.issues.join('；')).toContain('注释')
    expect(state.issues.join('；')).toContain('future_cache')
    expect(state.issues.join('；')).toContain('custom_block')
  })

  it('重复块和跨行规则不会被误判为可无损往返', () => {
    const state = readDNSState(`dns {
  upstream { a: 'udp://1.1.1.1:53' }
  upstream { b: 'udp://8.8.8.8:53' }
  routing {
    request {
      qname(geosite:cn) &&
        qtype(a) -> a
      fallback: a
    }
  }
}`)
    expect(state.simpleSafe).toBe(false)
    expect(state.issues.join('；')).toContain('upstream 块重复')
    expect(state.issues.join('；')).toContain('跨行')
  })

  it('拒绝重复上游、悬空目标和多个 fallback', () => {
    const duplicate = defaultDNSDraft()
    duplicate.upstreams[1].name = 'alidns'
    expect(() => buildDNSBody(duplicate)).toThrow('重复')

    const dangling = defaultDNSDraft()
    dangling.requestRules[0].target = 'missing'
    expect(() => buildDNSBody(dangling)).toThrow('不是内置动作或现有上游')

    const fallbacks = defaultDNSDraft()
    fallbacks.requestRules[0].fallback = true
    expect(() => buildDNSBody(fallbacks)).toThrow('只能有一个 fallback')
  })

  it('拒绝从匹配输入注入第二条规则、配置块或注释', () => {
    for (const matcher of [
      'qname(geosite:cn) -> reject',
      'qname(geosite:cn) { fallback: reject }',
      'qname(geosite:cn) # 隐藏后续出站',
      'qname(geosite:cn',
    ]) {
      const draft = defaultDNSDraft()
      draft.requestRules[0].matcher = matcher
      expect(() => buildDNSBody(draft), matcher).toThrow('不完整或不安全')
    }
  })

  it('固定 TTL 域名只生成 dae 接受的裸声明键', () => {
    const draft = defaultDNSDraft()
    draft.fixedTTLs.push({ id: 'unsafe', domain: "bad'domain", ttl: 60 })
    expect(() => buildDNSBody(draft)).toThrow('不能作为 dae 声明键')
  })

  it('写回 CRLF 配置时不混入 LF', () => {
    const crlf = "global {}\r\ndns {\r\n  upstream {\r\n    a: 'udp://1.1.1.1:53'\r\n  }\r\n}\r\n"
    const state = readDNSState(crlf)
    const next = setSectionBody(crlf, 'dns', buildDNSBody(state.draft))
    expect(next).not.toMatch(/[^\r]\n/)
    expect(readDNSState(next).draft.upstreams[0].name).toBe('a')
  })
})

describe('DNS outline 能力', () => {
  it('读取当前二进制支持字段和实际默认值', () => {
    const capabilities = readDNSCapabilities({
      version: 'v1.0.6',
      leaves: [],
      structure: [{
        name: 'Dns',
        mapping: 'dns',
        structure: [
          { mapping: 'upstream' },
          { mapping: 'optimistic_cache', defaultValue: 'true' },
          { mapping: 'max_cache_size', defaultValue: '65536' },
        ],
      }],
    })
    expect(capabilities?.supported).toEqual(new Set(['upstream', 'optimistic_cache', 'max_cache_size']))
    expect(capabilities?.defaults.get('max_cache_size')).toBe('65536')
  })

  it('outline 缺少 DNS 结构时返回 null，不误判全部字段不支持', () => {
    expect(readDNSCapabilities({ version: 'old', leaves: [], structure: [] })).toBeNull()
  })
})
