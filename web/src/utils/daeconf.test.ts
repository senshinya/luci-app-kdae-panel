import { describe, expect, it } from 'vitest'
import {
  addGroup,
  addRoutingRule,
  appendToSection,
  findSection,
  isQuotable,
  isValidTag,
  maskComments,
  parseEntries,
  parseGroups,
  parseRoutingRules,
  parseSection,
  quote,
  readSectionBody,
  removeGroup,
  removeLine,
  removeRoutingRule,
  replaceLine,
  scanSections,
  setGroupFilter,
  setGroupPolicy,
  setRoutingRule,
  setSectionBody,
  unquote,
} from './daeconf'

const SAMPLE = `global {
  log_level: info
  # tproxy_port: 12345
  tcp_check_url: 'http://cp.cloudflare.com'
}

subscription {
  my_sub: 'https://example.com/sub?token=a#b'
  'https://example.com/plain'
}

node {
  hk-1: 'vmess://eyJhZGQiOiIxLjIuMy40In0='
  node2: 'ss://YWVzOm1pbWk@5.6.7.8:8388#HK%202'
  # disabled: 'trojan://x@y:443'
}

dns {
  upstream {
    alidns: 'udp://dns.alidns.com:53'
  }
  routing {
    request {
      fallback: alidns
    }
  }
}

group {
  proxy {
    filter: subtag(my_sub) && !name(keyword: '过期')
    policy: min_moving_avg
  }
  direct_group {
    policy: fixed(0)
  }
}

routing {
  pname(NetworkManager) -> direct
  dip(224.0.0.0/3, 'ff00::/8') -> direct
  domain(geosite:cn) -> direct # 国内直连
  fallback: proxy
}
`

describe('scanSections', () => {
  it('识别全部顶层节并跳过注释与字符串中的花括号', () => {
    const braced = "global {\n  tcp_check_url: 'http://x/{a}' # } 注释 {\n}\nnode {\n}\n"
    expect(scanSections(braced).map((section) => section.name)).toEqual(['global', 'node'])
    expect(scanSections(SAMPLE).map((section) => section.name)).toEqual([
      'global', 'subscription', 'node', 'dns', 'group', 'routing',
    ])
  })

  it('支持块注释,且 token 内的 # 与 /* 不是注释', () => {
    const blocky = "/* 整段\nnode { 'x': 'ss://a@b:1' }\n注释 */\nglobal {\n  path: config.d/*.dae\n  tag: foo#bar\n}\n"
    expect(scanSections(blocky).map((section) => section.name)).toEqual(['global'])
    const section = scanSections(blocky)[0]
    expect(blocky.slice(section.bodyStart, section.bodyEnd)).toContain('foo#bar')
  })

  it('块注释或 && 之后的 # 处在 token 边界上,开启行注释', () => {
    const afterBlock = 'group {\n  g1 {\n    policy: min /* a */# b {\n  }\n  g2 {\n    policy: random\n  }\n}\n'
    expect(maskComments(afterBlock)).not.toContain('# b {')
    expect(scanSections(afterBlock).map((section) => section.name)).toEqual(['group'])
    expect(parseGroups(afterBlock).map((group) => group.name)).toEqual(['g1', 'g2'])

    const afterOperator = 'group {\n  g1 {\n    filter: subtag(a)&&# note {\n    policy: min\n  }\n  g2 {\n    policy: random\n  }\n}\n'
    expect(parseGroups(afterOperator).map((group) => group.name)).toEqual(['g1', 'g2'])
  })

  it('嵌套节只归属最外层', () => {
    const dns = findSection(SAMPLE, 'dns')!
    expect(SAMPLE.slice(dns.bodyStart, dns.bodyEnd)).toContain('upstream')
  })
})

describe('parseEntries', () => {
  it('解析带 tag、不带 tag 与被注释的条目', () => {
    const section = findSection(SAMPLE, 'node')!
    const entries = parseEntries(SAMPLE, section)
    expect(entries).toHaveLength(2)
    expect(entries[0].tag).toBe('hk-1')
    expect(entries[0].value).toBe('vmess://eyJhZGQiOiIxLjIuMy40In0=')
    expect(entries[1].tag).toBe('node2')
  })

  it('未加引号的裸链接视为无 tag 条目', () => {
    const section = findSection(SAMPLE, 'subscription')!
    const entries = parseEntries(SAMPLE, section)
    expect(entries).toHaveLength(2)
    expect(entries[0].tag).toBe('my_sub')
    expect(entries[1].tag).toBeNull()
    expect(entries[1].value).toBe('https://example.com/plain')
  })

  it('一行多个条目时拒绝解析(不可按行安全编辑)', () => {
    const text = "node {\n  'a': 'ss://x@h:1' 'b': 'ss://y@h:2'\n  'c': 'ss://z@h:3'\n}\n"
    const entries = parseEntries(text, findSection(text, 'node')!)
    expect(entries).toHaveLength(1)
    expect(entries[0].tag).toBe('c')
  })

  it('跨行块注释中的条目不会被解析', () => {
    const text = "node {\n  /* 停用:\n  x: 'ss://a@b:1'\n  */\n  y: 'ss://c@d:2'\n}\n"
    const entries = parseEntries(text, findSection(text, 'node')!)
    expect(entries).toHaveLength(1)
    expect(entries[0].tag).toBe('y')
  })

  it('宽容对待带引号的 tag(dae 只接受裸 ID,但读取时不因此丢条目)', () => {
    const text = "node {\n  'quoted': 'ss://a@b:1'\n}\n"
    expect(parseEntries(text, findSection(text, 'node')!)[0].tag).toBe('quoted')
  })

  it('被跨行块注释穿过的行不作为可编辑条目', () => {
    const opensHere = "node {\n  a: 'ss://x@h:1' /* two\n  line note */\n  b: 'ss://y@h:2'\n}\n"
    expect(parseEntries(opensHere, findSection(opensHere, 'node')!).map((entry) => entry.tag)).toEqual(['b'])

    const closesHere = "node {\n  /* note\n  */ a: 'ss://x@h:1'\n  b: 'ss://y@h:2'\n}\n"
    expect(parseEntries(closesHere, findSection(closesHere, 'node')!).map((entry) => entry.tag)).toEqual(['b'])
  })

  it('引号内换行不会被拆成两个垃圾条目', () => {
    const text = "node {\n  a: 'x\ny'\n  b: 'ss://y@h:2'\n}\n"
    expect(parseEntries(text, findSection(text, 'node')!).map((entry) => entry.tag)).toEqual(['b'])
  })

  it('裸值中的冒号按 dae 词法解释为声明', () => {
    const text = 'subscription {\n  sub.example.com:443/path\n}\n'
    expect(parseEntries(text, findSection(text, 'subscription')!)).toEqual([
      expect.objectContaining({ tag: 'sub.example.com', value: '443/path' }),
    ])
  })
})

describe('parseGroups', () => {
  it('解析分组的 policy 与 filter', () => {
    const groups = parseGroups(SAMPLE)
    expect(groups.map((group) => group.name)).toEqual(['proxy', 'direct_group'])
    expect(groups[0].policy?.value).toBe('min_moving_avg')
    expect(groups[0].filters[0].value).toBe("subtag(my_sub) && !name(keyword: '过期')")
    expect(groups[1].policy?.value).toBe('fixed(0)')
    expect(groups[1].filters).toHaveLength(0)
  })
})

describe('parseRoutingRules', () => {
  it('解析规则与 fallback,忽略行尾注释', () => {
    const rules = parseRoutingRules(SAMPLE)
    expect(rules).toHaveLength(4)
    expect(rules[0]).toMatchObject({
      match: 'pname(NetworkManager)',
      outbound: 'direct',
      isFallback: false,
      editable: true,
    })
    expect(rules[0].lineStart).toBeLessThan(rules[0].lineEnd)
    expect(rules[2].match).toBe('domain(geosite:cn)')
    expect(rules[3]).toMatchObject({ match: 'fallback', outbound: 'proxy', isFallback: true, editable: true })
  })

  it('参数字符串里的 -> 不会切分规则', () => {
    const text = "routing {\n  domain(full: 'weird->name')\n    -> proxy\n  fallback: direct\n}\n"
    expect(parseRoutingRules(text)[0]).toMatchObject({
      match: "domain(full: 'weird->name')",
      outbound: 'proxy',
      isFallback: false,
      editable: false,
    })
  })
})

describe('路由规则改写', () => {
  it('原位替换规则并保留行尾注释', () => {
    const text = 'routing {\n  domain(geosite:gfw) -> proxy # 主规则\n  fallback: direct\n}\n'
    const rule = parseRoutingRules(text)[0]
    const next = setRoutingRule(text, rule, 'dport(443)', 'block')
    expect(next).toBe('routing {\n  dport(443) -> block # 主规则\n  fallback: direct\n}\n')
  })

  it('规则可在普通匹配与 fallback 之间转换', () => {
    const text = 'routing {\n  domain(geosite:gfw) -> proxy\n}\n'
    const fallback = setRoutingRule(text, parseRoutingRules(text)[0], 'fallback', 'direct', true)
    expect(fallback).toBe('routing {\n  fallback: direct\n}\n')
    const ordinary = setRoutingRule(fallback, parseRoutingRules(fallback)[0], 'domain(geosite:cn)', 'proxy', false)
    expect(ordinary).toBe('routing {\n  domain(geosite:cn) -> proxy\n}\n')
  })

  it('删除只移除目标规则', () => {
    const text = 'routing {\n  domain(geosite:gfw) -> proxy\n  fallback: direct\n}\n'
    const next = removeRoutingRule(text, parseRoutingRules(text)[0])
    expect(next).toBe('routing {\n  fallback: direct\n}\n')
  })

  it('新增普通规则插在 fallback 前', () => {
    const text = 'routing {\n  fallback: direct\n}\n'
    const next = addRoutingRule(text, 'domain(geosite:gfw)', 'proxy')
    expect(next).toBe('routing {\n  domain(geosite:gfw) -> proxy\n  fallback: direct\n}\n')
  })

  it('跨行规则只展示，不允许定点改写或删除', () => {
    const text = "routing {\n  domain(full: 'example.com')\n    -> proxy\n  fallback: direct\n}\n"
    const rule = parseRoutingRules(text)[0]
    expect(rule.editable).toBe(false)
    expect(setRoutingRule(text, rule, 'domain(geosite:gfw)', 'direct')).toBe(text)
    expect(removeRoutingRule(text, rule)).toBe(text)
  })
})

describe('节原文编辑', () => {
  it('读取和替换目标节，不改动其他节', () => {
    const text = 'global {\n  log_level: info\n}\nrouting {\n  fallback: direct\n}\nnode {\n  a: ss://x\n}\n'
    expect(readSectionBody(text, 'routing')).toBe('  fallback: direct')
    const next = setSectionBody(text, 'routing', '  domain(geosite:gfw) -> proxy\n  fallback: direct')
    expect(next).toBe(
      'global {\n  log_level: info\n}\nrouting {\n  domain(geosite:gfw) -> proxy\n  fallback: direct\n}\nnode {\n  a: ss://x\n}\n',
    )
  })

  it('节缺失时创建，并沿用 CRLF', () => {
    const text = 'global {\r\n}\r\n'
    const next = setSectionBody(text, 'routing', '  fallback: direct\n')
    expect(next).toBe('global {\r\n}\r\nrouting {\r\n  fallback: direct\r\n}\r\n')
    expect(next).not.toMatch(/[^\r]\n/)
  })
})

describe('跨行或不闭合的分组声明只展示不改写', () => {
  it('多行 filter 标记为不可编辑,改写请求被忽略', () => {
    const text = 'group {\n  g {\n    filter: subtag(a) &&\n      !name(keyword: x)\n    policy: min\n  }\n}\n'
    const group = parseGroups(text)[0]
    expect(group.filters[0]).toMatchObject({ value: 'subtag(a) &&', editable: false })
    expect(setGroupFilter(text, group, 0, 'name(keyword: HK)')).toBe(text)
    expect(setGroupFilter(text, group, 0, '')).toBe(text)
  })

  it('续行运算符写在次行行首时同样不可编辑', () => {
    const text = 'group {\n  g {\n    filter: subtag(a)\n      && !name(keyword: HK)\n    policy: min\n  }\n}\n'
    const group = parseGroups(text)[0]
    expect(group.filters[0]).toMatchObject({ value: 'subtag(a)', editable: false })
    expect(setGroupFilter(text, group, 0, 'name(keyword: US)')).toBe(text)
    expect(setGroupFilter(text, group, 0, '')).toBe(text)
    expect(group.policy?.editable).toBe(true)
  })

  it('分组级覆盖键不会误伤上一条声明', () => {
    const text = "group {\n  g {\n    filter: subtag(a)\n    tcp_check_url: 'http://x'\n    policy: min\n  }\n}\n"
    const group = parseGroups(text)[0]
    expect(group.filters[0].editable).toBe(true)
    expect(group.policy?.editable).toBe(true)
  })

  it('括号闭合的单行 filter 仍可编辑', () => {
    const text = "group {\n  g {\n    filter: subtag(a) && !name(keyword: '过期')\n  }\n}\n"
    const group = parseGroups(text)[0]
    expect(group.filters[0].editable).toBe(true)
    expect(setGroupFilter(text, group, 0, 'name(keyword: HK)')).toContain('filter: name(keyword: HK)')
  })
})

describe('同一行的分组声明按独立范围编辑', () => {
  it('修改 policy 不会吞掉同一行后面的 filter', () => {
    const text = 'group {\n  proxy { policy: fixed(0) filter: name(a) }\n}\n'
    const group = parseGroups(text)[0]

    expect(group.policy).toMatchObject({ value: 'fixed(0)', editable: true })
    expect(group.filters).toHaveLength(1)
    expect(group.filters[0]).toMatchObject({ value: 'name(a)', editable: true })
    expect(setGroupPolicy(text, group, 'random')).toBe(
      'group {\n  proxy { policy: random filter: name(a) }\n}\n',
    )
  })

  it('替换或删除 filter 不会吞掉同一行前面的 policy', () => {
    const text = 'group {\n  proxy { policy: fixed(0) filter: name(a) }\n}\n'
    const group = parseGroups(text)[0]

    expect(setGroupFilter(text, group, 0, 'name(b)')).toBe(
      'group {\n  proxy { policy: fixed(0) filter: name(b) }\n}\n',
    )
    const removed = setGroupFilter(text, group, 0, '')
    expect(removed).toContain('policy: fixed(0)')
    expect(removed).not.toContain('filter: name(a)')
  })

  it('行尾注释不属于 policy 值', () => {
    const text = 'group {\n  proxy {\n    policy: fixed(0) # filter: name(fake)\n  }\n}\n'
    const group = parseGroups(text)[0]
    const next = setGroupPolicy(text, group, 'random')

    expect(group.policy).toMatchObject({ value: 'fixed(0)', editable: true })
    expect(next).toContain('policy: random # filter: name(fake)')
  })

  it('未知分组键也会隔开相邻声明', () => {
    const text = "group {\n  proxy { policy: fixed(0) tcp_check_url: 'http://x' filter: name(a) }\n}\n"
    const group = parseGroups(text)[0]

    expect(group.policy).toMatchObject({ value: 'fixed(0)', editable: true })
    expect(group.filters[0]).toMatchObject({ value: 'name(a)', editable: true })
    expect(setGroupPolicy(text, group, 'random')).toContain(
      "policy: random tcp_check_url: 'http://x' filter: name(a)",
    )
    expect(setGroupFilter(text, group, 0, 'name(b)')).toContain(
      "policy: fixed(0) tcp_check_url: 'http://x' filter: name(b)",
    )
  })

  it('无法确认 policy 边界时拒绝结构化修改', () => {
    const text = 'group {\n  proxy { policy: fixed(0) unknown filter: name(a) }\n}\n'
    const group = parseGroups(text)[0]

    expect(group.policy?.editable).toBe(false)
    expect(setGroupPolicy(text, group, 'random')).toBe(text)
  })
})

describe('parseSection 的未解析行计数', () => {
  it('跨行与多条目写法都被计入而不是静默消失', () => {
    const text = [
      'node {',
      "  a: 'ss://x@h:1'",
      "  b: 'ss://y@h:2' c: 'ss://z@h:3'", // 一行两条
      "  d: 'multi",
      "line'", // 跨行字符串
      '  e:', // 悬挂声明,值缺失
      '}',
      '',
    ].join('\n')
    const parsed = parseSection(text, findSection(text, 'node')!)
    expect(parsed.entries.map((entry) => entry.tag)).toEqual(['a'])
    expect(parsed.unparsedLines).toBe(4)
  })

  it('规整配置没有未解析行', () => {
    expect(parseSection(SAMPLE, findSection(SAMPLE, 'node')!).unparsedLines).toBe(0)
    expect(parseSection(SAMPLE, findSection(SAMPLE, 'subscription')!).unparsedLines).toBe(0)
  })
})

describe('重复 policy 声明', () => {
  it('哪条生效由 dae 决定,面板一律降级为只读', () => {
    const text = 'group {\n  g {\n    policy: min\n    policy: random\n  }\n}\n'
    const group = parseGroups(text)[0]
    expect(group.policy).toMatchObject({ value: 'min', editable: false })
    expect(setGroupPolicy(text, group, 'fixed(0)')).toBe(text)
  })
})

describe('悬挂的 tag: 声明', () => {
  it('值在下一行的条目按跨行处理,不可编辑', () => {
    const text = "node {\n  b:\n  'ss://x@h:1'\n  c: 'ss://y@h:2'\n}\n"
    const entries = parseEntries(text, findSection(text, 'node')!)
    expect(entries).toEqual([
      expect.objectContaining({ tag: 'b', value: 'ss://x@h:1', editable: false }),
      expect.objectContaining({ tag: 'c', value: 'ss://y@h:2', editable: true }),
    ])
  })

  it('普通条目仍标记为可编辑', () => {
    const entries = parseEntries(SAMPLE, findSection(SAMPLE, 'node')!)
    expect(entries.every((entry) => entry.editable)).toBe(true)
  })
})

describe('isQuotable', () => {
  it('同时含单双引号或含换行的值不可无损写回', () => {
    expect(isQuotable('ss://x@h:1#a\'b"c')).toBe(false)
    expect(isQuotable('多\n行')).toBe(false)
    for (const value of ["含'单引号", '含"双引号', '普通值']) {
      expect(isQuotable(value)).toBe(true)
      expect(unquote(quote(value))).toBe(value)
    }
  })
})

describe('改写操作', () => {
  it('appendToSection 追加到已有节且不动其他内容', () => {
    const next = appendToSection(SAMPLE, 'node', ["'new': 'trojan://a@b:443'"])
    const entries = parseEntries(next, findSection(next, 'node')!)
    expect(entries).toHaveLength(3)
    expect(entries[2].tag).toBe('new')
    expect(next.replace("  'new': 'trojan://a@b:443'\n", '')).toBe(SAMPLE)
  })

  it('appendToSection 在节缺失时创建', () => {
    const next = appendToSection('global {\n}\n', 'node', ["'a': 'ss://x@y:1'"])
    expect(findSection(next, 'node')).not.toBeNull()
    expect(parseEntries(next, findSection(next, 'node')!)).toHaveLength(1)
  })

  it('replaceLine 原位改写条目(如打标签)', () => {
    const section = findSection(SAMPLE, 'subscription')!
    const untagged = parseEntries(SAMPLE, section)[1]
    const next = replaceLine(SAMPLE, untagged.lineStart, untagged.lineEnd, "plain: 'https://example.com/plain'")
    const entries = parseEntries(next, findSection(next, 'subscription')!)
    expect(entries[1]).toMatchObject({ tag: 'plain', value: 'https://example.com/plain' })
  })

  it('replaceLine 保留行尾注释', () => {
    const text = "node {\n  a: 'ss://x@h:1' # 备用节点\n}\n"
    const entry = parseEntries(text, findSection(text, 'node')!)[0]
    const next = replaceLine(text, entry.lineStart, entry.lineEnd, "b: 'ss://x@h:1'")
    expect(next).toBe("node {\n  b: 'ss://x@h:1' # 备用节点\n}\n")
  })

  it('replaceLine 在无注释时不引入多余空白', () => {
    const text = "node {\n  a: 'ss://x@h:1'\n}\n"
    const entry = parseEntries(text, findSection(text, 'node')!)[0]
    expect(replaceLine(text, entry.lineStart, entry.lineEnd, "b: 'ss://x@h:1'"))
      .toBe("node {\n  b: 'ss://x@h:1'\n}\n")
  })

  it('removeLine 精确移除条目行', () => {
    const section = findSection(SAMPLE, 'node')!
    const entry = parseEntries(SAMPLE, section)[0]
    const next = removeLine(SAMPLE, entry.lineStart, entry.lineEnd)
    expect(next).not.toContain('hk-1')
    expect(next).toContain('node2')
    expect(next).toContain('# disabled')
  })

  it('setGroupPolicy 原位替换并保留注释', () => {
    const withComment = SAMPLE.replace('policy: min_moving_avg', 'policy: min_moving_avg # 首选')
    const group = parseGroups(withComment)[0]
    const next = setGroupPolicy(withComment, group, 'random')
    expect(next).toContain('policy: random # 首选')
  })

  it('setGroupPolicy 在缺失时插入', () => {
    const text = 'group {\n  bare {\n  }\n}\n'
    const next = setGroupPolicy(text, parseGroups(text)[0], 'random')
    expect(parseGroups(next)[0].policy?.value).toBe('random')
  })

  it('setGroupFilter 支持替换、插入与清除', () => {
    const groups = parseGroups(SAMPLE)
    const replaced = setGroupFilter(SAMPLE, groups[0], 0, 'name(keyword: HK)')
    expect(parseGroups(replaced)[0].filters[0].value).toBe('name(keyword: HK)')

    const inserted = setGroupFilter(SAMPLE, groups[1], 0, 'subtag(my_sub)')
    expect(parseGroups(inserted)[1].filters[0].value).toBe('subtag(my_sub)')

    const cleared = setGroupFilter(SAMPLE, groups[0], 0, '')
    expect(parseGroups(cleared)[0].filters).toHaveLength(0)
  })

  it('追加的过滤排在已有声明之后,顺序与界面一致', () => {
    const groups = parseGroups(SAMPLE)
    const appended = setGroupFilter(SAMPLE, groups[0], groups[0].filters.length, 'name(keyword: US)')
    const filters = parseGroups(appended)[0].filters
    expect(filters.map((filter) => filter.value)).toEqual([
      "subtag(my_sub) && !name(keyword: '过期')",
      'name(keyword: US)',
    ])
    expect(parseGroups(appended)[0].policy?.value).toBe('min_moving_avg')
  })

  it('对没有任何声明的分组,插入仍然成立', () => {
    const text = 'group {\n  bare {\n  }\n}\n'
    const withFilter = setGroupFilter(text, parseGroups(text)[0], 0, 'subtag(s)')
    expect(parseGroups(withFilter)[0].filters[0].value).toBe('subtag(s)')
    const withBoth = setGroupPolicy(withFilter, parseGroups(withFilter)[0], 'min')
    const group = parseGroups(withBoth)[0]
    expect([group.policy?.value, group.filters[0].value]).toEqual(['min', 'subtag(s)'])
  })

  it('removeGroup 不波及同一行上的相邻分组', () => {
    const text = 'group {\n  a { policy: min } proxy {\n    policy: random\n  }\n}\n'
    const groups = parseGroups(text)
    expect(groups.map((group) => group.name)).toEqual(['a', 'proxy'])
    const next = removeGroup(text, groups[1])
    expect(next).toContain('a { policy: min }')
    expect(parseGroups(next).map((group) => group.name)).toEqual(['a'])
  })

  it('独占整行的分组连同行尾一起删除', () => {
    const text = 'group {\n  a {\n    policy: min\n  }\n  b {\n    policy: random\n  }\n}\n'
    expect(removeGroup(text, parseGroups(text)[0])).toBe('group {\n  b {\n    policy: random\n  }\n}\n')
  })

  it('addGroup 与 removeGroup 往返', () => {
    const added = addGroup(SAMPLE, 'us_group', 'min')
    const groups = parseGroups(added)
    expect(groups.map((group) => group.name)).toContain('us_group')
    expect(groups.find((group) => group.name === 'us_group')?.policy?.value).toBe('min')

    const removed = removeGroup(added, groups.find((group) => group.name === 'us_group')!)
    expect(parseGroups(removed).map((group) => group.name)).toEqual(['proxy', 'direct_group'])
  })

  it('addGroup 在 group 节缺失时创建整个节', () => {
    const next = addGroup('global {\n}\n', 'proxy', 'min_moving_avg')
    expect(parseGroups(next)[0]?.name).toBe('proxy')
    expect(parseGroups(next)[0]?.policy?.value).toBe('min_moving_avg')
  })
})

describe('quote/unquote', () => {
  it('quote 选用值中不存在的引号类型(dae 不做反转义)', () => {
    expect(quote('plain')).toBe("'plain'")
    expect(quote("带'单引号")).toBe('"带\'单引号"')
    expect(quote('带"双引号')).toBe("'带\"双引号'")
    for (const value of ['plain', "带'单引号", '带"双引号', '反斜杠\\path', '空 格,冒号:']) {
      expect(unquote(quote(value))).toBe(value)
    }
  })

  it('unquote 与 dae 一致:只剥外层引号,不做反转义', () => {
    expect(unquote("'a\\'b'")).toBe("a\\'b")
    expect(unquote('"double"')).toBe('double')
    expect(unquote('bare')).toBe('bare')
  })
})

describe('CRLF 配置', () => {
  const CRLF = "node {\r\n  'a': 'ss://x@h:1'\r\n  'b': 'ss://y@h:2'\r\n}\r\n"

  it('解析与删除保持 CRLF 不变', () => {
    const entries = parseEntries(CRLF, findSection(CRLF, 'node')!)
    expect(entries.map((entry) => entry.tag)).toEqual(['a', 'b'])
    const removed = removeLine(CRLF, entries[0].lineStart, entries[0].lineEnd)
    expect(removed).toBe("node {\r\n  'b': 'ss://y@h:2'\r\n}\r\n")
  })

  it('改写与追加沿用 CRLF', () => {
    const entries = parseEntries(CRLF, findSection(CRLF, 'node')!)
    expect(replaceLine(CRLF, entries[0].lineStart, entries[0].lineEnd, "t: 'ss://x@h:1'"))
      .toBe("node {\r\n  t: 'ss://x@h:1'\r\n  'b': 'ss://y@h:2'\r\n}\r\n")
    expect(appendToSection(CRLF, 'node', ["'c': 'ss://z@h:3'"]))
      .toBe("node {\r\n  'a': 'ss://x@h:1'\r\n  'b': 'ss://y@h:2'\r\n  'c': 'ss://z@h:3'\r\n}\r\n")
    expect(appendToSection(CRLF, 'group', ['p {', '  policy: min', '}'])).toContain('\r\ngroup {\r\n')
  })

  it('分组声明插入不产生 \\r\\r\\n 或混合行尾', () => {
    const text = 'group {\r\n  proxy {\r\n    policy: min_moving_avg\r\n  }\r\n}\r\n'
    const withFilter = setGroupFilter(text, parseGroups(text)[0], 0, 'name(keyword: HK)')
    expect(withFilter).toBe(
      'group {\r\n  proxy {\r\n    policy: min_moving_avg\r\n    filter: name(keyword: HK)\r\n  }\r\n}\r\n',
    )
    // 反复插入不应逐步退化成 LF
    const twice = setGroupFilter(withFilter, parseGroups(withFilter)[0], 1, 'subtag(s)')
    expect(twice).not.toContain('\r\r')
    expect(twice).not.toMatch(/[^\r]\n/)

    const bare = 'group {\r\n  proxy {\r\n  }\r\n}\r\n'
    const withPolicy = setGroupPolicy(bare, parseGroups(bare)[0], 'min')
    expect(withPolicy).not.toContain('\r\r')
    expect(withPolicy).not.toMatch(/[^\r]\n/)
    expect(parseGroups(withPolicy)[0].policy?.value).toBe('min')
  })
})

describe('isValidTag', () => {
  it('只接受词法安全的裸 ID', () => {
    for (const valid of ['my_sub', 'HK.01', 'node-2', '_x']) expect(isValidTag(valid)).toBe(true)
    for (const invalid of ['1st', '香港', 'a b', "quo'te", 'x:y', '']) expect(isValidTag(invalid)).toBe(false)
  })
})

describe('异常输入不应崩溃或误吞内容', () => {
  it('未闭合引号', () => {
    const text = "node {\n  a: 'unterminated\n  b: 'ss://x@h:1'\n}\n"
    expect(() => scanSections(text)).not.toThrow()
    expect(maskComments(text).length).toBe(text.length)
  })

  it('未闭合花括号', () => {
    const text = "global {\n  log_level: info\nnode {\n  a: 'ss://x@h:1'\n"
    expect(() => scanSections(text)).not.toThrow()
    expect(scanSections(text)).toEqual([])
  })

  it('未闭合块注释', () => {
    const text = "node {\n  a: 'ss://x@h:1'\n}\n/* 未闭合\nglobal { log_level: info }\n"
    expect(scanSections(text).map((section) => section.name)).toEqual(['node'])
    expect(maskComments(text).length).toBe(text.length)
  })

  it('空文本与仅注释', () => {
    expect(scanSections('')).toEqual([])
    expect(scanSections('# 只有注释\n')).toEqual([])
    expect(parseGroups('')).toEqual([])
    expect(findSection('', 'node')).toBeNull()
  })

  it('掩码保持长度与换行位置,偏移可直接用于原文', () => {
    const text = "node {\n  a: 'x' # 注释\n  /* 块 */ b: 'y'\n}\n"
    const masked = maskComments(text)
    expect(masked.length).toBe(text.length)
    for (let index = 0; index < text.length; index += 1) {
      if (text[index] === '\n') expect(masked[index]).toBe('\n')
    }
    const entries = parseEntries(text, findSection(text, 'node')!)
    for (const entry of entries) {
      expect(text.slice(entry.lineStart, entry.lineEnd)).toContain(entry.value)
    }
  })

  it('深层嵌套只暴露顶层节', () => {
    const text = 'dns {\n  routing {\n    request {\n      fallback: a\n    }\n  }\n}\n'
    expect(scanSections(text).map((section) => section.name)).toEqual(['dns'])
  })
})
