<script setup lang="ts">
import { computed } from 'vue'
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import { NButton, NCheckbox, NIcon, NInput, NSelect, NText, type SelectOption } from 'naive-ui'
import { newDNSRule, type DNSRule, type DNSUpstream } from '../../utils/dns'

const props = defineProps<{
  kind: 'request' | 'response'
  upstreams: DNSUpstream[]
}>()
const rules = defineModel<DNSRule[]>({ required: true })

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
  ...props.upstreams.map((upstream) => ({ label: `${upstream.name || '未命名'} · 上游`, value: upstream.name })),
  ...builtinOptions.value,
])

function add() {
  rules.value = [...rules.value, newDNSRule(props.kind)]
}

function remove(index: number) {
  rules.value = rules.value.filter((_, current) => current !== index)
}

function setFallback(rule: DNSRule, fallback: boolean) {
  rule.fallback = fallback
  if (fallback) rule.matcher = ''
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
        <NInput
          v-model:value="rule.matcher"
          class="mono dns-rule-matcher"
          :disabled="rule.fallback"
          :placeholder="kind === 'request' ? 'qname(geosite:cn) 或 qtype(a, aaaa)' : 'upstream(googledns) 或 ip(geoip:private)'"
          aria-label="DNS 匹配条件"
        />
        <NSelect
          v-model:value="rule.target"
          :options="targetOptions"
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
      {{ kind === 'request' ? '请求规则支持 qname、qtype、sub、node、subnode 等匹配器。' : '响应规则支持 qname、qtype、upstream、ip 等匹配器。' }}
    </NText>
  </section>
</template>
