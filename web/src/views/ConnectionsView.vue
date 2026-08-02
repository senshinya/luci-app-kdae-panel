<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCollapseTransition,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NEmpty,
  NIcon,
  NInput,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NText,
  useMessage,
  type DataTableColumns,
  type DataTableSortState,
} from 'naive-ui'
import {
  ArrowDownOutline,
  ArrowForwardOutline,
  ArrowUpOutline,
  FunnelOutline,
  RefreshOutline,
  SearchOutline,
} from '@vicons/ionicons5'
import { getJSON } from '../api/client'
import { useMobileViewport } from '../composables/useMobileViewport'
import type { ConnectionEvent, ConnectionsResponse } from '../types/api'
import { formatDateTime, formatElapsedSince } from '../utils/format'

const message = useMessage()
const mobile = useMobileViewport()
const data = ref<ConnectionsResponse | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const autoRefresh = ref(true)
const errorMessage = ref('')
const search = ref('')
const outbound = ref<string | null>(null)
const network = ref<string | null>(null)
const limit = ref(500)
const windowMinutes = ref(15)
const filtersOpen = ref(false)
const endpointsOpen = ref(false)
const endpointDrawerOpen = ref(false)
const sortOrder = ref<'descend' | 'ascend'>('descend')
const now = ref(Date.now())
let refreshTimer: number | undefined
let clockTimer: number | undefined

const limitOptions = [100, 200, 500, 1000, 2000].map((value) => ({ label: `${value} 条`, value }))
const windowOptions = [
  { label: '最近 5 分钟', value: 5 },
  { label: '最近 15 分钟', value: 15 },
  { label: '最近 1 小时', value: 60 },
  { label: '最近 6 小时', value: 360 },
  { label: '最近 24 小时', value: 1440 },
]
const networkOptions = [
  { label: '全部协议', value: '' },
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
]
const mobileListCap = 100

const entries = computed(() => data.value?.entries ?? [])
const summary = computed(() => data.value?.summary)
const snapshotLabel = computed(() => data.value?.snapshotAt ? formatDateTime(data.value.snapshotAt) : '等待快照')
const activeFilterCount = computed(() => Number(!!outbound.value) + Number(!!network.value))
const outboundOptions = computed(() => {
  const names = new Set(entries.value.map((entry) => entry.outbound).filter(Boolean))
  return [
    { label: '全部出站', value: '' },
    ...[...names].sort().map((name) => ({ label: name, value: name })),
  ]
})

const filteredEntries = computed(() => {
  const query = search.value.trim().toLowerCase()
  const cutoff = now.value - windowMinutes.value * 60_000
  const filtered = entries.value.filter((entry) => {
    if (Date.parse(entry.at) < cutoff) return false
    if (network.value && !entry.network.startsWith(network.value)) return false
    if (outbound.value && entry.outbound !== outbound.value) return false
    if (!query) return true
    return [entry.src, entry.dst, entry.dstAddr, entry.sniffed, entry.dialer, entry.outbound, entry.policy, entry.pname, entry.mac]
      .some((value) => value?.toLowerCase().includes(query))
  })
  const direction = sortOrder.value === 'descend' ? -1 : 1
  return filtered.sort((left, right) => {
    const delta = Date.parse(left.at) - Date.parse(right.at)
    return delta === 0 ? left.src.localeCompare(right.src) : delta * direction
  })
})

const mobileEntries = computed(() => filteredEntries.value.slice(0, mobileListCap))
const visibleEndpoints = computed(() => data.value?.endpoints.slice(0, 3) ?? [])
const endpointMaximum = computed(() => visibleEndpoints.value[0]?.count ?? 1)
const hiddenEndpointCount = computed(() => Math.max(0, (data.value?.endpoints.length ?? 0) - visibleEndpoints.value.length))

function destinationTitle(event: ConnectionEvent): string {
  return event.sniffed || event.dst
}

function sourceDetail(event: ConnectionEvent): string {
  return [event.pname, event.mac].filter(Boolean).join(' · ')
}

function routeTitle(event: ConnectionEvent): string {
  return [event.outbound || '未知出站', event.dialer].filter(Boolean).join(' / ')
}

function exactTimeLabel(event: ConnectionEvent): string {
  return `${event.approxTime ? '约 ' : ''}${formatDateTime(event.at)}`
}

function agoLabel(event: ConnectionEvent): string {
  return `${formatElapsedSince(event.at, now.value)}前`
}

function clearFilters() {
  outbound.value = null
  network.value = null
}

const columns = computed<DataTableColumns<ConnectionEvent>>(() => [
  {
    title: () => h('div', { class: 'connection-flow-header' }, [
      h('span', '来源'),
      h('span', '目标'),
    ]),
    key: 'flow',
    minWidth: 520,
    render: (row) => h('div', { class: 'connection-flow-cell' }, [
      h('div', { class: 'connection-primary-cell' }, [
        h('strong', { class: 'mono' }, row.src),
        h('small', [row.network, sourceDetail(row)].filter(Boolean).join(' · ')),
      ]),
      h(NIcon, { class: 'connection-flow-arrow', size: 14, 'aria-hidden': 'true' }, {
        default: () => h(ArrowForwardOutline),
      }),
      h('div', { class: 'connection-primary-cell' }, [
        h('strong', { class: 'mono' }, destinationTitle(row)),
        ...(row.dstAddr && row.dstAddr !== destinationTitle(row)
          ? [h('small', { class: 'mono' }, row.dstAddr)]
          : []),
      ]),
    ]),
  },
  {
    title: '路由', key: 'outbound', minWidth: 220,
    render: (row) => h('div', { class: ['connection-route', `is-${row.outbound || 'unknown'}`] }, [
      h('strong', routeTitle(row)),
      ...(row.policy ? [h('small', row.policy)] : []),
    ]),
  },
  {
    title: '建立时间', key: 'at', width: 176, sorter: true, sortOrder: sortOrder.value,
    render: (row) => h('div', { class: 'connection-time-cell' }, [
      h('strong', agoLabel(row)),
      h('small', exactTimeLabel(row)),
    ]),
  },
])

function handleSorter(sorter: DataTableSortState | DataTableSortState[] | null) {
  const state = Array.isArray(sorter) ? sorter[0] : sorter
  sortOrder.value = state?.order === 'ascend' ? 'ascend' : 'descend'
}

function toggleSort() {
  sortOrder.value = sortOrder.value === 'descend' ? 'ascend' : 'descend'
}

async function load(silent = false) {
  if (refreshing.value) return
  refreshing.value = true
  if (!silent && !data.value) loading.value = true
  try {
    data.value = await getJSON<ConnectionsResponse>(`/api/v1/connections?limit=${limit.value}`)
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取连接活动失败'
    if (!silent) message.error(errorMessage.value)
  } finally {
    refreshing.value = false
    loading.value = false
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible' && autoRefresh.value) void load(true)
}

onMounted(() => {
  void load()
  refreshTimer = window.setInterval(() => {
    if (autoRefresh.value && document.visibilityState === 'visible') void load(true)
  }, 5000)
  clockTimer = window.setInterval(() => { now.value = Date.now() }, 1000)
  document.addEventListener('visibilitychange', handleVisibilityChange)
})
onBeforeUnmount(() => {
  window.clearInterval(refreshTimer)
  window.clearInterval(clockTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<template>
  <div class="page-stack connections-page">
    <div class="page-toolbar">
      <div>
        <h2>连接活动</h2>
        <NText depth="3">最近快照 {{ snapshotLabel }}</NText>
      </div>
      <NSpace align="center" class="connection-refresh-controls">
        <span class="connection-auto-refresh"><NSwitch v-model:value="autoRefresh" size="small" />自动刷新</span>
        <NButton quaternary circle size="small" :loading="refreshing" title="立即刷新" aria-label="立即刷新" @click="load()">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
        </NButton>
      </NSpace>
    </div>

    <NAlert v-if="errorMessage" type="error" closable @close="errorMessage = ''">{{ errorMessage }}</NAlert>
    <NAlert v-if="data && !data.logsOk" type="warning">暂时无法读取 dae 日志，连接流水保留已经收集到的记录。</NAlert>
    <NAlert v-if="data && !data.snapshotOk" type="warning">暂时无法读取 dae 的 socket 快照，实时出站端点不可用。</NAlert>
    <NAlert v-if="data?.truncated" type="info">记录或端点超过展示上限，摘要计数仍来自完整扫描。</NAlert>
    <NAlert v-if="data?.dropped" type="warning">有 {{ data.dropped }} 条疑似连接日志无法解析，dae 的日志格式可能已经变化。</NAlert>

    <section class="connection-pulse" aria-label="连接摘要">
      <div class="connection-pulse-primary">
        <span class="connection-live-beacon" :class="{ muted: !data?.snapshotOk }"></span>
        <strong>{{ data?.snapshotOk ? summary?.outboundTcp : '—' }}</strong>
        <span>条 dae TCP 出站</span>
      </div>
      <dl class="connection-pulse-metrics">
        <div><dt>远端</dt><dd>{{ data?.snapshotOk ? data.endpoints.length : '—' }}</dd></div>
        <div><dt>UDP socket</dt><dd>{{ data?.snapshotOk ? summary?.udpSockets : '—' }}</dd></div>
        <div><dt>所选时段</dt><dd>{{ filteredEntries.length }}</dd></div>
      </dl>
    </section>

    <section class="connection-workbench">
      <div class="connection-toolbar">
        <div class="connection-commandbar">
          <NSelect v-model:value="windowMinutes" :options="windowOptions" class="connections-window" />
          <NInput v-model:value="search" clearable placeholder="搜索地址、域名、节点或进程" class="connections-search">
            <template #prefix><NIcon><SearchOutline /></NIcon></template>
          </NInput>
          <NButton secondary :type="activeFilterCount ? 'primary' : 'default'" @click="filtersOpen = !filtersOpen">
            <template #icon><NIcon><FunnelOutline /></NIcon></template>
            筛选{{ activeFilterCount ? ` ${activeFilterCount}` : '' }}
          </NButton>
          <NButton v-if="mobile" size="small" quaternary class="connections-sort" @click="toggleSort">
            <template #icon>
              <NIcon><ArrowDownOutline v-if="sortOrder === 'descend'" /><ArrowUpOutline v-else /></NIcon>
            </template>
            {{ sortOrder === 'descend' ? '最新在前' : '最早在前' }}
          </NButton>
          <NText depth="3" class="connection-result-count">{{ filteredEntries.length }} / {{ entries.length }}</NText>
        </div>
        <NCollapseTransition :show="filtersOpen">
          <div class="connection-filter-panel">
            <NSelect v-model:value="outbound" clearable :options="outboundOptions" placeholder="全部出站" />
            <NSelect v-model:value="network" clearable :options="networkOptions" placeholder="全部协议" />
            <NSelect v-model:value="limit" :options="limitOptions" @update:value="load()" />
            <NButton v-if="activeFilterCount" quaternary @click="clearFilters">清除筛选</NButton>
          </div>
        </NCollapseTransition>
      </div>

      <div class="connection-workspace">
        <div class="connection-stream">
          <NDataTable
            v-if="!mobile"
            class="connections-table"
            :columns="columns"
            :data="filteredEntries"
            :loading="loading"
            :row-key="(row: ConnectionEvent) => `${row.network}|${row.src}|${row.dst}|${row.at}`"
            :scroll-x="916"
            :bordered="false"
            :pagination="{ pageSize: 50 }"
            @update:sorter="handleSorter"
          >
            <template #empty><NEmpty class="empty-state" description="所选时段内没有连接记录" /></template>
          </NDataTable>

          <NSpin v-else :show="loading">
            <div v-if="mobileEntries.length" class="connection-mobile-list">
              <article
                v-for="event in mobileEntries"
                :key="`${event.network}|${event.src}|${event.dst}|${event.at}`"
                class="connection-mobile-row"
                :class="`is-${event.outbound || 'unknown'}`"
              >
                <header>
                  <div class="connection-mobile-target">
                    <strong class="mono">{{ destinationTitle(event) }}</strong>
                    <small v-if="event.dstAddr && event.dstAddr !== destinationTitle(event)" class="mono">{{ event.dstAddr }}</small>
                  </div>
                  <time>{{ agoLabel(event) }}</time>
                </header>
                <div class="connection-mobile-route">
                  <span class="mono">{{ event.src }}</span>
                  <span aria-hidden="true">→</span>
                  <strong>{{ routeTitle(event) }}</strong>
                </div>
                <footer>
                  <span class="connection-protocol mono">{{ event.network }}</span>
                  <span v-if="event.policy">{{ event.policy }}</span>
                  <span v-if="sourceDetail(event)">{{ sourceDetail(event) }}</span>
                  <time>{{ exactTimeLabel(event) }}</time>
                </footer>
              </article>
              <NText v-if="filteredEntries.length > mobileListCap" depth="3" class="conn-mobile-cap">
                移动端仅显示前 {{ mobileListCap }} 条，可用筛选缩小范围
              </NText>
            </div>
            <NEmpty v-else class="mobile-empty" description="所选时段内没有连接记录" />
          </NSpin>
        </div>

        <aside class="connection-endpoints" aria-label="dae 出站远端">
          <header>
            <div><strong>dae 出站远端</strong><small>当前 TCP socket 分布</small></div>
            <NButton v-if="mobile" text @click="endpointsOpen = !endpointsOpen">{{ endpointsOpen ? '收起' : '展开' }}</NButton>
          </header>
          <NCollapseTransition :show="!mobile || endpointsOpen">
            <div v-if="visibleEndpoints.length" class="connection-endpoint-list">
              <div v-for="endpoint in visibleEndpoints" :key="endpoint.address" class="connection-endpoint-row">
                <div><span class="mono">{{ endpoint.address }}</span><strong>{{ endpoint.count }}</strong></div>
                <span class="connection-endpoint-track"><i :style="{ width: `${Math.max(5, endpoint.count / endpointMaximum * 100)}%` }"></i></span>
              </div>
              <NButton v-if="mobile && hiddenEndpointCount" text class="connection-endpoint-more" @click="endpointDrawerOpen = true">
                查看全部 {{ data?.endpoints.length }} 个远端
              </NButton>
            </div>
            <NText v-else depth="3" class="connection-endpoint-empty">当前没有可见的 dae TCP 出站</NText>
          </NCollapseTransition>
          <NButton
            v-if="!mobile && hiddenEndpointCount"
            text
            class="connection-endpoint-all"
            @click="endpointDrawerOpen = true"
          >
            查看全部 {{ data?.endpoints.length }}
          </NButton>
        </aside>
      </div>
    </section>

    <NDrawer
      v-model:show="endpointDrawerOpen"
      :placement="mobile ? 'bottom' : 'right'"
      :width="mobile ? undefined : 420"
      :height="mobile ? '72vh' : undefined"
    >
      <NDrawerContent title="dae 出站远端" closable :native-scrollbar="false">
        <NText depth="3" class="connection-drawer-description">当前 TCP socket 按远端 IP:端口聚合</NText>
        <div class="connection-endpoint-drawer-list">
          <div v-for="endpoint in data?.endpoints ?? []" :key="endpoint.address" class="connection-endpoint-row">
            <div><span class="mono">{{ endpoint.address }}</span><strong>{{ endpoint.count }}</strong></div>
            <span class="connection-endpoint-track"><i :style="{ width: `${Math.max(2, endpoint.count / endpointMaximum * 100)}%` }"></i></span>
          </div>
        </div>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>
