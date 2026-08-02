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
import type { ConnectionRecord, ConnectionsResponse } from '../types/api'
import { formatDateTime, formatElapsedSince } from '../utils/format'
import { useMobileViewport } from '../composables/useMobileViewport'

const message = useMessage()
const mobile = useMobileViewport()
const data = ref<ConnectionsResponse | null>(null)
const loading = ref(true)
const autoRefresh = ref(true)
const errorMessage = ref('')
const view = ref<'live' | 'all'>('live')
const search = ref('')
const outbound = ref<string | null>(null)
const network = ref<string | null>(null)
const limit = ref(500)
const sortOrder = ref<'descend' | 'ascend'>('descend')
const now = ref(Date.now())
let timer: number | undefined
let clock: number | undefined

const limitOptions = [100, 200, 500, 1000, 2000].map((value) => ({ label: `${value} 条`, value }))
const networkOptions = [
  { label: '全部协议', value: '' },
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]

// 移动端不分页，一次渲染太多卡片会明显卡顿，超出部分提示用筛选缩小范围。
const mobileListCap = 100

const entries = computed(() => data.value?.entries ?? [])
const summary = computed(() => data.value?.summary)
const logLevelInsufficient = computed(() => {
  const level = data.value?.logLevel
  return !!level && !['info', 'debug', 'trace'].includes(level)
})

const outboundOptions = computed(() => {
  const names = new Set<string>()
  for (const entry of entries.value) {
    if (entry.outbound) names.add(entry.outbound)
  }
  return [
    { label: '全部出站', value: '' },
    ...[...names].sort().map((name) => ({ label: name, value: name })),
  ]
})

const filteredEntries = computed(() => {
  const query = search.value.trim().toLowerCase()
  const rows = entries.value.filter((entry) => {
    // UDP 没有关闭语义、被 eBPF 卸载的连接未必留有 socket，两类都判不了存活，
    // 因此"存活中"只收 socket 确定命中的 TCP（含日志已滚掉的孤儿）。
    if (view.value === 'live' && entry.status !== 'live' && entry.status !== 'orphan') return false
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
    const delta = Date.parse(left.firstSeen) - Date.parse(right.firstSeen)
    if (delta !== 0) return delta * direction
    return left.src < right.src ? -1 : 1
  })
})

const mobileEntries = computed(() => filteredEntries.value.slice(0, mobileListCap))

function statusMeta(record: ConnectionRecord): { label: string; type: 'success' | 'error' | 'warning' | 'default' } {
  switch (record.status) {
    case 'live':
      return { label: '存活', type: 'success' }
    case 'orphan':
      return { label: '存活 · 无日志', type: 'success' }
    case 'closed':
      return { label: '已结束', type: 'default' }
    default:
      return record.network.startsWith('udp')
        ? { label: 'UDP · 不判定', type: 'default' }
        : { label: record.offloaded ? '未知 · 已卸载' : '未知', type: 'warning' }
  }
}

function outboundType(record: ConnectionRecord): 'success' | 'error' | 'primary' | 'default' {
  if (!record.outbound) return 'default'
  if (record.outbound === 'direct') return 'success'
  if (record.outbound === 'block') return 'error'
  return 'primary'
}

function isAlive(record: ConnectionRecord): boolean {
  return record.status === 'live' || record.status === 'orphan'
}

function firstSeenLabel(record: ConnectionRecord): string {
  const prefix = record.approxFirstSeen ? '≥ ' : ''
  return prefix + formatDateTime(record.firstSeen)
}

// 起点是近似值时，时长也只能是下界；两处口径必须一致，否则"开始时间带 ≥、
// 已持续却是精确值"会读成面板知道它活了多久。
function elapsedLabel(record: ConnectionRecord): string {
  const prefix = record.approxFirstSeen ? '≥ ' : ''
  return prefix + formatElapsedSince(record.firstSeen, now.value)
}

function destinationTitle(record: ConnectionRecord): string {
  if (record.status === 'orphan') return record.dst
  return record.sniffed || record.dst
}

function sourceDetail(record: ConnectionRecord): string {
  const details = [record.pname, record.mac].filter(Boolean)
  return details.join(' · ')
}

const columns = computed<DataTableColumns<ConnectionRecord>>(() => [
  {
    title: '状态',
    key: 'status',
    width: 116,
    render: (row) => {
      const meta = statusMeta(row)
      return h(NTag, { size: 'small', type: meta.type, bordered: false }, { default: () => meta.label })
    },
  },
  {
    title: '活跃时间',
    key: 'firstSeen',
    width: 210,
    sorter: true,
    sortOrder: sortOrder.value,
    render: (row) => {
      const lines = [h('div', firstSeenLabel(row))]
      if (isAlive(row)) {
        lines.push(h('div', { class: 'conn-secondary' }, `已持续 ${elapsedLabel(row)}`))
      }
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
    title: '源',
    key: 'src',
    minWidth: 180,
    render: (row) => {
      const lines = [h('div', { class: 'mono' }, row.src)]
      const detail = sourceDetail(row)
      if (detail) lines.push(h('div', { class: 'conn-secondary' }, detail))
      return lines
    },
  },
  {
    title: '目的',
    key: 'dst',
    minWidth: 230,
    ellipsis: { tooltip: true },
    render: (row) => {
      const lines = [h('div', { class: 'mono' }, destinationTitle(row))]
      if (row.dstAddr && row.dstAddr !== destinationTitle(row)) {
        lines.push(h('div', { class: 'conn-secondary mono' }, row.dstAddr))
      }
      return lines
    },
  },
  {
    title: '出站',
    key: 'outbound',
    minWidth: 150,
    render: (row) => {
      if (row.status === 'orphan') return h(NText, { depth: 3 }, { default: () => '未知（日志已滚出）' })
      const lines = [h(NTag, { size: 'small', type: outboundType(row), bordered: false }, { default: () => row.outbound })]
      const detail = [row.dialer, row.policy].filter(Boolean).join(' · ')
      if (detail) lines.push(h('div', { class: 'conn-secondary' }, detail))
      return lines
    },
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
    data.value = await getJSON<ConnectionsResponse>(`/api/v1/connections?limit=${limit.value}`)
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
        <NText depth="3">从 dae 连接日志与 socket 快照对账还原的连接状态，不含流量与速率</NText>
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
      log_level 调整为 info 并重载，连接活动才能被采集；存活的 socket 仍会以"存活 · 无日志"形式列出。
    </NAlert>
    <NAlert v-else-if="data && !data.snapshotOk" type="warning">
      无法读取 dae 的 socket 快照，存活判定暂不可用；下方记录的状态一律显示为"未知"。
    </NAlert>
    <NAlert v-else-if="data?.truncated" type="info">
      连接数量超出单次列出的上限，下方只列出较新的一部分；上方三个计数仍是完整值。
    </NAlert>

    <NGrid responsive="screen" cols="1 s:3" :x-gap="14" :y-gap="14">
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">存活 TCP 连接</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.liveTcp ?? '—' }}</strong>
          <small>dae 当前持有 {{ summary?.tcpSockets ?? '—' }} 个 TCP socket（含出站腿）</small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">窗口内连接事件</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.windowEvents ?? '—' }}</strong>
          <small>
            面板运行期内累积的建立事件
            <template v-if="data?.dropped">· {{ data.dropped }} 行未能解析</template>
          </small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">UDP socket</NText>
          <NSkeleton v-if="loading && !data" text style="width: 60%" />
          <strong v-else class="metric-value">{{ summary?.udpSockets ?? '—' }}</strong>
          <small>UDP 无关闭语义，不参与存活判定</small>
        </NCard>
      </NGridItem>
    </NGrid>

    <NCard content-style="padding: 0;">
      <div class="filter-bar">
        <NRadioGroup v-model:value="view" size="small">
          <NRadioButton value="live">存活中</NRadioButton>
          <NRadioButton value="all">全部活动</NRadioButton>
        </NRadioGroup>
        <NInput v-model:value="search" clearable placeholder="搜索地址、域名、节点或进程" class="log-search">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
        <NSelect v-model:value="outbound" clearable :options="outboundOptions" placeholder="全部出站" class="log-select" />
        <NSelect v-model:value="network" clearable :options="networkOptions" placeholder="全部协议" class="log-select" />
        <NSelect v-model:value="limit" :options="limitOptions" class="log-limit" @update:value="load()" />
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
        :row-key="(row: ConnectionRecord) => `${row.network}|${row.src}|${row.dst}|${row.firstSeen}`"
        :scroll-x="980"
        :bordered="false"
        :pagination="{ pageSize: 50 }"
        @update:sorter="handleSorter"
      >
        <template #empty>
          <NEmpty
            class="empty-state"
            :description="view === 'live' ? '当前没有存活的 TCP 连接' : '窗口内没有连接活动'"
          />
        </template>
      </NDataTable>

      <NSpin v-else :show="loading && !data">
        <div v-if="mobileEntries.length" class="mobile-record-list">
          <article v-for="record in mobileEntries" :key="`${record.network}|${record.src}|${record.dst}|${record.firstSeen}`" class="mobile-record">
            <div class="mobile-record-head">
              <div class="mobile-record-title">
                <NTag size="small" :type="statusMeta(record).type" :bordered="false">{{ statusMeta(record).label }}</NTag>
                <span class="conn-mobile-destination mono">{{ destinationTitle(record) }}</span>
              </div>
              <NTag size="tiny" :bordered="false">{{ record.network }}</NTag>
            </div>
            <p class="mobile-record-description mono">{{ record.src }} → {{ record.dstAddr || record.dst }}</p>
            <div class="mobile-record-meta">
              <span v-if="record.status !== 'orphan'">出站<strong>{{ record.outbound }}</strong></span>
              <span v-if="record.dialer">节点<strong>{{ record.dialer }}</strong></span>
              <span>开始<strong>{{ firstSeenLabel(record) }}</strong></span>
              <span v-if="isAlive(record)">已持续<strong>{{ elapsedLabel(record) }}</strong></span>
              <span v-if="sourceDetail(record)">来源<strong>{{ sourceDetail(record) }}</strong></span>
            </div>
          </article>
          <NText v-if="filteredEntries.length > mobileListCap" depth="3" class="conn-mobile-cap">
            仅显示前 {{ mobileListCap }} 条，可用筛选缩小范围
          </NText>
        </div>
        <NEmpty
          v-else
          class="mobile-empty"
          :description="view === 'live' ? '当前没有存活的 TCP 连接' : '窗口内没有连接活动'"
        />
      </NSpin>
    </NCard>
  </div>
</template>
