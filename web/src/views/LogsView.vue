<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NEmpty,
  NIcon,
  NInput,
  NSelect,
  NSpace,
  NSpin,
  NTag,
  NText,
  useMessage,
} from 'naive-ui'
import { RefreshOutline, SearchOutline } from '@vicons/ionicons5'
import { getJSON } from '../api/client'
import type { LogEntry } from '../types/api'
import { formatDateTime } from '../utils/format'
import { useBackendStore } from '../stores/backend'

const message = useMessage()
// procd 部署读的是 logread 的系统日志缓冲区，既没有 journald 也没有 systemd
// 单元；标题照抄 systemd 那套，用户会去找一个本机不存在的东西。
const backend = useBackendStore()
const entries = ref<LogEntry[]>([])
const loading = ref(true)
const autoRefresh = ref(true)
const search = ref('')
const levelStorageKey = 'kdae-panel:log-level-filter'
const levelValues = new Set(['error', 'warning', 'info', 'debug', 'trace'])
const storedLevel = window.sessionStorage.getItem(levelStorageKey)
const level = ref<string | null>(storedLevel && levelValues.has(storedLevel) ? storedLevel : null)
const limit = ref(200)
const errorMessage = ref('')
let timer: number | undefined

const levelOptions = [
  { label: '全部级别', value: '' },
  { label: '错误', value: 'error' },
  { label: '警告', value: 'warning' },
  { label: '信息', value: 'info' },
  { label: '调试', value: 'debug' },
  { label: '跟踪', value: 'trace' },
]

const limitOptions = [100, 200, 300, 500].map((value) => ({ label: `${value} 条`, value }))
const filteredEntries = computed(() => {
  const query = search.value.trim().toLowerCase()
  return entries.value
    .filter((entry) => {
      const levelMatches = !level.value || entry.level === level.value
      const searchMatches = !query || entry.message.toLowerCase().includes(query) || entry.unit?.toLowerCase().includes(query)
      return levelMatches && searchMatches
    })
    .sort((left, right) => Date.parse(right.timestamp) - Date.parse(left.timestamp))
})

watch(level, (value) => {
  if (value) window.sessionStorage.setItem(levelStorageKey, value)
  else window.sessionStorage.removeItem(levelStorageKey)
})

function levelType(entry: LogEntry): 'error' | 'warning' | 'info' | 'default' {
  if (entry.priority >= 0 && entry.priority <= 3) return 'error'
  if (entry.priority === 4) return 'warning'
  if (entry.priority >= 5 && entry.priority <= 6) return 'info'
  return 'default'
}

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    entries.value = await getJSON<LogEntry[]>(`/api/v1/logs?limit=${limit.value}`)
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取日志失败'
    if (!silent) message.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

function schedule() {
  window.clearInterval(timer)
  timer = window.setInterval(() => {
    if (autoRefresh.value && document.visibilityState === 'visible') void load(true)
  }, 5000)
}

onMounted(() => {
  void backend.ensure()
  void load()
  schedule()
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<template>
  <div class="page-stack logs-page">
    <div class="page-toolbar">
      <div>
        <h2>{{ backend.isProcd ? '服务日志' : 'journald 日志' }}</h2>
        <NText depth="3">
          {{ backend.isProcd ? '读取系统日志缓冲区中 dae 的近期日志' : '读取 dae systemd 单元的近期结构化日志' }}
        </NText>
      </div>
      <NSpace align="center">
        <NCheckbox v-model:checked="autoRefresh">每 5 秒刷新</NCheckbox>
        <NButton secondary :loading="loading" @click="load()">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>刷新
        </NButton>
      </NSpace>
    </div>

    <NAlert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</NAlert>

    <NCard class="logs-card" content-style="padding: 0;">
      <div class="filter-bar">
        <NInput v-model:value="search" clearable placeholder="搜索日志内容" class="log-search">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
        <NSelect v-model:value="level" clearable :options="levelOptions" placeholder="全部级别" class="log-select" />
        <NSelect v-model:value="limit" :options="limitOptions" class="log-limit" @update:value="load()" />
        <NText depth="3">最新在前 · 显示 {{ filteredEntries.length }} / {{ entries.length }}</NText>
      </div>
      <NSpin :show="loading">
        <div v-if="filteredEntries.length" class="log-stream">
          <div v-for="(entry, index) in filteredEntries" :key="`${entry.timestamp}-${index}`" class="log-row">
            <time>{{ formatDateTime(entry.timestamp) }}</time>
            <NTag size="tiny" :type="levelType(entry)" :bordered="false">{{ entry.level }}</NTag>
            <span class="log-source">{{ entry.pid ? `PID ${entry.pid}` : entry.unit || 'dae' }}</span>
            <pre>{{ entry.message }}</pre>
          </div>
        </div>
        <NEmpty v-else description="没有符合条件的日志" class="empty-state" />
      </NSpin>
    </NCard>
  </div>
</template>
