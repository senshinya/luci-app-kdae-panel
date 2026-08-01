<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NAlert, NButton, NCard, NGrid, NGridItem, NIcon, NSpin, NTag, NText, useMessage } from 'naive-ui'
import { DownloadOutline, ReaderOutline, RefreshOutline } from '@vicons/ionicons5'
import { getDownload, getJSON } from '../api/client'
import type { DiagnosticItem, DiagnosticLevel, DiagnosticReport } from '../types/api'
import { formatDateTime } from '../utils/format'
import { useBackendStore } from '../stores/backend'

const message = useMessage()
// 副标题是这份报告的取证清单。procd 上既没有 systemd 也没有 journald，照抄
// systemd 那套等于告诉一个正在排障的人：去看两个他机器上不存在的东西。
const backend = useBackendStore()
const router = useRouter()
const loading = ref(true)
const exporting = ref(false)
const errorText = ref('')
const report = ref<DiagnosticReport | null>(null)

const levelMeta: Record<DiagnosticLevel, { label: string; type: 'success' | 'warning' | 'error' | 'default' }> = {
  ok: { label: '正常', type: 'success' },
  warning: { label: '注意', type: 'warning' },
  error: { label: '故障', type: 'error' },
  unknown: { label: '未知', type: 'default' },
}

const overallCopy = computed(() => {
  if (!report.value) return ''
  switch (report.value.overall) {
    case 'error': return '发现需要先处理的故障'
    case 'warning': return '基础检查通过，但有需要确认的项目'
    case 'unknown': return '部分信息无法读取，结果不完整'
    default: return '公开接口基础检查全部通过'
  }
})

const categories = computed(() => {
  const grouped = new Map<string, DiagnosticItem[]>()
  for (const item of report.value?.items || []) {
    const values = grouped.get(item.category) || []
    values.push(item)
    grouped.set(item.category, values)
  }
  return [...grouped.entries()].map(([name, items]) => ({ name, items }))
})

async function load() {
  loading.value = true
  try {
    report.value = await getJSON<DiagnosticReport>('/api/v1/diagnostics/report')
    errorText.value = ''
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : '生成故障诊断报告失败'
  } finally {
    loading.value = false
  }
}

async function exportSysdump() {
  exporting.value = true
  try {
    const response = await getDownload('/api/v1/diagnostics/sysdump')
    const url = URL.createObjectURL(response.blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = response.filename
    anchor.click()
    URL.revokeObjectURL(url)
    message.success('诊断归档已生成')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '生成诊断归档失败')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  void backend.ensure()
  void load()
})
</script>

<template>
  <div class="page-stack diagnostics-page">
    <div class="page-toolbar">
      <div>
        <h2>故障诊断</h2>
        <NText depth="3">
          {{ backend.isProcd
            ? '基于 procd、dae 公开命令、配置、Geo、logread 与 Linux 标准接口检查'
            : '基于 systemd、dae 公开命令、配置、Geo、journald 与 Linux 标准接口检查' }}
        </NText>
      </div>
      <div class="diagnostics-actions">
        <NButton secondary :loading="loading" @click="load">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>重新检查
        </NButton>
        <NButton secondary :loading="exporting" @click="exportSysdump">
          <template #icon><NIcon><DownloadOutline /></NIcon></template>导出 sysdump
        </NButton>
      </div>
    </div>

    <NAlert v-if="errorText" type="error" :bordered="false">{{ errorText }}</NAlert>

    <NSpin :show="loading">
      <template v-if="report">
        <div class="diagnostics-summary" :class="`diagnostics-summary-${report.overall}`">
          <div>
            <NTag :type="levelMeta[report.overall].type" :bordered="false">
              {{ levelMeta[report.overall].label }}
            </NTag>
            <strong>{{ overallCopy }}</strong>
            <NText depth="3">检查于 {{ formatDateTime(report.generatedAt) }}</NText>
          </div>
          <dl>
            <div><dt>正常</dt><dd>{{ report.counts.ok }}</dd></div>
            <div><dt>注意</dt><dd>{{ report.counts.warning }}</dd></div>
            <div><dt>故障</dt><dd>{{ report.counts.error }}</dd></div>
            <div><dt>未知</dt><dd>{{ report.counts.unknown }}</dd></div>
          </dl>
        </div>

        <section v-for="category in categories" :key="category.name" class="diagnostics-section">
          <h3>{{ category.name }}</h3>
          <NGrid responsive="screen" cols="1 l:2" :x-gap="14" :y-gap="14" class="equal-height-grid">
            <NGridItem v-for="item in category.items" :key="item.id">
              <NCard class="diagnostic-item" size="small">
                <div class="diagnostic-item-head">
                  <div>
                    <strong>{{ item.title }}</strong>
                    <span>{{ item.summary }}</span>
                  </div>
                  <NTag size="small" :type="levelMeta[item.level].type" :bordered="false">
                    {{ levelMeta[item.level].label }}
                  </NTag>
                </div>
                <ul v-if="item.details?.length" class="diagnostic-details">
                  <li v-for="(detail, index) in item.details" :key="index">{{ detail }}</li>
                </ul>
                <div v-if="item.suggestion" class="diagnostic-suggestion">
                  <span>建议</span>{{ item.suggestion }}
                </div>
              </NCard>
            </NGridItem>
          </NGrid>
        </section>

        <div class="diagnostics-footer-action">
          <NButton secondary @click="router.push({ name: 'logs' })">
            <template #icon><NIcon><ReaderOutline /></NIcon></template>查看完整运行日志
          </NButton>
        </div>
      </template>
    </NSpin>
  </div>
</template>
