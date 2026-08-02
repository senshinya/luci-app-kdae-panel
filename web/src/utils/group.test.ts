import { describe, expect, it } from 'vitest'
import {
  createGroupFilter,
  describeGroupFilter,
  includeNodesInGroups,
  knownFixedCandidateCount,
  parseGroupFilter,
  serializeGroupFilter,
} from './group'

describe('分组资源过滤', () => {
  it('解析精确节点与订阅多选', () => {
    expect(parseGroupFilter("name(node1, '香港 01')")).toMatchObject({
      kind: 'nodes', values: ['node1', '香港 01'], exclude: false,
    })
    expect(parseGroupFilter('!subtag(sub_a, sub-b)')).toMatchObject({
      kind: 'subscriptions', values: ['sub_a', 'sub-b'], exclude: true,
    })
    expect(parseGroupFilter("subtag(main) && name('香港 01', sg_02)")).toMatchObject({
      kind: 'subscriptionNodes', source: 'main', values: ['香港 01', 'sg_02'], exclude: false,
    })
  })

  it('关键词、正则和复杂表达式不会被误判为资源多选', () => {
    expect(parseGroupFilter("name(keyword: 'HK')")).toMatchObject({ kind: 'nameKeyword', value: 'HK' })
    expect(parseGroupFilter("name(regex: '^HK|SG$')")).toMatchObject({ kind: 'nameRegex', value: '^HK|SG$' })
    expect(parseGroupFilter("subtag(regex: '^my_') && !name(keyword: '过期')")).toMatchObject({ kind: 'raw' })
  })

  it('生成 dae 原生 name/subtag 表达式并去重', () => {
    expect(serializeGroupFilter({
      ...createGroupFilter('nodes'),
      values: ['node1', '香港 01', 'node1'],
    })).toBe("name(node1, '香港 01')")
    expect(serializeGroupFilter({
      ...createGroupFilter('subscriptions'),
      values: ['sub_a', 'sub-b'],
      exclude: true,
    })).toBe('!subtag(sub_a, sub-b)')
    expect(serializeGroupFilter({
      ...createGroupFilter('subscriptionNodes'),
      source: 'main',
      values: ['香港 01', '香港 01', 'sg_02'],
    })).toBe("subtag(main) && name('香港 01', sg_02)")
  })

  it('空选择和无法由 dae 字符串表示的名称会被拒绝', () => {
    expect(serializeGroupFilter(createGroupFilter('nodes'))).toBeNull()
    expect(serializeGroupFilter({
      ...createGroupFilter('nodes'),
      values: [`a'b"c`],
    })).toBeNull()
    expect(serializeGroupFilter({
      ...createGroupFilter('subscriptionNodes'), values: ['香港 01'],
    })).toBeNull()
  })

  it('卡片摘要对资源过滤使用可读标签', () => {
    expect(describeGroupFilter('name(node1, node2)')).toBe('节点：node1、node2')
    expect(describeGroupFilter('subtag(sub_a)')).toBe('订阅：sub_a')
    expect(describeGroupFilter("subtag(main) && name('香港 01')")).toBe('订阅 main：香港 01')
  })

  it('把导入节点并入已有节点过滤，同时保留其他分组与条件', () => {
    const text = [
      'group {',
      '  proxy {',
      '    policy: min_moving_avg',
      '    filter: name(old_node)',
      '    filter: subtag(my_sub)',
      '  }',
      '  direct_group {',
      '    policy: random',
      '  }',
      '}',
      '',
    ].join('\n')
    const next = includeNodesInGroups(text, ['proxy', 'direct_group'], ['new_node', 'old_node'])
    expect(next).toContain('filter: name(old_node, new_node)')
    expect(next).toContain('filter: subtag(my_sub)')
    expect(next.match(/direct_group \{[\s\S]*?\n  \}/)?.[0]).not.toContain('filter:')
  })

  it('复杂过滤保持原样，并追加独立的显式节点过滤', () => {
    const text = "group {\n  proxy {\n    filter: name(regex: '^HK') && !name(keyword: '过期')\n  }\n}\n"
    const next = includeNodesInGroups(text, ['proxy'], ['hk_01'])
    expect(next).toContain("filter: name(regex: '^HK') && !name(keyword: '过期')")
    expect(next).toContain('filter: name(hk_01)')
  })

  it('跨行过滤分组保持逐字节不变，避免把新声明插进续行中间', () => {
    const text = "group {\n  proxy {\n    filter: subtag(a) &&\n      !name(keyword: '过期')\n  }\n}\n"
    expect(includeNodesInGroups(text, ['proxy'], ['hk_01'])).toBe(text)
  })

  it('只在 fixed 候选数量可精确推导时返回数量', () => {
    expect(knownFixedCandidateCount([], 2, false)).toBe(2)
    expect(knownFixedCandidateCount([], 2, true)).toBeNull()
    expect(knownFixedCandidateCount([{
      ...createGroupFilter('nodes'), values: ['a', 'b', 'a'],
    }], 9, false)).toBe(2)
    expect(knownFixedCandidateCount([{
      ...createGroupFilter('nodes'), values: ['a'],
    }, {
      ...createGroupFilter('subscriptionNodes'), source: 'main', values: ['a', 'b'],
    }], 9, true)).toBe(3)
    expect(knownFixedCandidateCount([createGroupFilter('subscriptions')], 2, false)).toBeNull()
    expect(knownFixedCandidateCount([{
      ...createGroupFilter('nodes'), values: ['a'], exclude: true,
    }], 2, false)).toBeNull()
  })
})
