import type { SubscriptionNodeSource } from '../types/api'
import { readSection, type Group } from './daeconf'
import { parseGroupFilter, type GroupFilterDraft } from './group'
import { parseNodeLink } from './nodelink'

/**
 * fixed(n) 的 n 是 dae 内部 dialer 列表的下标。dae 构建那份列表时按 tag 遍历一个 map
 * （本地 node 节是一个 tag，每份订阅各一个），跨 tag 的块顺序每次启动和每次 reload
 * 都可能不同，同一 tag 内部才保持声明顺序。因此只有当一个分组的候选节点全部落在同一个
 * tag 里、且能与面板已知的列表逐一对齐时，下标才能安全地换算成节点名。
 *
 * 已知残留风险：dae 自己的链接解析器可能拒绝面板认为合法的链接，被拒的节点不占下标，
 * 此时 dae 的列表比这里算出的短，下标会错位而面板无从察觉。不做静默兜底。
 */
export interface FixedCandidate {
  /** dae 里的节点名；数组下标即 fixed(n) 的 n */
  name: string
  /** 协议，取不到时为空串 */
  protocol: string
  /** 主机名或 IP，取不到时为空串 */
  host: string
  /** 端口，取不到时为 null */
  port: number | null
}

export type FixedCandidates =
  | { resolvable: true; nodes: FixedCandidate[] }
  | { resolvable: false; reason: string }

/**
 * 拆分后端用 Go 的 net.JoinHostPort 拼出的 "主机:端口"（IPv6 形如 "[::1]:443"）。
 * 端口缺失、非数字或超出 1-65535 范围时一律退化为 port: null；不是端口分隔符的冒号
 * 不拆分，整串原样保留在 host 里——这只用于展示与延迟探测，宁可显示原文也不猜错。
 */
export function splitHostPort(hostport: string): { host: string; port: number | null } {
  const bracketed = /^\[([^\]]*)\](?::(\d+))?$/.exec(hostport)
  if (bracketed) {
    const port = bracketed[2] ? Number(bracketed[2]) : null
    return { host: bracketed[1], port: port !== null && port >= 1 && port <= 65535 ? port : null }
  }
  const colon = hostport.lastIndexOf(':')
  if (colon < 0 || !/^\d+$/.test(hostport.slice(colon + 1))) return { host: hostport, port: null }
  const port = Number(hostport.slice(colon + 1))
  return { host: hostport.slice(0, colon), port: port >= 1 && port <= 65535 ? port : null }
}

const FIXED_PATTERN = /^fixed\((\d+)\)$/

export function parseFixedIndex(policy: string | undefined): number | null {
  const matched = FIXED_PATTERN.exec((policy || '').trim())
  return matched ? Number(matched[1]) : null
}

function unresolvable(reason: string): FixedCandidates {
  return { resolvable: false, reason }
}

/** 分组的候选是否全部落在同一份订阅里；是则返回该订阅 tag。 */
function singleSubscriptionTag(filters: GroupFilterDraft[]): string | null {
  if (filters.length === 0 || filters.some((filter) => filter.exclude)) return null
  if (filters.every((filter) => filter.kind === 'subscriptionNodes')) {
    const tags = new Set(filters.map((filter) => filter.source))
    const tag = [...tags][0]
    return tags.size === 1 && tag ? tag : null
  }
  if (filters.length === 1 && filters[0].kind === 'subscriptions' && filters[0].values.length === 1) {
    return filters[0].values[0]
  }
  return null
}

function fromSubscription(
  tag: string,
  filters: GroupFilterDraft[],
  sources: SubscriptionNodeSource[],
  sourcesLoaded: boolean,
): FixedCandidates {
  if (!sourcesLoaded) return unresolvable('订阅节点缓存尚未读取，暂时只能按索引切换')
  const source = sources.find((candidate) => candidate.tag === tag)
  if (!source) return unresolvable(`订阅 ${tag} 没有离线缓存，无法确定节点顺序`)
  if (source.problem) return unresolvable(`订阅 ${tag} 的缓存不可用：${source.problem}`)
  if (source.skipped) {
    return unresolvable(`订阅 ${tag} 的缓存里有 ${source.skipped} 个无名节点，dae 仍会为它们建立连接，索引对不上`)
  }
  if (source.nodes.some((node) => node.matches > 1)) {
    return unresolvable(`订阅 ${tag} 里有同名节点，无法确定下标对应哪一个`)
  }

  const wanted = filters.every((filter) => filter.kind === 'subscriptionNodes')
    ? new Set(filters.flatMap((filter) => filter.values.map((value) => value.trim()).filter(Boolean)))
    : null
  if (wanted) {
    const known = new Set(source.nodes.map((node) => node.name))
    const missing = [...wanted].filter((name) => !known.has(name))
    if (missing.length > 0) return unresolvable(`${missing.join('、')} 不在订阅 ${tag} 的缓存中`)
  }
  const nodes: FixedCandidate[] = source.nodes
    .filter((node) => !wanted || wanted.has(node.name))
    .map((node) => {
      const { host, port } = splitHostPort(node.host || '')
      return { name: node.name, protocol: node.protocol || '', host, port }
    })
  if (nodes.length === 0) return unresolvable('该分组当前没有可选节点')
  return { resolvable: true, nodes }
}

function fromLocalNodes(
  content: string,
  filters: GroupFilterDraft[],
  sources: SubscriptionNodeSource[],
  sourcesLoaded: boolean,
): FixedCandidates {
  // 订阅是否存在只看条目本身，不能先按 tag 过滤——没写标签的订阅一样会被 dae 拉取节点、
  // 一样占用某个（面板不知道的）tag，遗漏它会让下面的判断误以为“没有订阅”。
  const subscriptionEntries = readSection(content, 'subscription').entries
  const anonymousSubscription = subscriptionEntries.some((entry) => !entry.tag?.trim())
  const subscriptionTags = subscriptionEntries
    .map((entry) => entry.tag?.trim() || '')
    .filter(Boolean)
  if (filters.length === 0 && subscriptionEntries.length > 0) {
    return unresolvable('该分组包含全部节点，其中含订阅节点；dae 跨来源的顺序每次重载都可能不同')
  }

  const entries = readSection(content, 'node').entries
  // 未命名的本地节点条目 dae 仍会为它建立 dialer、占用一个下标，面板却无法为它取名，
  // 一旦被静默剔除，它之后的所有下标都会整体错位而不自知，所以整组直接判不可解。
  if (entries.some((entry) => !entry.tag?.trim())) {
    return unresolvable('本地节点里有未命名的条目，dae 仍会为它建立连接，索引会整体偏移')
  }
  const wanted = filters.length === 0
    ? null
    : new Set(filters.flatMap((filter) => filter.values.map((value) => value.trim()).filter(Boolean)))
  const available = new Set(entries.map((entry) => entry.tag!.trim()))
  if (wanted) {
    const missing = [...wanted].filter((name) => !available.has(name))
    if (missing.length > 0) {
      return unresolvable(`${missing.join('、')} 不是显式的本地节点标签，无法确定它在 dae 里的位置`)
    }
  }

  const selected = entries.filter((entry) => !wanted || wanted.has(entry.tag!.trim()))
  if (selected.some((entry) => parseNodeLink(entry.value) === null)) {
    return unresolvable('候选里有无法解析的节点链接，dae 可能会跳过它，索引会整体前移')
  }

  // dae 的 name() 不区分来源，本地标签与订阅节点重名时会同时命中，顺序随之不可知。
  if (subscriptionEntries.length > 0) {
    // 没有标签的订阅无法用 tag 去查缓存，因而永远没法确认它是否与本地节点重名。
    if (anonymousSubscription) return unresolvable('存在未命名的订阅，无法确认它的节点是否与本地节点重名')
    if (!sourcesLoaded) return unresolvable('订阅节点缓存尚未读取，无法确认节点名是否与订阅重名')
    const usable = new Map(sources.map((source) => [source.tag, source]))
    const unusable = subscriptionTags.filter((tag) => {
      const source = usable.get(tag)
      return !source || !!source.problem
    })
    if (unusable.length > 0) {
      return unresolvable(`订阅 ${unusable.join('、')} 没有可用缓存，无法确认节点名是否与订阅重名`)
    }
    const subscriptionNames = new Set(sources.flatMap((source) => source.nodes.map((node) => node.name)))
    const collided = selected.map((entry) => entry.tag!.trim()).filter((tag) => subscriptionNames.has(tag))
    if (collided.length > 0) {
      return unresolvable(`${collided.join('、')} 与订阅里的节点同名，dae 的 name() 不区分来源`)
    }
  }

  const nodes: FixedCandidate[] = selected.map((entry) => {
    // 上面已经拒绝过 parseNodeLink 为 null 的候选，这里非空断言是安全的。
    const info = parseNodeLink(entry.value)!
    return { name: entry.tag!.trim(), protocol: info.protocol || '', host: info.host || '', port: info.port ?? null }
  })
  if (nodes.length === 0) return unresolvable('该分组当前没有可选节点')
  return { resolvable: true, nodes }
}

/** 解析某个 fixed 分组的候选节点；只有能确定 dae 真实顺序时才返回名称列表。 */
export function resolveFixedCandidates(
  content: string,
  group: Group,
  sources: SubscriptionNodeSource[],
  sourcesLoaded: boolean,
): FixedCandidates {
  if (group.policy && !group.policy.editable) {
    return unresolvable('该分组的策略声明跨行或重复，请使用原文编辑')
  }
  if (group.filters.some((filter) => !filter.editable)) {
    return unresolvable('该分组的过滤条件跨行或结构复杂，请使用原文编辑')
  }

  const filters = group.filters.map((filter) => parseGroupFilter(filter.value))
  const tag = singleSubscriptionTag(filters)
  if (tag !== null) return fromSubscription(tag, filters, sources, sourcesLoaded)
  if (filters.length === 0 || filters.every((filter) => filter.kind === 'nodes' && !filter.exclude)) {
    return fromLocalNodes(content, filters, sources, sourcesLoaded)
  }
  return unresolvable('该分组的过滤条件无法静态展开为固定顺序，只能按索引切换')
}
