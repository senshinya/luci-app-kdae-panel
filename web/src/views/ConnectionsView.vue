<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NEmpty,
  NGrid,
  NGridItem,
  NIcon,
  NInput,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSkeleton,
  NSpace,
  NSpin,
  NTag,
  NText,
  useMessage,
  type DataTableColumns,
  type DataTableSortState,
} from 'naive-ui'
import { ArrowDownOutline, ArrowUpOutline, RefreshOutline, SearchOutline } from '@vicons/ionicons5'
import { getJSON } from '../api/client'
import type { ConnectionEvent, ConnectionsResponse } from '../types/api'
import { formatDateTime, formatElapsedSince } from '../utils/format'
import { useMobileViewport } from '../composables/useMobileViewport'

const message = useMessage()
const mobile = useMobileViewport()
const data = ref<ConnectionsResponse | null>(null)
const loading = ref(true)
const autoRefresh = ref(true)
const errorMessage = ref('')
const windowMinutes = ref(5)
const search = ref('')
const outbound = ref<string | null>(null)
const network = ref<string | null>(null)
const sortOrder = ref<'descend' | 'ascend'>('descend')
const now = ref(Date.now())
let timer: number | undefined
let clock: number | undefined

// 0 表示不按时间过滤，展示存储里的全部流水。
const windowOptions = [
  { label: '最近 1 分钟', value: 1 },
  { label: '最近 5 分钟', value: 5 },
  { label: '最近 15 分钟', value: 15 },
  { label: '最近 1 小时', value: 60 },
  { label: '全部', value: 0 },
]
const networkOptions = [
  { label: '全部协议', value: '' },
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]

// 移动端不分页，一次渲染太多卡片会明显卡顿，超出部分提示用筛选缩小范围。
const mobileListCap = 100

const entries = computed(() => data.value?.entries ?? [])
const summary = computed(() => data.value?.summary)

// 分布卡的维度。按远端 IP 分组的出站 socket 曾经也在这里，但只有几个节点时
// 它退化成"全压在主用节点上"，说的和上面的总数是同一件事，因此换成了这四个
// 真正有区分度的维度。
const dimension = ref<'clients' | 'domains' | 'nodes' | 'groups'>('domains')
const dimensionOptions = [
  { label: '目的域名', value: 'domains' },
  { label: '客户端', value: 'clients' },
  { label: '节点', value: 'nodes' },
  { label: '出站组', value: 'groups' },
]
const distributionCap = 8
const distribution = computed(() => (data.value?.[dimension.value] ?? []).slice(0, distributionCap))
const distributionTotal = computed(() => (data.value?.[dimension.value] ?? []).length)
const logLevelInsufficient = computed(() => {
  const level = data.value?.logLevel
  return !!level && !['info', 'debug', 'trace'].includes(level)
})

const outboundOptions = computed(() => {
  const names = (data.value?.groups ?? []).map((item) => item.key)
  return [{ label: '全部出站', value: '' }, ...names.map((name) => ({ label: name, value: name }))]
})

const filteredEntries = computed(() => {
  const query = search.value.trim().toLowerCase()
  const cutoff = windowMinutes.value > 0 ? now.value - windowMinutes.value * 60_000 : 0
  const rows = entries.value.filter((entry) => {
    if (cutoff && Date.parse(entry.at) < cutoff) return false
    if (network.value && !entry.network.startsWith(network.value)) return false
    if (outbound.value && entry.outbound !== outbound.value) return false
    if (query) {
      const haystack = [entry.src, entry.dst, entry.dstAddr, entry.sniffed, entry.dialer, entry.outbound, entry.policy, entry.pname, entry.mac]
      if (!haystack.some((value) => value?.toLowerCase().includes(query))) return false
    }
    return true
  })
  const direction = sortOrder.value === 'descend' ? -1 : 1
  return rows.sort((left, right) => {
    const delta = Date.parse(left.at) - Date.parse(right.at)
    if (delta !== 0) return delta * direction
    return left.src < right.src ? -1 : 1
  })
})

const mobileEntries = computed(() => filteredEntries.value.slice(0, mobileListCap))

function outboundType(entry: ConnectionEvent): 'success' | 'error' | 'primary' | 'default' {
  if (!entry.outbound) return 'default'
  if (entry.outbound === 'direct') return 'success'
  if (entry.outbound === 'block') return 'error'
  return 'primary'
}

function timeLabel(entry: ConnectionEvent): string {
  return (entry.approxTime ? '≈ ' : '') + formatDateTime(entry.at)
}

function agoLabel(entry: ConnectionEvent): string {
  return formatElapsedSince(entry.at, now.value) + '前'
}

function destinationTitle(entry: ConnectionEvent): string {
  return entry.sniffed || entry.dst
}

function sourceDetail(entry: ConnectionEvent): string {
  return [entry.pname, entry.mac].filter(Boolean).join(' · ')
}

const columns = computed<DataTableColumns<ConnectionEvent>>(() => [
  {
    // 目的放在首列并加重：用户看这张表是想知道"谁在访问什么"，
    // 域名是那个答案，时间只是限定语。
    title: '目的',
    key: 'dst',
    minWidth: 260,
    ellipsis: { tooltip: true },
    render: (row) => {
      const lines = [h('div', { class: 'conn-primary' }, destinationTitle(row))]
      if (row.dstAddr && row.dstAddr !== destinationTitle(row)) {
        lines.push(h('div', { class: 'conn-secondary mono' }, row.dstAddr))
      }
      return lines
    },
  },
  {
    title: '源',
    key: 'src',
    minWidth: 190,
    render: (row) => {
      const lines = [h('div', { class: 'mono' }, row.src)]
      const detail = sourceDetail(row)
      if (detail) lines.push(h('div', { class: 'conn-secondary' }, detail))
      return lines
    },
  },
  {
    title: '出站',
    key: 'outbound',
    minWidth: 160,
    render: (row) => {
      const lines = [h(NTag, { size: 'small', type: outboundType(row), bordered: false }, { default: () => row.outbound })]
      const detail = [row.dialer, row.policy].filter(Boolean).join(' · ')
      if (detail) lines.push(h('div', { class: 'conn-secondary' }, detail))
      return lines
    },
  },
  {
    title: '协议',
    key: 'network',
    width: 78,
    render: (row) => h(NTag, { size: 'tiny', bordered: false }, { default: () => row.network }),
  },
  {
    title: '建立时间',
    key: 'at',
    width: 176,
    sorter: true,
    sortOrder: sortOrder.value,
    render: (row) => [
      h('div', { class: 'conn-secondary' }, timeLabel(row)),
      h('div', { class: 'conn-secondary' }, agoLabel(row)),
    ],
  },
])

function handleSorter(sorter: DataTableSortState | DataTableSortState[] | null) {
  const state = Array.isArray(sorter) ? sorter[0] : sorter
  // naive-ui 的第三态（取消排序）对时间列没有意义，折回默认的最新在前。
  sortOrder.value = state?.order === 'ascend' ? 'ascend' : 'descend'
}

function toggleSort() {
  sortOrder.value = sortOrder.value === 'descend' ? 'ascend' : 'descend'
}

async function load(silent = false) {
  if (!silent) loading.value = true
  try {
    data.value = await getJSON<ConnectionsResponse>('/api/v1/connections')
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取连接活动失败'
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
  void load()
  schedule()
  clock = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
})
onBeforeUnmount(() => {
  window.clearInterval(timer)
  window.clearInterval(clock)
})
</script>

<template>
  <div class="page-stack connections-page">
    <div class="page-toolbar">
      <div>
        <h2>连接活动</h2>
        <NText depth="3">dae 记录的连接建立流水，按客户端、域名、节点与出站组分布</NText>
      </div>
      <NSpace align="center">
        <NCheckbox v-model:checked="autoRefresh">每 5 秒刷新</NCheckbox>
        <NButton secondary :loading="loading" @click="load()">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>刷新
        </NButton>
      </NSpace>
    </div>

    <NAlert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</NAlert>
    <NAlert v-if="logLevelInsufficient" type="warning">
      当前 dae 日志级别为 {{ data?.logLevel }}，不会输出连接日志。请在配置管理里把 global 的
      log_level 调整为 info 并重载，连接流水才能被采集；出站连接分布不受影响。
    </NAlert>
    <NAlert v-else-if="data && !data.snapshotOk" type="warning">
      无法读取 dae 的 socket 快照，出站连接分布暂不可用；下方流水不受影响。
    </NAlert>

    <NGrid responsive="screen" cols="1 s:3" :x-gap="14" :y-gap="14">
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">当前出站连接</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.outboundSockets ?? '—' }}</strong>
          <small>dae 此刻持有的连接，实时值 · UDP {{ summary?.udpSockets ?? '—' }}</small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">窗口内新建连接</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.windowEvents ?? '—' }}</strong>
          <small>
            面板运行期内累积
            <template v-if="data?.dropped">· {{ data.dropped }} 行未能解析</template>
          </small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">活跃节点</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.activeNodes ?? '—' }}</strong>
          <small>窗口内被选中过的节点数</small>
        </NCard>
      </NGridItem>
    </NGrid>

    <NCard size="small">
      <template #header>
        <NRadioGroup v-model:value="dimension" size="small">
          <NRadioButton v-for="option in dimensionOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </NRadioButton>
        </NRadioGroup>
      </template>
      <template #header-extra>
        <NText depth="3">窗口内新建连接数 · 共 {{ distributionTotal }} 项</NText>
      </template>
      <div v-if="distribution.length" class="conn-bars">
        <div v-for="item in distribution" :key="item.key" class="conn-bar">
          <span class="conn-bar-label">
            <span :class="{ mono: dimension === 'clients' }">{{ item.key }}</span>
            <NText v-if="item.note" depth="3" class="conn-bar-note">{{ item.note }}</NText>
          </span>
          <span class="conn-bar-track">
            <span
              class="conn-bar-fill"
              :style="{ width: `${Math.round((item.count / distribution[0].count) * 100)}%` }"
            />
          </span>
          <strong>{{ item.count }}</strong>
        </div>
      </div>
      <NText v-else depth="3">窗口内没有连接记录</NText>
    </NCard>

    <NCard content-style="padding: 0;">
      <div class="filter-bar">
        <NSelect v-model:value="windowMinutes" :options="windowOptions" class="log-select" />
        <NInput v-model:value="search" clearable placeholder="搜索地址、域名、节点或进程" class="log-search">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
        <NSelect v-model:value="outbound" clearable :options="outboundOptions" placeholder="全部出站" class="log-select" />
        <NSelect v-model:value="network" clearable :options="networkOptions" placeholder="全部协议" class="log-select" />
        <NButton v-if="mobile" size="small" quaternary @click="toggleSort">
          <template #icon>
            <NIcon><ArrowDownOutline v-if="sortOrder === 'descend'" /><ArrowUpOutline v-else /></NIcon>
          </template>
          {{ sortOrder === 'descend' ? '最新在前' : '最早在前' }}
        </NButton>
        <NText depth="3">显示 {{ filteredEntries.length }} / {{ entries.length }}</NText>
      </div>

      <NDataTable
        v-if="!mobile"
        :columns="columns"
        :data="filteredEntries"
        :loading="loading && !data"
        :row-key="(row: ConnectionEvent) => `${row.network}|${row.src}|${row.dst}|${row.at}`"
        :scroll-x="920"
        :bordered="false"
        :pagination="{ pageSize: 50 }"
        @update:sorter="handleSorter"
      >
        <template #empty>
          <NEmpty class="empty-state" description="所选时间窗内没有连接记录" />
        </template>
      </NDataTable>

      <NSpin v-else :show="loading && !data">
        <div v-if="mobileEntries.length" class="mobile-record-list">
          <article v-for="entry in mobileEntries" :key="`${entry.network}|${entry.src}|${entry.dst}|${entry.at}`" class="mobile-record">
            <div class="mobile-record-head">
              <div class="mobile-record-title">
                <span class="conn-mobile-destination">{{ destinationTitle(entry) }}</span>
                <NTag size="small" :type="outboundType(entry)" :bordered="false">{{ entry.outbound }}</NTag>
              </div>
              <NTag size="tiny" :bordered="false">{{ entry.network }}</NTag>
            </div>
            <p class="mobile-record-description mono">{{ entry.src }} → {{ entry.dstAddr || entry.dst }}</p>
            <div class="mobile-record-meta">
              <span v-if="entry.dialer">节点<strong>{{ entry.dialer }}</strong></span>
              <span>建立<strong>{{ timeLabel(entry) }}</strong></span>
              <span>{{ agoLabel(entry) }}</span>
              <span v-if="sourceDetail(entry)">来源<strong>{{ sourceDetail(entry) }}</strong></span>
            </div>
          </article>
          <NText v-if="filteredEntries.length > mobileListCap" depth="3" class="conn-mobile-cap">
            仅显示前 {{ mobileListCap }} 条，可用筛选缩小范围
          </NText>
        </div>
        <NEmpty v-else class="mobile-empty" description="所选时间窗内没有连接记录" />
      </NSpin>
    </NCard>
  </div>
</template>
