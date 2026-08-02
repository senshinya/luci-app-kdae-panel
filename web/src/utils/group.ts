import { isQuotable, isValidTag, parseGroups, quote, setGroupFilter, unquote } from './daeconf'

export type GroupFilterKind = 'nodes' | 'subscriptionNodes' | 'subscriptions' | 'nameKeyword' | 'nameRegex' | 'raw'

export interface GroupFilterDraft {
  kind: GroupFilterKind
  value: string
  values: string[]
  source: string
  exclude: boolean
}

export function createGroupFilter(kind: GroupFilterKind): GroupFilterDraft {
  return { kind, value: '', values: [], source: '', exclude: false }
}

/**
 * 解析 name()/subtag() 中不带命名参数的简单值列表。
 * regex: 等高级写法故意返回 null，交给高级表达式原样处理。
 */
function parseArguments(body: string): string[] | null {
  const values: string[] = []
  let start = 0
  let quoteChar = ''

  const push = (end: number): boolean => {
    const raw = body.slice(start, end).trim()
    if (raw === '') return false
    const first = raw[0]
    if (first === "'" || first === '"') {
      if (raw.length < 2 || raw[raw.length - 1] !== first) return false
      values.push(unquote(raw))
      return true
    }
    // 裸参数只接受 dae 标识符的安全子集；空格、冒号和括号意味着更复杂的表达式。
    if (!/^[A-Za-z_][A-Za-z0-9_.-]*$/.test(raw)) return false
    values.push(raw)
    return true
  }

  for (let index = 0; index < body.length; index += 1) {
    const char = body[index]
    if (quoteChar !== '') {
      if (char === '\\' && body[index + 1] === quoteChar) index += 1
      else if (char === quoteChar) quoteChar = ''
      continue
    }
    if (char === "'" || char === '"') quoteChar = char
    else if (char === ',') {
      if (!push(index)) return null
      start = index + 1
    } else if (char === ':' || char === '(' || char === ')') {
      return null
    }
  }
  if (quoteChar !== '' || !push(body.length)) return null
  return [...new Set(values)]
}

export function parseGroupFilter(value: string): GroupFilterDraft {
  const trimmed = value.trim()
  const exclude = trimmed.startsWith('!')
  const expression = exclude ? trimmed.slice(1).trim() : trimmed

  const subscriptionNodes = /^subtag\((.*?)\)\s*&&\s*name\((.*)\)$/.exec(expression)
  if (subscriptionNodes && !exclude) {
    const sources = parseArguments(subscriptionNodes[1])
    const values = parseArguments(subscriptionNodes[2])
    if (sources?.length === 1 && isValidTag(sources[0]) && values) {
      return { kind: 'subscriptionNodes', value: '', values, source: sources[0], exclude: false }
    }
  }

  const subscription = /^subtag\((.*)\)$/.exec(expression)
  if (subscription) {
    const values = parseArguments(subscription[1])
    if (values && values.every(isValidTag)) {
      return { kind: 'subscriptions', value: '', values, source: '', exclude }
    }
  }

  const nameMatcher = /^name\((keyword|regex)\s*:\s*(['"])(.*)\2\)$/.exec(expression)
  if (nameMatcher) {
    return {
      kind: nameMatcher[1] === 'keyword' ? 'nameKeyword' : 'nameRegex',
      value: nameMatcher[3],
      values: [],
      source: '',
      exclude,
    }
  }

  const nodes = /^name\((.*)\)$/.exec(expression)
  if (nodes) {
    const values = parseArguments(nodes[1])
    if (values) return { kind: 'nodes', value: '', values, source: '', exclude }
  }

  return { kind: 'raw', value: trimmed, values: [], source: '', exclude: false }
}

function uniqueValues(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
}

export function serializeGroupFilter(filter: GroupFilterDraft): string | null {
  if (filter.kind === 'raw') return filter.value.trim() || null

  let expression: string
  if (filter.kind === 'nodes') {
    const values = uniqueValues(filter.values)
    if (values.length === 0 || values.some((value) => !isQuotable(value))) return null
    expression = `name(${values.map((value) => isValidTag(value) ? value : quote(value)).join(', ')})`
  } else if (filter.kind === 'subscriptionNodes') {
    const values = uniqueValues(filter.values)
    if (!isValidTag(filter.source) || filter.exclude || values.length === 0
      || values.some((value) => !isQuotable(value))) return null
    expression = `subtag(${filter.source}) && name(${values.map((value) => isValidTag(value) ? value : quote(value)).join(', ')})`
  } else if (filter.kind === 'subscriptions') {
    const values = uniqueValues(filter.values)
    if (values.length === 0 || values.some((value) => !isValidTag(value))) return null
    expression = `subtag(${values.join(', ')})`
  } else {
    const value = filter.value.trim()
    if (value === '' || !isQuotable(value)) return null
    expression = `name(${filter.kind === 'nameKeyword' ? 'keyword' : 'regex'}: ${quote(value)})`
  }
  return filter.exclude ? `!${expression}` : expression
}

export function describeGroupFilter(value: string): string {
  const parsed = parseGroupFilter(value)
  if (parsed.kind === 'nodes') return `${parsed.exclude ? '排除节点' : '节点'}：${parsed.values.join('、')}`
  if (parsed.kind === 'subscriptionNodes') return `订阅 ${parsed.source}：${parsed.values.join('、')}`
  if (parsed.kind === 'subscriptions') return `${parsed.exclude ? '排除订阅' : '订阅'}：${parsed.values.join('、')}`
  return value
}

/** 将新导入的显式节点标签加入指定分组；无过滤分组本来就包含全部节点。 */
export function includeNodesInGroups(text: string, groupNames: string[], nodeTags: string[]): string {
  const selected = new Set(groupNames)
  const tags = uniqueValues(nodeTags).filter(isValidTag)
  if (selected.size === 0 || tags.length === 0) return text

  let next = text
  for (const groupName of selected) {
    const group = parseGroups(next).find((candidate) => candidate.name === groupName)
    if (!group || group.filters.length === 0 || group.filters.some((filter) => !filter.editable)) continue

    const filterIndex = group.filters.findIndex((filter) => {
      const parsed = parseGroupFilter(filter.value)
      return filter.editable && parsed.kind === 'nodes' && !parsed.exclude
    })
    if (filterIndex >= 0) {
      const parsed = parseGroupFilter(group.filters[filterIndex].value)
      parsed.values = uniqueValues([...parsed.values, ...tags])
      next = setGroupFilter(next, group, filterIndex, serializeGroupFilter(parsed)!)
      continue
    }
    next = setGroupFilter(next, group, group.filters.length, `name(${tags.join(', ')})`)
  }
  return next
}

/**
 * 只在候选数量可由当前配置精确得出时返回数字；整份订阅、关键词和高级表达式均返回 null。
 * fixed(n) 的 n 是从 0 开始的索引。
 */
export function knownFixedCandidateCount(
  filters: GroupFilterDraft[],
  unfilteredNodeCount: number,
  hasSubscriptions: boolean,
): number | null {
  if (filters.length === 0) return hasSubscriptions ? null : unfilteredNodeCount
  if (filters.some((filter) => !['nodes', 'subscriptionNodes'].includes(filter.kind) || filter.exclude)) return null
  const candidates = new Set<string>()
  for (const filter of filters) {
    for (const value of filter.values.map((item) => item.trim()).filter(Boolean)) {
      candidates.add(filter.kind === 'nodes' ? `node:${value}` : `subscription:${filter.source}:${value}`)
    }
  }
  return candidates.size
}
