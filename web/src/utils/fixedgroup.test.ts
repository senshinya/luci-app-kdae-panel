import { describe, expect, it } from 'vitest'
import { parseGroups } from './daeconf'
import { parseFixedIndex, resolveFixedCandidates, splitHostPort } from './fixedgroup'
import type { SubscriptionNodeSource } from '../types/api'

function groupOf(content: string, name: string) {
  const group = parseGroups(content).find((candidate) => candidate.name === name)
  if (!group) throw new Error(`测试配置里没有分组 ${name}`)
  return group
}

const nodes = `node {
    hk01: 'vless://u@hk1.example.com:443#HK01'
    hk02: 'vless://u@hk2.example.com:443#HK02'
    jp01: 'vless://u@jp1.example.com:443#JP01'
}
`

function source(tag: string, names: string[], extra: Partial<SubscriptionNodeSource> = {}): SubscriptionNodeSource {
  return {
    tag,
    nodes: names.map((name) => ({ name, protocol: 'vless', host: `${name}.example.com:443`, matches: 1 })),
    cachedAt: '2026-08-04T00:00:00Z',
    ...extra,
  }
}

describe('parseFixedIndex', () => {
  it('只认 fixed(n)', () => {
    expect(parseFixedIndex('fixed(3)')).toBe(3)
    expect(parseFixedIndex('fixed(0)')).toBe(0)
    expect(parseFixedIndex('min_moving_avg')).toBeNull()
    expect(parseFixedIndex(undefined)).toBeNull()
  })
})

describe('resolveFixedCandidates', () => {
  it('本地节点显式列举时按 node 节声明顺序给候选', () => {
    const content = `${nodes}group {
    proxy {
        filter: name(jp01, hk01)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [
        { name: 'hk01', protocol: 'vless', host: 'hk1.example.com', port: 443 },
        { name: 'jp01', protocol: 'vless', host: 'jp1.example.com', port: 443 },
      ],
    })
  })

  it('没有过滤且配置里没有订阅时候选是全部本地节点', () => {
    const content = `${nodes}group {
    proxy {
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [
        { name: 'hk01', protocol: 'vless', host: 'hk1.example.com', port: 443 },
        { name: 'hk02', protocol: 'vless', host: 'hk2.example.com', port: 443 },
        { name: 'jp01', protocol: 'vless', host: 'jp1.example.com', port: 443 },
      ],
    })
  })

  it('没有过滤但存在订阅时不可解', () => {
    const content = `${nodes}subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', ['S1'])], true)
    expect(result.resolvable).toBe(false)
  })

  it('整份订阅按缓存顺序给候选', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', ['S1', 'S2', 'S3'])], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [
        { name: 'S1', protocol: 'vless', host: 'S1.example.com', port: 443 },
        { name: 'S2', protocol: 'vless', host: 'S2.example.com', port: 443 },
        { name: 'S3', protocol: 'vless', host: 'S3.example.com', port: 443 },
      ],
    })
  })

  it('订阅内指定节点按缓存顺序过滤', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub) && name(S3, S1)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', ['S1', 'S2', 'S3'])], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [
        { name: 'S1', protocol: 'vless', host: 'S1.example.com', port: 443 },
        { name: 'S3', protocol: 'vless', host: 'S3.example.com', port: 443 },
      ],
    })
  })

  it('订阅缓存有被跳过的无名节点时不可解', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', ['S1'], { skipped: 2 })], true)
    expect(result.resolvable).toBe(false)
  })

  it('订阅缓存有同名节点时不可解', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    const duplicated = source('mysub', ['S1', 'S2'])
    duplicated.nodes[0].matches = 2
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [duplicated], true)
    expect(result.resolvable).toBe(false)
  })

  it('订阅缓存未加载时不可解', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], false).resolvable).toBe(false)
  })

  it('订阅缓存报错时不可解', () => {
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', [], { problem: '缓存缺失' })], true)
    expect(result.resolvable).toBe(false)
  })

  it('跨两份订阅时不可解', () => {
    const content = `subscription {
    a: 'https://example.com/a'
    b: 'https://example.com/b'
}

group {
    proxy {
        filter: subtag(a, b)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('a', ['A1']), source('b', ['B1'])], true)
    expect(result.resolvable).toBe(false)
  })

  it('本地节点名与订阅节点重名时不可解', () => {
    const content = `${nodes}subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: name(hk01)
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [source('mysub', ['hk01'])], true)
    expect(result.resolvable).toBe(false)
  })

  it('存在订阅但缓存拿不全时本地节点组也不可解', () => {
    const content = `${nodes}subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: name(hk01)
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('过滤引用了不存在的本地节点名时不可解', () => {
    const content = `${nodes}group {
    proxy {
        filter: name(hk01, ghost)
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('候选范围内有无法解析的本地链接时不可解', () => {
    const content = `node {
    hk01: 'vless://u@hk1.example.com:443#HK01'
    broken: 'not-a-link'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('本地节点里有未命名的条目时不可解', () => {
    // 未命名节点是面板本身承认并处理的真实状态（见 NodesCard.vue 的 labelAnonymousNodes），
    // dae 仍会为它建立连接、占用一个下标，只是面板不知道该叫它什么。
    const content = `node {
    'vless://u@hk1.example.com:443#HK01'
    hk02: 'vless://u@hk2.example.com:443#HK02'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('存在未命名的订阅时不可解', () => {
    // 订阅标签是可选的（见 SubscriptionsCard.vue），没写标签的订阅一样会被 dae 拉取节点、
    // 落在面板不知道的某个 tag 里，不能因为面板过滤不出它的 tag 就当它不存在。
    const content = `${nodes}subscription {
    'https://example.com/sub'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('排除条件不可解', () => {
    const content = `${nodes}group {
    proxy {
        filter: !name(jp01)
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('关键词与高级表达式不可解', () => {
    const content = `${nodes}group {
    proxy {
        filter: name(keyword: 'HK')
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('候选为空时不可解', () => {
    const content = `node {
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    expect(resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true).resolvable).toBe(false)
  })

  it('不可解时给出中文原因', () => {
    const content = `${nodes}group {
    proxy {
        filter: name(keyword: 'HK')
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    if (result.resolvable) throw new Error('该用例应不可解')
    expect(result.reason.length).toBeGreaterThan(0)
  })

  it('订阅节点缺协议或主机时仍可解，只是候选字段留空', () => {
    // 判定规则只关心顺序能否对齐，协议/主机是否取得到不该影响可解性。
    const content = `subscription {
    mysub: 'https://example.com/sub'
}

group {
    proxy {
        filter: subtag(mysub)
        policy: fixed(0)
    }
}
`
    const bare: SubscriptionNodeSource = {
      tag: 'mysub',
      cachedAt: '2026-08-04T00:00:00Z',
      nodes: [{ name: 'S1', matches: 1 }],
    }
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [bare], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [{ name: 'S1', protocol: '', host: '', port: null }],
    })
  })

  it('本地节点链接解析不出主机时仍可解，只是候选字段留空', () => {
    const content = `node {
    weird: 'foo://'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    expect(result).toEqual({
      resolvable: true,
      nodes: [{ name: 'weird', protocol: 'foo', host: '', port: null }],
    })
  })

  // Critical 1 回归：readSection 对“一行写两条”或“被跨行块注释穿过”的行只计入
  // unparsedLines、不产 entry；fromLocalNodes 曾经只看 entries，导致把“面板没列全”
  // 误判成“配置里确实没有”，展示一个顺序早已错位的节点名。
  it('本地节点一行写两条时不可解，不能因为少看一条就展示错位的节点名', () => {
    const content = `node {
    hk01: 'vless://u@h1.example.com:443#HK01' hk02: 'vless://u@h2.example.com:443#HK02'
    jp01: 'vless://u@h3.example.com:443#JP01'
}

group {
    proxy {
        policy: fixed(0)
    }
}
`
    // dae 实际看到 hk01/hk02/jp01 三个 dialer，fixed(0) 是 hk01；面板的 entries 解析
    // 只剩 jp01，若忽略 unparsedLines 就会误判为“唯一候选是 jp01”并展示成当前节点。
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    expect(result.resolvable).toBe(false)
  })

  it('订阅一行写多条时不可解，即便按 entries 数看起来配置里没有订阅', () => {
    const content = `${nodes}subscription { a: 'https://example.com/a' b: 'https://example.com/b' }

group {
    proxy {
        policy: fixed(0)
    }
}
`
    // 本地三个节点自身完整可解析，但 subscriptionEntries 会被“一行写两条”坑成空数组，
    // 让 fromLocalNodes 误以为没有订阅，从而展示一份不完整的候选（漏掉两个订阅 tag）。
    const result = resolveFixedCandidates(content, groupOf(content, 'proxy'), [], true)
    expect(result.resolvable).toBe(false)
  })
})

describe('splitHostPort', () => {
  it('IPv6 地址带方括号时拆出裸地址与端口', () => {
    expect(splitHostPort('[::1]:443')).toEqual({ host: '::1', port: 443 })
    expect(splitHostPort('[2001:db8::1]:8443')).toEqual({ host: '2001:db8::1', port: 8443 })
  })

  it('没有端口时端口为 null，主机原样保留', () => {
    expect(splitHostPort('example.com')).toEqual({ host: 'example.com', port: null })
    expect(splitHostPort('[::1]')).toEqual({ host: '::1', port: null })
  })

  it('端口不是数字时不拆分，整串原样作为主机', () => {
    expect(splitHostPort('example.com:abc')).toEqual({ host: 'example.com:abc', port: null })
  })

  it('端口越界时端口为 null，但仍能拆出主机', () => {
    expect(splitHostPort('example.com:0')).toEqual({ host: 'example.com', port: null })
    expect(splitHostPort('example.com:99999')).toEqual({ host: 'example.com', port: null })
  })
})
