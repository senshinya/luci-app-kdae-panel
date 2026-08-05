// dae 配置 DSL 的节级解析与手术式改写。
// 原则:只改写被编辑的行或节,其余文本(注释、空行、未知节)逐字节保留;
// 所有写回仍经由后端 dae validate 校验,解析器本身不是安全边界。
//
// 词法与 dae-config-dist 的 dae_config.g4 对齐:
// - 注释有 `#`(行注释)与 `/* */`(块注释),但两者只在 token 边界成立。
//   词法上 `#`、`*`、`/` 都是合法的 token 内字符,所以 `foo#bar` 是一个完整
//   token,`config.d/*.dae` 不是块注释;反过来 `/* x */# y` 里的 `#` 紧跟在
//   块注释之后,处在 token 边界上,确实开启行注释。
// - 字符串单双引号等价,`\<引号>` 不闭合字符串,但 dae 剥引号时不做反转义,
//   因此同时含单双引号的值无法表示(见 isQuotable)。
// - 换行不是语法分隔符。本模块的按行编辑因此附带守卫:被多行字符串或跨行块
//   注释穿过的行不作为可编辑条目,交回原文编辑器处理。

export interface Section {
  name: string
  nameStart: number
  bodyStart: number
  bodyEnd: number
}

export interface Entry {
  tag: string | null
  value: string
  /** 条目跨行时为 false:可以展示,但不能安全地按行改写。 */
  editable: boolean
  lineStart: number
  lineEnd: number
}

export interface GroupProperty {
  key: string
  value: string
  /** 值跨行或括号/引号不闭合时为 false:可以展示,但不能安全地定点改写。 */
  editable: boolean
  lineStart: number
  lineEnd: number
  valueStart: number
  valueEnd: number
  /** 声明本身的范围，用于同一行存在多个声明时只删除当前声明。 */
  declarationStart: number
  declarationEnd: number
}

export interface Group {
  name: string
  section: Section
  policy: GroupProperty | null
  filters: GroupProperty[]
}

export interface RoutingRule {
  match: string
  outbound: string
  isFallback: boolean
  /** 只有独占一行且未被跨行字符串/注释穿过时才能定点改写。 */
  editable: boolean
  lineStart: number
  lineEnd: number
}

const INDENT = '  '

/** dae_config.g4 的 SAFE_CHAR:出现在 token 内部的字符。 */
const SAFE_CHAR = /[A-Za-z0-9_/\\^*.+=@$!#%-]/

interface Span {
  start: number
  end: number
}

interface Masked {
  /** 与原文等长,注释被替换成空格(换行保留),偏移可直接用于原文。 */
  text: string
  /** 跨行的字符串与块注释:被它们穿过的行不能按行编辑。 */
  multiLine: Span[]
}

/** 跳过引号字符串,返回闭合引号之后的位置。`\<同类引号>` 不闭合。 */
function skipString(text: string, index: number, end: number): number {
  const quoteChar = text[index]
  for (index += 1; index < end; index += 1) {
    if (text[index] === '\\' && text[index + 1] === quoteChar) index += 1
    else if (text[index] === quoteChar) return index + 1
  }
  return end
}

/**
 * 单遍扫描,按 dae 词法把注释空白化,并记录跨行的字符串与块注释。
 * `insideToken` 复刻最长匹配:token 内部的 `#` 与 `/*` 不开启注释,而字符串或
 * 注释结束后一定回到 token 边界。
 */
function maskWithSpans(text: string): Masked {
  const output = text.split('')
  const multiLine: Span[] = []
  const blank = (from: number, to: number) => {
    for (let index = from; index < to; index += 1) {
      if (output[index] !== '\n') output[index] = ' '
    }
  }
  const record = (from: number, to: number) => {
    if (text.lastIndexOf('\n', to - 1) >= from) multiLine.push({ start: from, end: to })
  }

  let insideToken = false
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]
    if (char === "'" || char === '"') {
      const end = skipString(text, index, text.length)
      record(index, end)
      index = end - 1
      insideToken = false
    } else if (!insideToken && char === '#') {
      let end = text.indexOf('\n', index)
      if (end < 0) end = text.length
      blank(index, end)
      index = end - 1
      insideToken = false
    } else if (!insideToken && char === '/' && text[index + 1] === '*') {
      const closing = text.indexOf('*/', index + 2)
      const end = closing < 0 ? text.length : closing + 2
      record(index, end)
      blank(index, end)
      index = end - 1
      insideToken = false
    } else {
      insideToken = SAFE_CHAR.test(char)
    }
  }
  return { text: output.join(''), multiLine }
}

export function maskComments(text: string): string {
  return maskWithSpans(text).text
}

/** 扫描掩码文本 [from, to) 内的一层节:`name { ... }`。 */
function scanMasked(masked: string, from: number, to: number): Section[] {
  const sections: Section[] = []
  let tokenStart = -1
  let token = { start: -1, end: -1 }
  const flushToken = (end: number, discard = false) => {
    if (tokenStart >= 0 && !discard) token = { start: tokenStart, end }
    if (discard) token = { start: -1, end: -1 }
    tokenStart = -1
  }
  for (let index = from; index < to; index += 1) {
    const char = masked[index]
    if (char === "'" || char === '"') {
      flushToken(index, true)
      index = skipString(masked, index, to) - 1
    } else if (char === '{') {
      flushToken(index)
      if (token.start < 0) continue
      const section: Section = {
        name: masked.slice(token.start, token.end),
        nameStart: token.start,
        bodyStart: index + 1,
        bodyEnd: -1,
      }
      let depth = 1
      for (index += 1; index < to && depth > 0; index += 1) {
        const inner = masked[index]
        if (inner === "'" || inner === '"') index = skipString(masked, index, to) - 1
        else if (inner === '{') depth += 1
        else if (inner === '}') depth -= 1
      }
      if (depth > 0) break
      section.bodyEnd = index - 1
      sections.push(section)
      index -= 1
      token = { start: -1, end: -1 }
    } else if (/\s/.test(char)) {
      flushToken(index)
    } else if (':,()[]}'.includes(char)) {
      // 键值行(如 policy: …)与函数参数不是节名,丢弃当前 token
      flushToken(index, true)
    } else if (tokenStart < 0) {
      tokenStart = index
    }
  }
  return sections
}

export function scanSections(text: string): Section[] {
  return scanMasked(maskWithSpans(text).text, 0, text.length)
}

export function findSection(text: string, name: string): Section | null {
  return scanSections(text).find((section) => section.name === name) || null
}

interface Line {
  start: number
  end: number
  editable: boolean
}

/** 切分节内的行,并标记哪些行被跨行字符串或块注释穿过。 */
function linesOf(masked: Masked, section: Section): Line[] {
  const lines: Line[] = []
  let start = section.bodyStart
  while (start < section.bodyEnd) {
    let end = masked.text.indexOf('\n', start)
    if (end < 0 || end > section.bodyEnd) end = section.bodyEnd
    const crossed = masked.multiLine.some((span) => span.start < end && span.end > start)
    lines.push({ start, end, editable: !crossed })
    start = end + 1
  }
  return lines
}

/** 剥掉外层引号。与 dae 的 walker 一致:不做反转义。 */
export function unquote(value: string): string {
  const trimmed = value.trim()
  const first = trimmed[0]
  if ((first === "'" || first === '"') && trimmed.length >= 2 && trimmed[trimmed.length - 1] === first) {
    return trimmed.slice(1, -1)
  }
  return trimmed
}

/**
 * dae 剥引号时不反转义,因此同时含单引号和双引号的值无法无损表示;
 * 换行也会破坏按行编辑。写回前必须先用本函数确认。
 */
export function isQuotable(value: string): boolean {
  return !(value.includes("'") && value.includes('"')) && !/[\r\n]/.test(value)
}

/** 选用值中不存在的引号类型。调用前应先经 isQuotable 判定。 */
export function quote(value: string): string {
  return value.includes("'") ? '"' + value + '"' : "'" + value + "'"
}

/**
 * 读取一行中的单个值 token(引号字符串或裸字面量)。
 * 若其后还有其他内容,说明这一行不止一个条目,返回 null(拒绝按行编辑)。
 */
function singleToken(masked: string, from: number, to: number): string | null {
  let start = from
  while (start < to && /\s/.test(masked[start])) start += 1
  if (start >= to) return null
  let end: number
  if (masked[start] === "'" || masked[start] === '"') {
    end = skipString(masked, start, to)
  } else {
    end = start
    while (end < to && !/\s/.test(masked[end])) end += 1
  }
  if (masked.slice(end, to).trim() !== '') return null
  return masked.slice(start, end)
}

/**
 * 解析 node/subscription 一类 `tag: 'value'` 或 `'value'` 列表节。
 *
 * 换行在 dae 里只是空白,所以 `tag:` 独占一行时,值其实来自下一行。
 * 这类跨行条目按原样展示但标记为不可编辑——否则把下一行当成匿名条目删掉,
 * 会让再下一个节点静默改挂到这个 tag 名下。
 */
export function parseEntries(text: string, section: Section): Entry[] {
  return parseSection(text, section).entries
}

/** 按节名读取内容；节不存在时返回空集而不是 null，调用方无需分支。 */
export function readSection(text: string, name: string): SectionContents {
  const section = findSection(text, name)
  return section ? parseSection(text, section) : { entries: [], unparsedLines: 0 }
}

export interface SectionContents {
  entries: Entry[]
  /** 无法安全归类为条目、因而未出现在上面列表中的非空行数。 */
  unparsedLines: number
}

/** 解析节内容,同时报告有多少非空行没能被安全解析,便于界面如实说明。 */
export function parseSection(text: string, section: Section): SectionContents {
  const masked = maskWithSpans(text)
  const entries: Entry[] = []
  let unparsedLines = 0
  let danglingTag: string | null = null
  for (const line of linesOf(masked, section)) {
    const content = masked.text.slice(line.start, line.end)
    if (content.trim() === '') continue
    if (!line.editable) {
      // 跨行字符串或块注释穿过的行:归到未解析计数,而不是当作不存在
      if (danglingTag === null) unparsedLines += 1
      danglingTag = null
      continue
    }
    const push = (tag: string | null, raw: string, editable: boolean) => {
      const value = unquote(raw)
      if (value === '') unparsedLines += 1
      else entries.push({ tag, value, editable, lineStart: line.start, lineEnd: line.end })
    }
    if (danglingTag !== null) {
      const rawValue = singleToken(masked.text, line.start, line.end)
      if (rawValue === null) unparsedLines += 1
      else push(danglingTag, rawValue, false)
      danglingTag = null
      continue
    }
    const colon = topLevelColon(content)
    if (colon < 0) {
      const rawValue = singleToken(masked.text, line.start, line.end)
      if (rawValue === null) unparsedLines += 1
      else push(null, rawValue, true)
      continue
    }
    const rawTag = singleToken(masked.text, line.start, line.start + colon)
    if (rawTag === null) {
      unparsedLines += 1
      continue
    }
    if (content.slice(colon + 1).trim() === '') {
      danglingTag = unquote(rawTag)
      continue
    }
    const rawValue = singleToken(masked.text, line.start + colon + 1, line.end)
    if (rawValue === null) unparsedLines += 1
    else push(unquote(rawTag), rawValue, true)
  }
  if (danglingTag !== null) unparsedLines += 1
  return { entries, unparsedLines }
}

/** 引号外的第一个冒号,即 `ID ':' 值` 的分隔符。 */
function topLevelColon(line: string): number {
  for (let index = 0; index < line.length; index += 1) {
    const char = line[index]
    if (char === "'" || char === '"') index = skipString(line, index, line.length) - 1
    else if (char === ':') return index
  }
  return -1
}

/** 括号与引号都闭合的值才可以整体替换。 */
function balanced(value: string): boolean {
  let depth = 0
  for (let index = 0; index < value.length; index += 1) {
    const char = value[index]
    if (char === "'" || char === '"') {
      const end = skipString(value, index, value.length)
      if (end >= value.length && value[value.length - 1] !== char) return false
      index = end - 1
    } else if (char === '(') depth += 1
    else if (char === ')') depth -= 1
    if (depth < 0) return false
  }
  return depth === 0
}

interface GroupPropertyStart {
  key: string
  start: number
  colon: number
}

function isIdentifierStart(char: string | undefined): boolean {
  return char !== undefined && /[A-Za-z_]/.test(char)
}

function isIdentifierPart(char: string | undefined): boolean {
  return char !== undefined && /[A-Za-z0-9_.-]/.test(char)
}

/**
 * 找到一行中处于最外层的所有 `键: 值` 声明。
 *
 * filter 的值允许包含空格、括号和引号，因此不能用一个正则把整行当成值；
 * 包括 tcp_check_url 在内的未知键也必须成为边界，否则修改相邻的 policy/filter
 * 时仍可能把未知声明一起吞掉。
 */
function groupPropertyStarts(masked: string, lineStart: number, lineEnd: number): GroupPropertyStart[] {
  const starts: GroupPropertyStart[] = []
  let depth = 0
  for (let index = lineStart; index < lineEnd;) {
    const char = masked[index]
    if (char === "'" || char === '"') {
      index = skipString(masked, index, lineEnd)
      continue
    }
    if (char === '(') {
      depth += 1
      index += 1
      continue
    }
    if (char === ')') {
      depth = Math.max(0, depth - 1)
      index += 1
      continue
    }
    if (depth === 0 && isIdentifierStart(char)) {
      const wordStart = index
      let wordEnd = index + 1
      while (wordEnd < lineEnd && isIdentifierPart(masked[wordEnd])) wordEnd += 1
      const word = masked.slice(wordStart, wordEnd)
      let colon = wordEnd
      while (colon < lineEnd && /\s/.test(masked[colon])) colon += 1
      if (masked[colon] === ':') {
        starts.push({ key: word, start: wordStart, colon })
      }
      index = wordEnd
      continue
    }
    index += 1
  }
  return starts
}

function skipWhitespace(text: string, index: number, end: number): number {
  while (index < end && /\s/.test(text[index])) index += 1
  return index
}

/**
 * 按声明范围解析一行。policy 是单 token，filter 则取到下一条同层声明之前；
 * 无法确认边界时仍返回属性供展示，但整行属性都会被标记为不可编辑。
 */
function parseGroupProperties(masked: Masked, line: Line): GroupProperty[] {
  const starts = groupPropertyStarts(masked.text, line.start, line.end)
  if (starts.length === 0) return []

  const properties: GroupProperty[] = []

  for (let index = 0; index < starts.length; index += 1) {
    const start = starts[index]
    const boundary = starts[index + 1]?.start ?? line.end
    if (start.key !== 'policy' && start.key !== 'filter') continue

    const valueStart = skipWhitespace(masked.text, start.colon + 1, boundary)
    let valueEnd = valueStart
    let valueSafe = true

    if (start.key === 'policy') {
      // dae 的 policy 值（fixed(n)、min、random 等）本身不含空格。
      while (valueEnd < boundary && !/\s/.test(masked.text[valueEnd])) valueEnd += 1
      valueSafe = valueEnd > valueStart && masked.text.slice(valueEnd, boundary).trim() === ''
    } else {
      valueEnd = boundary
      while (valueEnd > valueStart && /\s/.test(masked.text[valueEnd - 1])) valueEnd -= 1
      valueSafe = valueEnd > valueStart
    }

    const value = masked.text.slice(valueStart, valueEnd)
    const prefixSafe = index > 0 || masked.text.slice(line.start, start.start).trim() === ''
    properties.push({
      key: start.key,
      value,
      // 以 && 结尾表示下一行仍是同一条 filter，不能按当前行定点改写。
      editable: line.editable && prefixSafe && value !== ''
        && !value.endsWith('&&') && balanced(value) && valueSafe,
      lineStart: line.start,
      lineEnd: line.end,
      valueStart,
      valueEnd,
      declarationStart: start.start,
      declarationEnd: valueEnd,
    })
  }
  return properties
}

export function parseGroups(text: string): Group[] {
  const masked = maskWithSpans(text)
  const groupSection = scanMasked(masked.text, 0, text.length).find((section) => section.name === 'group')
  if (!groupSection) return []
  return scanMasked(masked.text, groupSection.bodyStart, groupSection.bodyEnd).map((section) => {
    const group: Group = { name: section.name, section, policy: null, filters: [] }
    // 换行只是空白,续行可能出现在下一行行首(如 `&& !name(...)`)。
    // 任何不自成一条声明的非空行都只能是上一条声明的续行,据此把它降级为只读。
    let previous: GroupProperty | null = null
    for (const line of linesOf(masked, section)) {
      const content = masked.text.slice(line.start, line.end)
      const properties = parseGroupProperties(masked, line)
      if (properties.length === 0) {
        if (content.trim() !== '' && !/^\s*[A-Za-z_][\w.-]*\s*:/.test(content)) {
          if (previous) previous.editable = false
          previous = null
        }
        continue
      }
      for (const property of properties) {
        if (property.key === 'policy') {
          // 出现重复 policy 时哪条生效由 dae 决定,面板不猜,一律降级为只读
          if (group.policy) group.policy.editable = false
          else group.policy = property
        } else {
          group.filters.push(property)
        }
      }
      previous = properties[properties.length - 1]
    }
    return group
  })
}

/** 找到引号外最后一个 `->`,即规则的出站分隔符。 */
function outboundArrow(value: string): number {
  let arrow = -1
  for (let index = 0; index < value.length; index += 1) {
    const char = value[index]
    if (char === "'" || char === '"') index = skipString(value, index, value.length) - 1
    else if (char === '-' && value[index + 1] === '>') {
      arrow = index
      index += 1
    }
  }
  return arrow
}

/** 解析指定节内的规则；供顶层 routing 与 dns.routing 子节复用。 */
export function parseRoutingRulesInSection(text: string, section: Section): RoutingRule[] {
  const masked = maskWithSpans(text)
  const rules: RoutingRule[] = []
  let pendingParts: string[] = []
  let pendingStart = -1
  let pendingEnd = -1
  let pendingEditable = true
  let pendingLines = 0
  const reset = () => {
    pendingParts = []
    pendingStart = -1
    pendingEnd = -1
    pendingEditable = true
    pendingLines = 0
  }
  const push = (match: string, outbound: string, isFallback: boolean) => {
    rules.push({
      match,
      outbound,
      isFallback,
      editable: pendingLines === 1 && pendingEditable,
      lineStart: pendingStart,
      lineEnd: pendingEnd,
    })
    reset()
  }
  for (const line of linesOf(masked, section)) {
    const content = masked.text.slice(line.start, line.end).trim()
    if (content === '') continue
    if (pendingStart < 0) pendingStart = line.start
    pendingEnd = line.end
    pendingEditable = pendingEditable && line.editable
    pendingLines += 1
    pendingParts.push(content)
    const pending = pendingParts.join(' ')
    const fallback = /^fallback\s*:\s*([^\s]+)\s*$/.exec(pending)
    if (fallback && balanced(pending)) {
      push('fallback', fallback[1], true)
      continue
    }
    const arrow = outboundArrow(pending)
    if (arrow >= 0 && balanced(pending)) {
      const match = pending.slice(0, arrow).trim()
      const outbound = pending.slice(arrow + 2).trim()
      if (match !== '' && outbound !== '' && !/\s/.test(outbound)) push(match, outbound, false)
    }
  }
  return rules
}

export function parseRoutingRules(text: string): RoutingRule[] {
  const masked = maskWithSpans(text)
  const section = scanMasked(masked.text, 0, text.length).find((candidate) => candidate.name === 'routing')
  return section ? parseRoutingRulesInSection(text, section) : []
}

function splice(text: string, start: number, end: number, replacement: string): string {
  return text.slice(0, start) + replacement + text.slice(end)
}

/** 文件既有的换行风格,插入新行时沿用它。 */
function lineEnding(text: string): string {
  return text.includes('\r\n') && !/[^\r]\n/.test(text) ? '\r\n' : '\n'
}

/** 删除一整行(含行尾换行)。 */
export function removeLine(text: string, lineStart: number, lineEnd: number): string {
  const end = text[lineEnd] === '\n' ? lineEnd + 1 : lineEnd
  return splice(text, lineStart, end, '')
}

/**
 * 用新内容整行替换,保持缩进、该行原有的行尾,以及行尾注释。
 * 注释属于"未被编辑的内容",按模块约定必须原样保留。
 */
export function replaceLine(text: string, lineStart: number, lineEnd: number, line: string, indent = INDENT): string {
  const carriageReturn = text[lineEnd - 1] === '\r' ? '\r' : ''
  const existing = text.slice(lineStart, lineEnd).replace(/\r$/, '')
  const existingIndent = /^[ \t]*/.exec(existing)?.[0]
  const masked = maskWithSpans(text).text
  let comment = ''
  for (let index = lineStart; index < lineEnd; index += 1) {
    // 掩码把注释替换成空格,首个"原文非空白但掩码为空格"的位置就是注释起点
    if (masked[index] === ' ' && text[index] !== ' ' && text[index] !== '\t') {
      comment = ' ' + text.slice(index, lineEnd).replace(/\r$/, '').trimEnd()
      break
    }
  }
  return splice(text, lineStart, lineEnd, (existingIndent ?? indent) + line + comment + carriageReturn)
}

/** 声明的键在 dae 词法中必须是裸 ID;这里进一步收窄到无歧义的安全子集。 */
export function isValidTag(tag: string): boolean {
  return /^[A-Za-z_][A-Za-z0-9_.-]*$/.test(tag)
}

/** 向节内追加若干行;节不存在时在文末创建。 */
export function appendToSection(text: string, sectionName: string, lines: string[], indent = INDENT): string {
  const section = findSection(text, sectionName)
  const newline = lineEnding(text)
  const block = lines.map((line) => indent + line).join(newline)
  if (!section) {
    const trailing = /\n\s*\n\s*$/.test(text) ? '' : text.endsWith('\n') ? newline : newline + newline
    return text + (text === '' ? '' : trailing) + sectionName + ' {' + newline + block + newline + '}' + newline
  }
  const before = text.slice(section.bodyStart, section.bodyEnd)
  const insertion = (before === '' || before.endsWith('\n') ? '' : newline) + block + newline
  return splice(text, section.bodyEnd, section.bodyEnd, insertion)
}

/** 读取节大括号内的原文，只移除最外层各一个换行，内部缩进与注释不变。 */
export function readSectionBody(text: string, sectionName: string): string {
  const section = findSection(text, sectionName)
  if (!section) return ''
  let body = text.slice(section.bodyStart, section.bodyEnd)
  if (body.startsWith('\r\n')) body = body.slice(2)
  else if (body.startsWith('\n')) body = body.slice(1)
  if (body.endsWith('\r\n')) body = body.slice(0, -2)
  else if (body.endsWith('\n')) body = body.slice(0, -1)
  return body
}

/**
 * 替换单个节的正文；节不存在时在文末创建。
 * 这是用户明确选择“编辑本节原文”后的整节替换，其余顶层节仍逐字节保留。
 */
export function setSectionBody(text: string, sectionName: string, body: string): string {
  const newline = lineEnding(text)
  let normalized = body.replace(/\r\n|\r|\n/g, newline)
  if (normalized.startsWith(newline)) normalized = normalized.slice(newline.length)
  if (normalized.endsWith(newline)) normalized = normalized.slice(0, -newline.length)
  const replacement = newline + normalized + (normalized === '' ? '' : newline)
  const section = findSection(text, sectionName)
  if (section) return splice(text, section.bodyStart, section.bodyEnd, replacement)

  const trailing = text === '' || text.endsWith(newline) ? '' : newline
  const separation = text === '' || /(?:\r?\n)\s*$/.test(text) ? '' : newline
  return text + trailing + separation + sectionName + ' {' + replacement + '}' + newline
}

function routingLine(match: string, outbound: string, isFallback: boolean): string {
  return isFallback ? `fallback: ${outbound}` : `${match} -> ${outbound}`
}

/** 新规则默认插在 fallback 之前，避免生成永远匹配不到的规则。 */
export function addRoutingRule(text: string, match: string, outbound: string, isFallback = false): string {
  const line = routingLine(match, outbound, isFallback)
  if (!isFallback) {
    const fallback = parseRoutingRules(text).find((rule) => rule.isFallback)
    if (fallback) {
      const newline = lineEnding(text)
      return splice(text, fallback.lineStart, fallback.lineStart, INDENT + line + newline)
    }
  }
  return appendToSection(text, 'routing', [line])
}

export function setRoutingRule(
  text: string,
  rule: RoutingRule,
  match: string,
  outbound: string,
  isFallback = rule.isFallback,
): string {
  if (!rule.editable) return text
  return replaceLine(text, rule.lineStart, rule.lineEnd, routingLine(match, outbound, isFallback))
}

export function removeRoutingRule(text: string, rule: RoutingRule): string {
  if (!rule.editable) return text
  return removeLine(text, rule.lineStart, rule.lineEnd)
}

/** 在 group 节内新建子分组;group 节不存在时先创建。 */
export function addGroup(text: string, name: string, policy: string): string {
  return appendToSection(text, 'group', [name + ' {', INDENT + 'policy: ' + policy, '}'])
}

/**
 * 删除整个子分组。只有当分组独占若干整行时才连同缩进与行尾一起删,
 * 否则严格删除 `name { … }` 本身,避免连带同一行上的相邻分组。
 */
export function removeGroup(text: string, group: Group): string {
  const closing = group.section.bodyEnd + 1
  const lineStart = text.lastIndexOf('\n', group.section.nameStart) + 1
  const ownsLineStart = text.slice(lineStart, group.section.nameStart).trim() === ''
  let lineEnd = text.indexOf('\n', closing)
  if (lineEnd < 0) lineEnd = text.length
  const ownsLineEnd = text.slice(closing, lineEnd).trim() === ''
  if (!ownsLineStart || !ownsLineEnd) {
    return splice(text, group.section.nameStart, closing, '')
  }
  return splice(text, lineStart, lineEnd < text.length ? lineEnd + 1 : lineEnd, '')
}

/** 在子分组内已有声明之后插入一行,保持界面顺序与文件顺序一致。 */
function addDeclaration(text: string, group: Group, declaration: string): string {
  const newline = lineEnding(text)
  const lastDeclaration = [group.policy, ...group.filters]
    .filter((property): property is GroupProperty => property !== null)
    .reduce((furthest, property) => Math.max(furthest, property.lineEnd), -1)
  let at = lastDeclaration < 0 ? group.section.bodyStart : lastDeclaration
  // 行尾停在 '\n' 上,CRLF 文件要退到 '\r' 之前,否则插出 "\r\r\n"
  if (text[at] === '\n' && text[at - 1] === '\r') at -= 1
  return splice(text, at, at, newline + INDENT + INDENT + declaration)
}

export function setGroupPolicy(text: string, group: Group, policy: string): string {
  if (group.policy) {
    if (!group.policy.editable) return text
    return splice(text, group.policy.valueStart, group.policy.valueEnd, policy)
  }
  return addDeclaration(text, group, 'policy: ' + policy)
}

/** 替换第 index 条 filter;value 为空则删除该行,index 超出范围则追加一条。 */
export function setGroupFilter(text: string, group: Group, index: number, value: string): string {
  const existing = group.filters[index]
  if (existing) {
    if (!existing.editable) return text
    if (value === '') {
      const before = text.slice(existing.lineStart, existing.declarationStart)
      const after = text.slice(existing.declarationEnd, existing.lineEnd)
      if (before.trim() === '' && after.trim() === '') {
        return removeLine(text, existing.lineStart, existing.lineEnd)
      }
      return splice(text, existing.declarationStart, existing.declarationEnd, '')
    }
    return splice(text, existing.valueStart, existing.valueEnd, value)
  }
  if (value === '') return text
  return addDeclaration(text, group, 'filter: ' + value)
}
