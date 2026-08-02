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
  NRadioButton,
  NRadioGroup,
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
import type { ConnectionEvent, ConnectionFacet, ConnectionFacets, ConnectionsResponse } from '../types/api'
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
const endpointDrawerOpen = ref(false)
const facetDrawerOpen = ref(false)
const facetDimension = ref<keyof ConnectionFacets>('targets')
const selectedFacet = ref<{ dimension: keyof ConnectionFacets, id: string } | null>(null)
const sortOrder = ref<'descend' | 'ascend'>('descend')
const now = ref(Date.now())
let refreshTimer: number | undefined
let clockTimer: number | undefined
let reloadQueued = false

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
const desktopFacetCap = 6
const facetOptions: Array<{ label: string, value: keyof ConnectionFacets }> = [
  { label: '目标', value: 'targets' },
  { label: '客户端', value: 'clients' },
  { label: '节点', value: 'nodes' },
  { label: '出站组', value: 'groups' },
]

const entries = computed(() => data.value?.entries ?? [])
const summary = computed(() => data.value?.summary)
const snapshotLabel = computed(() => data.value?.snapshotAt ? formatDateTime(data.value.snapshotAt) : '等待快照')
const activeFilterCount = computed(() => Number(!!outbound.value) + Number(!!network.value) + Number(!!selectedFacet.value))
const facetItems = computed(() => data.value?.facets[facetDimension.value] ?? [])
const visibleFacets = computed(() => facetItems.value.slice(0, desktopFacetCap))
const facetMaximum = computed(() => facetItems.value[0]?.count ?? 1)
const facetDimensionLabel = computed(() => facetOptions.find((option) => option.value === facetDimension.value)?.label ?? '分布')
const activeFacet = computed(() => {
  const selected = selectedFacet.value
  if (!selected || selected.dimension !== facetDimension.value) return null
  return facetItems.value.find((item) => item.id === selected.id) ?? null
})
const outboundOptions = computed(() => {
  return [
    { label: '全部出站', value: '' },
    ...(data.value?.facets.groups ?? []).map((item) => ({ label: item.label, value: item.id })),
  ]
})

const filteredEntries = computed(() => {
  const query = search.value.trim().toLowerCase()
  const cutoff = now.value - windowMinutes.value * 60_000
  const filtered = entries.value.filter((entry) => {
    if (Date.parse(entry.at) < cutoff) return false
    if (network.value && !entry.network.startsWith(network.value)) return false
    if (outbound.value && entry.outbound !== outbound.value) return false
    if (selectedFacet.value && !matchesFacet(entry, selectedFacet.value)) return false
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
const endpointMaximum = computed(() => data.value?.endpoints[0]?.count ?? 1)

function destinationTitle(event: ConnectionEvent): string {
  return event.sniffed || event.dst
}

function sourceDetail(event: ConnectionEvent): string {
  return [event.pname, event.mac].filter(Boolean).join(' · ')
}

function connectionHost(value: string): string {
  const trimmed = value.trim()
  const bracketed = trimmed.match(/^\[([^\]]+)](?::\d+)?$/)
  if (bracketed) return bracketed[1]
  const firstColon = trimmed.indexOf(':')
  const lastColon = trimmed.lastIndexOf(':')
  if (firstColon > 0 && firstColon === lastColon && /^\d+$/.test(trimmed.slice(lastColon + 1))) {
    return trimmed.slice(0, lastColon)
  }
  return trimmed
}

function targetFacetID(event: ConnectionEvent): string {
  return connectionHost(event.sniffed || event.dst).toLowerCase().replace(/\.$/, '')
}

function clientFacetID(event: ConnectionEvent): string {
  const mac = normalizedClientMAC(event.mac)
  if (mac) return `mac:${mac}`
  return `ip:${connectionHost(event.src).toLowerCase()}`
}

function normalizedClientMAC(value?: string): string {
  const normalized = value?.trim().toLowerCase().replaceAll('-', ':') ?? ''
  const octets = normalized.split(':')
  if (![6, 8].includes(octets.length) || octets.some((octet) => !/^[0-9a-f]{2}$/.test(octet))) return ''
  if ((Number.parseInt(octets[0], 16) & 1) !== 0 || octets.every((octet) => octet === '00')) return ''
  return normalized
}

function matchesFacet(event: ConnectionEvent, selected: { dimension: keyof ConnectionFacets, id: string }): boolean {
  switch (selected.dimension) {
  case 'targets': return targetFacetID(event) === selected.id
  case 'clients': return clientFacetID(event) === selected.id
  case 'nodes': return event.dialer === selected.id
  case 'groups': return event.outbound === selected.id
  }
  return false
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
  selectedFacet.value = null
}

function selectFacet(item: ConnectionFacet) {
  const same = selectedFacet.value?.dimension === facetDimension.value && selectedFacet.value.id === item.id
  selectedFacet.value = same ? null : { dimension: facetDimension.value, id: item.id }
  if (mobile.value) facetDrawerOpen.value = false
}

function changeFacetDimension(value: keyof ConnectionFacets) {
  facetDimension.value = value
  selectedFacet.value = null
}

function handleWindowChange(value: number) {
  windowMinutes.value = value
  selectedFacet.value = null
  void load()
}

function handleLimitChange(value: number) {
  limit.value = value
  void load()
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
  if (refreshing.value) {
    if (!silent) reloadQueued = true
    return
  }
  refreshing.value = true
  if (!silent && !data.value) loading.value = true
  try {
    data.value = await getJSON<ConnectionsResponse>(`/api/v1/connections?limit=${limit.value}&window=${windowMinutes.value}`)
    errorMessage.value = ''
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取连接活动失败'
    if (!silent) message.error(errorMessage.value)
  } finally {
    refreshing.value = false
    loading.value = false
    if (reloadQueued) {
      reloadQueued = false
      void load(true)
    }
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
    <NAlert v-if="data?.truncated" type="info">部分连接明细或端点已截断；页面计数基于面板当前保留或扫描到的数据。</NAlert>
    <NAlert v-if="data?.facetLimited" type="info">分布维度过多，每个维度仅返回连接数最高的 200 项。</NAlert>
    <NAlert v-if="data?.dropped" type="warning">有 {{ data.dropped }} 条疑似连接日志无法解析，dae 的日志格式可能已经变化。</NAlert>

    <section class="connection-pulse" aria-label="连接摘要">
      <div class="connection-pulse-primary">
        <span class="connection-live-beacon" :class="{ muted: !data?.snapshotOk }"></span>
        <strong>{{ data?.snapshotOk ? summary?.outboundTcp : '—' }}</strong>
        <span>条 dae TCP 出站</span>
      </div>
      <dl class="connection-pulse-metrics">
        <div><dt>UDP socket</dt><dd>{{ data?.snapshotOk ? summary?.udpSockets : '—' }}</dd></div>
        <div><dt>新建连接</dt><dd>{{ summary?.windowEvents ?? '—' }}</dd></div>
        <div><dt>客户端</dt><dd>{{ summary?.windowClients ?? '—' }}</dd></div>
        <div><dt>目标</dt><dd>{{ summary?.windowTargets ?? '—' }}</dd></div>
      </dl>
      <NButton text class="connection-endpoint-trigger" :disabled="!data?.snapshotOk" @click="endpointDrawerOpen = true">
        实时端点 <strong>{{ data?.snapshotOk ? data.endpoints.length : '—' }}</strong>
      </NButton>
    </section>

    <section class="connection-workbench">
      <div class="connection-toolbar">
        <div class="connection-commandbar">
          <NSelect :value="windowMinutes" :options="windowOptions" class="connections-window" @update:value="handleWindowChange" />
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
            <NSelect :value="limit" :options="limitOptions" @update:value="handleLimitChange" />
            <NButton v-if="activeFilterCount" quaternary @click="clearFilters">清除筛选</NButton>
          </div>
        </NCollapseTransition>
      </div>

      <section class="connection-facets" aria-label="活动分布">
        <header>
          <div>
            <strong>活动分布</strong>
            <small>所选时段内的新建连接</small>
          </div>
          <NRadioGroup
            v-if="!mobile"
            :value="facetDimension"
            size="small"
            class="connection-facet-modes"
            @update:value="changeFacetDimension"
          >
            <NRadioButton v-for="option in facetOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </NRadioButton>
          </NRadioGroup>
        </header>

        <div v-if="!mobile" class="connection-facet-desktop">
          <div v-if="visibleFacets.length" class="connection-facet-grid">
            <button
              v-for="(item, index) in visibleFacets"
              :key="item.id"
              type="button"
              class="connection-facet-row"
              :class="{ active: activeFacet?.id === item.id }"
              :aria-pressed="activeFacet?.id === item.id"
              @click="selectFacet(item)"
            >
              <span class="connection-facet-rank">{{ index + 1 }}</span>
              <span class="connection-facet-copy">
                <strong :class="{ mono: facetDimension === 'clients' }">{{ item.label }}</strong>
                <small v-if="item.note" class="mono">{{ item.note }}</small>
              </span>
              <span class="connection-facet-track" aria-hidden="true">
                <i :style="{ width: `${Math.max(4, item.count / facetMaximum * 100)}%` }"></i>
              </span>
              <strong class="connection-facet-count">{{ item.count }}</strong>
            </button>
          </div>
          <NText v-else depth="3" class="connection-facet-empty">所选时段内没有可统计的{{ facetDimensionLabel }}</NText>
          <NText v-if="facetItems.length > desktopFacetCap" depth="3" class="connection-facet-overflow">
            展示前 {{ desktopFacetCap }} 项，共 {{ facetItems.length }} 项
          </NText>
        </div>

        <button v-else type="button" class="connection-facet-mobile" @click="facetDrawerOpen = true">
          <span>
            <strong>{{ facetDimensionLabel }}分布</strong>
            <small v-if="activeFacet">已筛选 · {{ activeFacet.label }}</small>
            <small v-else-if="facetItems[0]">{{ facetItems[0].label }} · {{ facetItems[0].count }} 次</small>
            <small v-else>暂无数据</small>
          </span>
          <span>查看 {{ facetItems.length }}</span>
        </button>
      </section>

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
      </div>
    </section>

    <NDrawer
      v-model:show="facetDrawerOpen"
      placement="bottom"
      height="78vh"
    >
      <NDrawerContent title="活动分布" closable :native-scrollbar="false">
        <NRadioGroup
          :value="facetDimension"
          size="small"
          class="connection-facet-drawer-modes"
          @update:value="changeFacetDimension"
        >
          <NRadioButton v-for="option in facetOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </NRadioButton>
        </NRadioGroup>
        <NText depth="3" class="connection-drawer-description">选择一项后会筛选下方连接流水</NText>
        <div v-if="facetItems.length" class="connection-facet-drawer-list">
          <button
            v-for="(item, index) in facetItems"
            :key="item.id"
            type="button"
            class="connection-facet-row"
            :class="{ active: activeFacet?.id === item.id }"
            :aria-pressed="activeFacet?.id === item.id"
            @click="selectFacet(item)"
          >
            <span class="connection-facet-rank">{{ index + 1 }}</span>
            <span class="connection-facet-copy">
              <strong :class="{ mono: facetDimension === 'clients' }">{{ item.label }}</strong>
              <small v-if="item.note" class="mono">{{ item.note }}</small>
            </span>
            <span class="connection-facet-track" aria-hidden="true">
              <i :style="{ width: `${Math.max(3, item.count / facetMaximum * 100)}%` }"></i>
            </span>
            <strong class="connection-facet-count">{{ item.count }}</strong>
          </button>
        </div>
        <NEmpty v-else description="所选时段内没有可统计的连接" />
      </NDrawerContent>
    </NDrawer>

    <NDrawer
      v-model:show="endpointDrawerOpen"
      :placement="mobile ? 'bottom' : 'right'"
      :width="mobile ? undefined : 420"
      :height="mobile ? '72vh' : undefined"
    >
      <NDrawerContent title="dae 出站远端" closable :native-scrollbar="false">
        <NText depth="3" class="connection-drawer-description">当前 TCP socket 按远端 IP:端口聚合</NText>
        <div v-if="data?.endpoints.length" class="connection-endpoint-drawer-list">
          <div v-for="endpoint in data?.endpoints ?? []" :key="endpoint.address" class="connection-endpoint-row">
            <div><span class="mono">{{ endpoint.address }}</span><strong>{{ endpoint.count }}</strong></div>
            <span class="connection-endpoint-track"><i :style="{ width: `${Math.max(2, endpoint.count / endpointMaximum * 100)}%` }"></i></span>
          </div>
        </div>
        <NEmpty v-else description="当前没有可见的 dae TCP 出站" />
      </NDrawerContent>
    </NDrawer>
  </div>
</template>
