<script setup lang="ts">
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import { NButton, NIcon, NInput, NText } from 'naive-ui'
import { newDNSUpstream, type DNSUpstream } from '../../utils/dns'

const upstreams = defineModel<DNSUpstream[]>({ required: true })

function add() {
  upstreams.value = [...upstreams.value, newDNSUpstream()]
}

function remove(index: number) {
  upstreams.value = upstreams.value.filter((_, current) => current !== index)
}
</script>

<template>
  <section class="dns-editor-section" data-testid="dns-upstreams-editor">
    <div class="dns-section-head">
      <div>
        <strong>DNS 上游</strong>
        <NText depth="3">为每个上游命名，规则会引用这些名称。</NText>
      </div>
      <NButton size="small" secondary @click="add">
        <template #icon><NIcon><AddOutline /></NIcon></template>添加上游
      </NButton>
    </div>
    <div v-if="upstreams.length === 0" class="orchestrate-empty dns-empty">
      <NText depth="3">还没有上游；request 规则可以使用 asis 或 reject。</NText>
    </div>
    <div v-else class="dns-upstream-list">
      <div v-for="(upstream, index) in upstreams" :key="upstream.id" class="dns-upstream-row">
        <NInput v-model:value="upstream.name" class="mono" placeholder="名称，如 alidns" aria-label="上游名称" />
        <NInput v-model:value="upstream.url" class="mono" placeholder="udp://223.5.5.5:53 或 https://dns.google/dns-query" aria-label="上游地址" />
        <NButton quaternary circle type="error" title="删除上游" aria-label="删除上游" @click="remove(index)">
          <template #icon><NIcon><TrashOutline /></NIcon></template>
        </NButton>
      </div>
    </div>
  </section>
</template>
