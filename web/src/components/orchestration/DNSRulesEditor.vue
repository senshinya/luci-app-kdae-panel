<script setup lang="ts">
import { computed, reactive } from 'vue'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import { NButton, NCheckbox, NIcon, NInput, NSelect, NText, type SelectOption } from 'naive-ui'
import {
  buildVisualDNSMatcher,
  defaultVisualMatcher,
  DNS_INTERNAL_MATCHERS,
  newDNSRule,
  parseVisualDNSMatcher,
  visualMatcherKeys,
  type DNSInternalMatcher,
  type DNSRule,
  type DNSUpstream,
  type DNSVisualMatcher,
  type DNSVisualMatcherKind,
} from '../../utils/dns'

const props = defineProps<{
  kind: 'request' | 'response'
  upstreams: DNSUpstream[]
  requestMatchers?: Set<DNSInternalMatcher>
  requestMatchersKnown?: boolean
}>()
const rules = defineModel<DNSRule[]>({ required: true })
const matcherDrafts = reactive(new Map<string, DNSVisualMatcher>())

const builtinOptions = computed<SelectOption[]>(() => props.kind === 'request'
  ? [
      { label: 'asis · 使用 dae 基础解析', value: 'asis' },
      { label: 'reject · 拒绝解析', value: 'reject' },
    ]
  : [
      { label: 'accept · 接受响应', value: 'accept' },
      { label: 'reject · 丢弃响应', value: 'reject' },
    ])
const targetOptions = computed<SelectOption[]>(() => [
  ...upstreamOptions.value,
  ...builtinOptions.value,
])
const upstreamOptions = computed<SelectOption[]>(() => props.upstreams
  .map((upstream) => ({ label: `${upstream.name || '未命名'} · 上游`, value: upstream.name })))

const matcherKindOptions = computed<SelectOption[]>(() => [
  { label: '域名 · qname', value: 'qname' },
  { label: '查询类型 · qtype', value: 'qtype' },
  {
    label: '订阅拉取 · sub',
    value: 'sub',
    disabled: !internalSupported('sub'),
  },
  {
    label: '节点解析 · node',
    value: 'node',
    disabled: !internalSupported('node'),
  },
  {
    label: '订阅节点 · subnode',
    value: 'subnode',
    disabled: !internalSupported('subnode'),
  },
  { label: '原始表达式', value: 'raw' },
])

const keyLabels = new Map<string, string>([
  ['*', '全部'],
  ['', '精确值'],
  ['full', '完整域名'],
  ['suffix', '域名后缀'],
  ['keyword', '包含关键字'],
  ['regex', '正则表达式'],
  ['geosite', 'GeoSite 集合'],
  ['tag', '订阅标签'],
  ['tag_regex', '订阅标签正则'],
  ['subtag', '来源订阅'],
  ['subtag_regex', '来源订阅正则'],
  ['name', '节点名称'],
  ['name_keyword', '节点名称包含'],
  ['name_regex', '节点名称正则'],
  ['link_keyword', '链接包含'],
  ['link_regex', '链接正则'],
])

function internalSupported(matcher: DNSInternalMatcher): boolean {
  return props.requestMatchersKnown === true && props.requestMatchers?.has(matcher) === true
}

function matcherDraft(rule: DNSRule): DNSVisualMatcher {
  return matcherDrafts.get(rule.id) || parseVisualDNSMatcher(rule.matcher)
}

function isInternal(kind: DNSVisualMatcherKind): kind is DNSInternalMatcher {
  return DNS_INTERNAL_MATCHERS.includes(kind as DNSInternalMatcher)
}

function matcherUnavailable(rule: DNSRule): boolean {
  const kind = matcherDraft(rule).kind
  return isInternal(kind) && !internalSupported(kind)
}

function matcherKeyOptions(rule: DNSRule): SelectOption[] {
  return visualMatcherKeys(matcherDraft(rule).kind)
    .map((key) => ({ label: keyLabels.get(key) || key, value: key }))
}

function matcherPlaceholder(rule: DNSRule): string {
  const draft = matcherDraft(rule)
  switch (draft.key) {
    case 'geosite': return '例如 cn'
    case 'tag':
    case 'subtag': return '选择或填写订阅标签'
    case 'name': return '填写节点名称'
    case 'name_keyword': return '例如 hk'
    case 'tag_regex':
    case 'subtag_regex':
    case 'name_regex':
    case 'link_regex':
    case 'regex': return '填写正则表达式'
    case 'link_keyword': return '填写链接关键字'
    default: return draft.kind === 'qtype' ? '例如 a' : '填写匹配值'
  }
}

function writeVisual(rule: DNSRule, draft: DNSVisualMatcher) {
  matcherDrafts.set(rule.id, draft)
  try {
    rule.matcher = buildVisualDNSMatcher(draft)
  } catch {
    // 输入过程允许暂时为空，最终应用仍由 buildDNSBody 统一校验。
    rule.matcher = draft.kind === 'raw' ? draft.value : `${draft.kind}(${draft.key === '*' ? '' : `${draft.key}: ${draft.value}`})`
  }
}

function setMatcherKind(rule: DNSRule, value: string) {
  const kind = value as DNSVisualMatcherKind
  if (kind === 'raw') {
    matcherDrafts.set(rule.id, { kind, key: '', value: rule.matcher })
    return
  }
  const draft = defaultVisualMatcher(kind)
  writeVisual(rule, draft)
  if (isInternal(kind) && !props.upstreams.some((upstream) => upstream.name === rule.target)) {
    rule.target = props.upstreams[0]?.name || ''
  }
}

function setMatcherKey(rule: DNSRule, key: string) {
  const current = matcherDraft(rule)
  if (current.kind === 'raw') return
  writeVisual(rule, { ...current, key, value: key === '*' ? '' : current.value })
}

function setMatcherValue(rule: DNSRule, value: string) {
  const current = matcherDraft(rule)
  writeVisual(rule, { ...current, value })
}

function ruleTargetOptions(rule: DNSRule): SelectOption[] {
  return props.kind === 'request' && isInternal(matcherDraft(rule).kind)
    ? upstreamOptions.value
    : targetOptions.value
}

function add() {
  rules.value = [...rules.value, newDNSRule(props.kind)]
}

function remove(index: number) {
  matcherDrafts.delete(rules.value[index].id)
  rules.value = rules.value.filter((_, current) => current !== index)
}

function setFallback(rule: DNSRule, fallback: boolean) {
  rule.fallback = fallback
  if (fallback) {
    rule.matcher = ''
    matcherDrafts.delete(rule.id)
  } else if (rule.matcher === '') {
    writeVisual(rule, defaultVisualMatcher('qname'))
  }
}
</script>

<template>
  <section class="dns-editor-section" :data-testid="`dns-${kind}-rules-editor`">
    <div class="dns-section-head">
      <div>
        <strong>{{ kind === 'request' ? '请求路由' : '响应路由' }}</strong>
        <NText depth="3">规则从上到下匹配，最后可设置一个 fallback。</NText>
      </div>
      <NButton size="small" secondary @click="add">
        <template #icon><NIcon><AddOutline /></NIcon></template>添加规则
      </NButton>
    </div>
    <div v-if="rules.length === 0" class="orchestrate-empty dns-empty">
      <NText depth="3">没有规则，dae 将使用默认解析行为。</NText>
    </div>
    <div v-else class="dns-rule-list">
      <div v-for="(rule, index) in rules" :key="rule.id" class="dns-rule-row">
        <NCheckbox :checked="rule.fallback" @update:checked="setFallback(rule, $event)">fallback</NCheckbox>
        <div v-if="kind === 'request'" class="dns-rule-condition" :class="{ raw: matcherDraft(rule).kind === 'raw' }">
          <NSelect
            :value="matcherDraft(rule).kind"
            :options="matcherKindOptions"
            :disabled="rule.fallback"
            aria-label="DNS 条件类型"
            @update:value="setMatcherKind(rule, $event)"
          />
          <NSelect
            v-if="matcherDraft(rule).kind !== 'raw' && matcherKeyOptions(rule).length > 1"
            :value="matcherDraft(rule).key"
            :options="matcherKeyOptions(rule)"
            :disabled="rule.fallback || matcherUnavailable(rule)"
            aria-label="DNS 条件字段"
            @update:value="setMatcherKey(rule, $event)"
          />
          <NInput
            v-if="matcherDraft(rule).kind === 'raw'"
            :value="rule.matcher"
            class="mono dns-rule-matcher"
            :disabled="rule.fallback"
            placeholder="输入 dae 匹配表达式"
            aria-label="DNS 原始匹配表达式"
            @update:value="setMatcherValue(rule, $event)"
          />
          <NInput
            v-else-if="matcherDraft(rule).key !== '*'"
            :value="matcherDraft(rule).value"
            class="mono dns-rule-matcher"
            :disabled="rule.fallback || matcherUnavailable(rule)"
            :placeholder="matcherPlaceholder(rule)"
            aria-label="DNS 条件值"
            @update:value="setMatcherValue(rule, $event)"
          />
          <NText v-if="matcherUnavailable(rule)" type="error" class="dns-rule-compat">
            当前 dae 未确认支持 {{ matcherDraft(rule).kind }}()；原文仍会保留。
          </NText>
        </div>
        <NInput
          v-else
          v-model:value="rule.matcher"
          class="mono dns-rule-matcher"
          :disabled="rule.fallback"
          placeholder="upstream(googledns) 或 ip(geoip:private)"
          aria-label="DNS 匹配条件"
        />
        <NSelect
          v-model:value="rule.target"
          :options="ruleTargetOptions(rule)"
          filterable
          :placeholder="kind === 'request' ? '目标上游 / asis / reject' : '目标上游 / accept / reject'"
          aria-label="DNS 规则目标"
        />
        <NButton quaternary circle type="error" title="删除 DNS 规则" aria-label="删除 DNS 规则" @click="remove(index)">
          <template #icon><NIcon><TrashOutline /></NIcon></template>
        </NButton>
      </div>
    </div>
    <NText depth="3" class="dns-rule-hint">
      <template v-if="kind === 'request' && !requestMatchersKnown">
        当前 dae 未在配置结构中声明 sub、node、subnode 能力，相关可视化选项已禁用；进阶模式仍会原样保留已有规则。
      </template>
      <template v-else-if="kind === 'request'">
        sub 用于订阅拉取，node 用于节点解析，subnode 用于订阅节点且优先于 node。
      </template>
      <template v-else>
        响应规则支持 qname、qtype、upstream、ip 等匹配器。
      </template>
    </NText>
  </section>
</template>
