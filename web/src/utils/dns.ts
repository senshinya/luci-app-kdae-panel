import type { DaeOutline, OutlineElement } from '../types/api'
import {
  findSection,
  isQuotable,
  isValidTag,
  maskComments,
  parseRoutingRulesInSection,
  parseSection,
  quote,
  readSectionBody,
  scanSections,
  setSectionBody,
  type Entry,
  type Section,
} from './daeconf'

export const DNS_FIELD_LABELS = new Map<string, string>([
  ['ipversion_prefer', 'IP 版本偏好'],
  ['fixed_domain_ttl', '固定域名 TTL'],
  ['upstream', 'DNS 上游'],
  ['routing', 'DNS 路由'],
  ['bind', '监听地址'],
  ['optimistic_cache', '乐观缓存'],
  ['optimistic_cache_ttl', '乐观缓存过期窗口'],
  ['max_cache_size', '最大缓存条目'],
])

const DNS_FIELDS = [...DNS_FIELD_LABELS.keys()]
const REQUEST_BUILTINS = new Set(['asis', 'reject'])
const RESPONSE_BUILTINS = new Set(['accept', 'reject'])
const DNS_DOMAIN_KEY = /^[A-Za-z0-9_][A-Za-z0-9_.-]*$/

let rowSequence = 0
function rowID(prefix: string): string {
  rowSequence += 1
  return `${prefix}-${rowSequence}`
}

export interface DNSUpstream {
  id: string
  name: string
  url: string
}

export interface DNSFixedTTL {
  id: string
  domain: string
  ttl: number | null
}

export interface DNSRule {
  id: string
  matcher: string
  target: string
  fallback: boolean
}

export interface DNSDraft {
  ipVersionPrefer: '' | '4' | '6'
  bind: string
  optimisticCache: boolean | null
  optimisticCacheTTL: number | null
  maxCacheSize: number | null
  fixedTTLs: DNSFixedTTL[]
  upstreams: DNSUpstream[]
  requestRules: DNSRule[]
  responseRules: DNSRule[]
}

export interface DNSState {
  present: boolean
  body: string
  draft: DNSDraft
  configured: Set<string>
  issues: string[]
  simpleSafe: boolean
}

export interface DNSCapabilities {
  version: string
  supported: Set<string>
  defaults: Map<string, string>
}

export function emptyDNSDraft(): DNSDraft {
  return {
    ipVersionPrefer: '',
    bind: '',
    optimisticCache: null,
    optimisticCacheTTL: null,
    maxCacheSize: null,
    fixedTTLs: [],
    upstreams: [],
    requestRules: [],
    responseRules: [],
  }
}

/** daed 的默认 DNS 方案；只在原配置没有 dns 内容时作为简单模式初始草稿。 */
export function defaultDNSDraft(): DNSDraft {
  const draft = emptyDNSDraft()
  draft.upstreams = [
    { id: rowID('upstream'), name: 'alidns', url: 'udp://223.5.5.5:53' },
    { id: rowID('upstream'), name: 'googledns', url: 'tcp+udp://8.8.8.8:53' },
  ]
  draft.requestRules = [
    { id: rowID('request'), matcher: 'qname(geosite:cn)', target: 'alidns', fallback: false },
    { id: rowID('request'), matcher: '', target: 'googledns', fallback: true },
  ]
  draft.responseRules = [
    { id: rowID('response'), matcher: 'upstream(googledns)', target: 'accept', fallback: false },
    { id: rowID('response'), matcher: '', target: 'accept', fallback: true },
  ]
  return draft
}

export function cloneDNSDraft(draft: DNSDraft): DNSDraft {
  return {
    ...draft,
    fixedTTLs: draft.fixedTTLs.map((item) => ({ ...item })),
    upstreams: draft.upstreams.map((item) => ({ ...item })),
    requestRules: draft.requestRules.map((item) => ({ ...item })),
    responseRules: draft.responseRules.map((item) => ({ ...item })),
  }
}

export function newDNSUpstream(): DNSUpstream {
  return { id: rowID('upstream'), name: '', url: '' }
}

export function newDNSFixedTTL(): DNSFixedTTL {
  return { id: rowID('ttl'), domain: '', ttl: null }
}

export function newDNSRule(kind: 'request' | 'response'): DNSRule {
  return {
    id: rowID(kind),
    matcher: kind === 'request' ? 'qname(geosite:cn)' : 'upstream()',
    target: kind === 'request' ? 'asis' : 'accept',
    fallback: false,
  }
}

function findDNSOutline(nodes: OutlineElement[]): OutlineElement | null {
  for (const node of nodes) {
    if (node.mapping?.toLowerCase() === 'dns') return node
    const nested = findDNSOutline(node.structure || [])
    if (nested) return nested
  }
  return null
}

/** 从当前 dae 二进制导出的 outline 读取 DNS 字段与默认值。 */
export function readDNSCapabilities(outline: DaeOutline): DNSCapabilities | null {
  const dns = findDNSOutline(outline.structure)
  if (!dns?.structure?.length) return null
  const fields = dns.structure.filter((field) => field.mapping)
  if (fields.length === 0) return null
  return {
    version: outline.version,
    supported: new Set(fields.map((field) => field.mapping!)),
    defaults: new Map(fields
      .filter((field) => field.defaultValue !== undefined && field.defaultValue !== '')
      .map((field) => [field.mapping!, field.defaultValue!])),
  }
}

function blankSections(text: string, sections: Section[]): string {
  const output = text.split('')
  for (const section of sections) {
    for (let index = section.nameStart; index <= section.bodyEnd && index < output.length; index += 1) {
      if (output[index] !== '\n') output[index] = ' '
    }
  }
  return output.join('')
}

function directEntries(body: string, sections: Section[]): { entries: Entry[]; unparsedLines: number } {
  const wrapper = `dns {\n${blankSections(body, sections)}\n}`
  const section = findSection(wrapper, 'dns')
  return section ? parseSection(wrapper, section) : { entries: [], unparsedLines: 0 }
}

function occurrence(sections: Section[], name: string): Section | null {
  return sections.find((section) => section.name === name) || null
}

function childEntries(body: string, section: Section | null, label: string, issues: string[]): Entry[] {
  if (!section) return []
  if (scanSections(body.slice(section.bodyStart, section.bodyEnd)).length > 0) {
    issues.push(`${label} 中包含嵌套块`)
  }
  const parsed = parseSection(body, section)
  if (parsed.unparsedLines > 0) issues.push(`${label} 中有 ${parsed.unparsedLines} 行无法结构化解析`)
  if (parsed.entries.some((entry) => !entry.editable)) issues.push(`${label} 中包含跨行条目`)
  return parsed.entries
}

function integer(value: string, min: number, label: string, issues: string[]): number | null {
  const parsed = Number(value)
  if (!Number.isSafeInteger(parsed) || parsed < min) {
    issues.push(`${label}不是有效整数`)
    return null
  }
  return parsed
}

function uniqueEntries(entries: Entry[], label: string, issues: string[]): Map<string, Entry> {
  const result = new Map<string, Entry>()
  for (const entry of entries) {
    if (entry.tag === null) {
      issues.push(`${label} 中存在没有名称的条目`)
      continue
    }
    if (result.has(entry.tag)) issues.push(`${label} ${entry.tag} 重复声明`)
    else result.set(entry.tag, entry)
  }
  return result
}

function ruleLineCount(text: string, section: Section): number {
  return maskComments(text.slice(section.bodyStart, section.bodyEnd))
    .split(/\r?\n/)
    .filter((line) => line.trim() !== '')
    .length
}

function readRules(
  text: string,
  section: Section | null,
  kind: 'request' | 'response',
  issues: string[],
): DNSRule[] {
  if (!section) return []
  if (scanSections(text.slice(section.bodyStart, section.bodyEnd)).length > 0) {
    issues.push(`DNS ${kind} 规则中包含未知嵌套块`)
  }
  const parsed = parseRoutingRulesInSection(text, section)
  if (ruleLineCount(text, section) !== parsed.length
    || parsed.some((rule) => !rule.editable || (!rule.isFallback && !safeMatcher(rule.match)))) {
    issues.push(`DNS ${kind} 规则包含跨行或无法识别的写法`)
  }
  if (parsed.filter((rule) => rule.isFallback).length > 1) issues.push(`DNS ${kind} 存在多个 fallback`)
  return parsed.map((rule) => ({
    id: rowID(kind),
    matcher: rule.isFallback ? '' : rule.match,
    target: rule.outbound,
    fallback: rule.isFallback,
  }))
}

function directResidual(text: string, sections: Section[]): string {
  return maskComments(blankSections(text, sections)).trim()
}

function parseScalar(entry: Entry, draft: DNSDraft, issues: string[]) {
  if (!entry.editable || entry.tag === null) {
    issues.push('DNS 基础设置包含跨行或匿名条目')
    return
  }
  switch (entry.tag) {
    case 'ipversion_prefer':
      if (entry.value === '4' || entry.value === '6') draft.ipVersionPrefer = entry.value
      else issues.push('IP 版本偏好只能是 4 或 6')
      break
    case 'bind':
      draft.bind = entry.value
      break
    case 'optimistic_cache':
      if (entry.value === 'true' || entry.value === 'false') draft.optimisticCache = entry.value === 'true'
      else issues.push('乐观缓存只能是 true 或 false')
      break
    case 'optimistic_cache_ttl':
      draft.optimisticCacheTTL = integer(entry.value, 0, '乐观缓存过期窗口', issues)
      break
    case 'max_cache_size':
      draft.maxCacheSize = integer(entry.value, 0, '最大缓存条目', issues)
      break
    default:
      issues.push(`未知 DNS 字段 ${entry.tag}`)
  }
}

function repeatedSections(sections: Section[], issues: string[]) {
  const count = new Map<string, number>()
  for (const section of sections) count.set(section.name, (count.get(section.name) || 0) + 1)
  for (const [name, total] of count) {
    if (total > 1) issues.push(`${name} 块重复出现 ${total} 次`)
  }
}

/** 读取完整配置中的 dns 节；复杂文本仍会提取摘要，但 simpleSafe 会阻止默认进入简单模式。 */
export function readDNSState(text: string): DNSState {
  const dnsSections = scanSections(text).filter((section) => section.name === 'dns')
  const present = dnsSections.length > 0
  const body = present ? readSectionBody(text, 'dns') : ''
  const draft = emptyDNSDraft()
  const configured = new Set<string>()
  const issues: string[] = []
  if (dnsSections.length > 1) issues.push(`dns 节重复出现 ${dnsSections.length} 次`)
  if (!present || body.trim() === '') return { present, body, draft, configured, issues, simpleSafe: issues.length === 0 }

  if (maskComments(body) !== body) issues.push('DNS 配置包含注释')
  const sections = scanSections(body)
  repeatedSections(sections, issues)
  for (const section of sections) {
    configured.add(section.name)
    if (!['fixed_domain_ttl', 'upstream', 'routing'].includes(section.name)) {
      issues.push(`未知 DNS 配置块 ${section.name}`)
    }
  }

  const scalar = directEntries(body, sections)
  if (scalar.unparsedLines > 0) issues.push(`DNS 基础设置中有 ${scalar.unparsedLines} 行无法结构化解析`)
  const scalarEntries = uniqueEntries(scalar.entries, 'DNS 字段', issues)
  for (const entry of scalarEntries.values()) {
    if (entry.tag) configured.add(entry.tag)
    parseScalar(entry, draft, issues)
  }

  const fixedEntries = childEntries(body, occurrence(sections, 'fixed_domain_ttl'), '固定域名 TTL', issues)
  const fixed = uniqueEntries(fixedEntries, '固定域名 TTL', issues)
  draft.fixedTTLs = [...fixed].map(([domain, entry]) => ({
    id: rowID('ttl'),
    domain,
    ttl: integer(entry.value, 0, `${domain} 的 TTL`, issues),
  }))

  const upstreamEntries = childEntries(body, occurrence(sections, 'upstream'), 'DNS 上游', issues)
  const upstreams = uniqueEntries(upstreamEntries, 'DNS 上游', issues)
  draft.upstreams = [...upstreams].map(([name, entry]) => ({ id: rowID('upstream'), name, url: entry.value }))

  const routing = occurrence(sections, 'routing')
  if (routing) {
    const routingBody = body.slice(routing.bodyStart, routing.bodyEnd)
    const routingSections = scanSections(routingBody)
    repeatedSections(routingSections, issues)
    for (const section of routingSections) {
      if (section.name !== 'request' && section.name !== 'response') {
        issues.push(`未知 DNS 路由块 ${section.name}`)
      }
    }
    if (directResidual(routingBody, routingSections) !== '') issues.push('DNS routing 中包含块外内容')
    draft.requestRules = readRules(routingBody, occurrence(routingSections, 'request'), 'request', issues)
    draft.responseRules = readRules(routingBody, occurrence(routingSections, 'response'), 'response', issues)
  }

  const upstreamNames = new Set(draft.upstreams.map((item) => item.name))
  for (const rule of draft.requestRules) {
    if (!upstreamNames.has(rule.target) && !REQUEST_BUILTINS.has(rule.target)) {
      issues.push(`DNS request 目标 ${rule.target} 没有对应上游`)
    }
  }
  for (const rule of draft.responseRules) {
    if (!upstreamNames.has(rule.target) && !RESPONSE_BUILTINS.has(rule.target)) {
      issues.push(`DNS response 目标 ${rule.target} 没有对应上游`)
    }
  }

  return {
    present,
    body,
    draft,
    configured,
    issues: [...new Set(issues)],
    simpleSafe: issues.length === 0,
  }
}

function safeText(value: string, label: string): string {
  const trimmed = value.trim()
  if (trimmed === '' || /[\r\n]/.test(trimmed)) throw new Error(`${label}不能为空或跨行`)
  return trimmed
}

function safeMatcher(value: string): boolean {
  const matcher = value.trim()
  if (matcher === '' || /[{}]|->/.test(matcher) || /^fallback\s*:/i.test(matcher)) return false
  if (maskComments(matcher) !== matcher) return false
  const probe = `routing {\n  ${matcher} -> target\n}`
  const section = findSection(probe, 'routing')
  const rules = section ? parseRoutingRulesInSection(probe, section) : []
  return rules.length === 1 && rules[0].editable && rules[0].match === matcher && rules[0].outbound === 'target'
}

function validateRules(rules: DNSRule[], upstreams: Set<string>, kind: 'request' | 'response') {
  if (rules.filter((rule) => rule.fallback).length > 1) throw new Error(`${kind} 规则只能有一个 fallback`)
  const builtins = kind === 'request' ? REQUEST_BUILTINS : RESPONSE_BUILTINS
  for (const rule of rules) {
    if (!rule.fallback) {
      const matcher = safeText(rule.matcher, `${kind} 匹配条件`)
      if (!safeMatcher(matcher)) throw new Error(`${kind} 匹配条件包含不完整或不安全的配置语法`)
    }
    const target = safeText(rule.target, `${kind} 目标`)
    if (/\s/.test(target) || (!upstreams.has(target) && !builtins.has(target))) {
      throw new Error(`${kind} 目标 ${target} 不是内置动作或现有上游`)
    }
  }
}

function ruleBlock(name: string, rules: DNSRule[], indent: string): string[] {
  if (rules.length === 0) return []
  return [
    `${indent}${name} {`,
    ...rules.map((rule) => rule.fallback
      ? `${indent}  fallback: ${rule.target.trim()}`
      : `${indent}  ${rule.matcher.trim()} -> ${rule.target.trim()}`),
    `${indent}}`,
  ]
}

/** 用简单模式草稿重建 dns 节正文；调用方必须在用户明确点击“应用”后才使用。 */
export function buildDNSBody(draft: DNSDraft): string {
  const blocks: string[] = []
  if (draft.ipVersionPrefer !== '') {
    if (draft.ipVersionPrefer !== '4' && draft.ipVersionPrefer !== '6') throw new Error('IP 版本偏好只能是 4 或 6')
    blocks.push(`  ipversion_prefer: ${draft.ipVersionPrefer}`)
  }

  if (draft.fixedTTLs.length > 0) {
    const domains = new Set<string>()
    const lines = ['  fixed_domain_ttl {']
    for (const item of draft.fixedTTLs) {
      const domain = safeText(item.domain, '固定 TTL 域名')
      // declaration 的键必须是裸 ID，不能套引号；这里接受域名需要的安全字符子集。
      if (!DNS_DOMAIN_KEY.test(domain)) throw new Error(`固定 TTL 域名 ${domain} 不能作为 dae 声明键`)
      if (domains.has(domain)) throw new Error(`固定 TTL 域名 ${domain} 重复`)
      if (item.ttl === null || !Number.isSafeInteger(item.ttl) || item.ttl < 0) throw new Error(`${domain} 的 TTL 必须是非负整数`)
      domains.add(domain)
      lines.push(`    ${domain}: ${item.ttl}`)
    }
    lines.push('  }')
    blocks.push(lines.join('\n'))
  }

  const cacheLines: string[] = []
  if (draft.optimisticCache !== null) cacheLines.push(`  optimistic_cache: ${draft.optimisticCache}`)
  if (draft.optimisticCacheTTL !== null) {
    if (!Number.isSafeInteger(draft.optimisticCacheTTL) || draft.optimisticCacheTTL < 0) throw new Error('乐观缓存过期窗口必须是非负整数')
    cacheLines.push(`  optimistic_cache_ttl: ${draft.optimisticCacheTTL}`)
  }
  if (draft.maxCacheSize !== null) {
    if (!Number.isSafeInteger(draft.maxCacheSize) || draft.maxCacheSize < 0) throw new Error('最大缓存条目必须是非负整数')
    cacheLines.push(`  max_cache_size: ${draft.maxCacheSize}`)
  }
  if (draft.bind.trim() !== '') {
    const bind = safeText(draft.bind, 'DNS 监听地址')
    if (!isQuotable(bind)) throw new Error('DNS 监听地址无法在 dae 配置中无损表示')
    cacheLines.push(`  bind: ${quote(bind)}`)
  }
  if (cacheLines.length > 0) blocks.push(cacheLines.join('\n'))

  const upstreamNames = new Set<string>()
  if (draft.upstreams.length > 0) {
    const lines = ['  upstream {']
    for (const item of draft.upstreams) {
      const name = safeText(item.name, 'DNS 上游名称')
      const url = safeText(item.url, `${name} 地址`)
      if (!isValidTag(name)) throw new Error(`DNS 上游名称 ${name} 不是安全的裸标识符`)
      if (upstreamNames.has(name)) throw new Error(`DNS 上游 ${name} 重复`)
      if (!isQuotable(url)) throw new Error(`${name} 地址无法在 dae 配置中无损表示`)
      upstreamNames.add(name)
      lines.push(`    ${name}: ${quote(url)}`)
    }
    lines.push('  }')
    blocks.push(lines.join('\n'))
  }

  validateRules(draft.requestRules, upstreamNames, 'request')
  validateRules(draft.responseRules, upstreamNames, 'response')
  if (draft.requestRules.length > 0 || draft.responseRules.length > 0) {
    const lines = [
      '  routing {',
      ...ruleBlock('request', draft.requestRules, '    '),
      ...ruleBlock('response', draft.responseRules, '    '),
      '  }',
    ]
    blocks.push(lines.join('\n'))
  }

  return blocks.join('\n\n')
}

export function configuredUnsupported(state: DNSState, capabilities: DNSCapabilities | null): string[] {
  if (!capabilities) return []
  return DNS_FIELDS.filter((field) => state.configured.has(field) && !capabilities.supported.has(field))
}

/** 给旧版生成的配置补一份待保存的默认 DNS；已有 dns 节时逐字节保持原样。 */
export function withDefaultDNS(text: string): string {
  if (findSection(text, 'dns')) return text
  return setSectionBody(text, 'dns', buildDNSBody(defaultDNSDraft()))
}

/** 配置文件尚不存在时，配置管理页展示的完整初始草稿。 */
export function defaultConfiguration(): string {
  return `global {}

dns {
${buildDNSBody(defaultDNSDraft())}
}

routing {}
`
}
