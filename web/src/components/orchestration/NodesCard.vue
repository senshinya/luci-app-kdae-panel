<script setup lang="ts">
import { computed, h, ref } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NInput,
  NModal,
  NSpace,
  NTag,
  NText,
  NTooltip,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { CreateOutline, DownloadOutline, FlashOutline, PricetagOutline, TrashOutline } from '@vicons/ionicons5'
import { postJSON } from '../../api/client'
import { useMobileViewport } from '../../composables/useMobileViewport'
import type { LatencyResult, LatencyTarget } from '../../types/api'
import { appendToSection, isQuotable, isValidTag, quote, readSection, removeLine, type Entry } from '../../utils/daeconf'
import { parseNodeLink, type NodeLinkInfo } from '../../utils/nodelink'
import { entryActions, useEntryRewrite, type EntryTarget } from './entry'
import SectionEditorModal from './SectionEditorModal.vue'

interface NodeRow {
  entry: Entry
  info: NodeLinkInfo | null
}

const content = defineModel<string>({ required: true })
const message = useMessage()
const mobile = useMobileViewport()
const { captureEntry, rewriteEntry } = useEntryRewrite(content, message)
const sourceVisible = ref(false)

const nodes = computed<NodeRow[]>(() =>
  readSection(content.value, 'node').entries.map((entry) => ({ entry, info: parseNodeLink(entry.value) })),
)

// ---- 导入 ----
const importVisible = ref(false)
const importText = ref('')

function importNodes() {
  const links = importText.value.split('\n').map((line) => line.trim()).filter((line) => line !== '')
  if (links.length === 0) {
    message.warning('请粘贴至少一个分享链接')
    return
  }
  const invalid = links.filter((link) => !/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(link))
  if (invalid.length > 0) {
    message.error(`有 ${invalid.length} 行不是有效的分享链接: ${invalid[0].slice(0, 40)}…`)
    return
  }
  const unrepresentable = links.filter((link) => !isQuotable(link))
  if (unrepresentable.length > 0) {
    message.error('链接同时包含单引号和双引号，dae 配置无法无损表示，请先修正链接')
    return
  }
  content.value = appendToSection(content.value, 'node', links.map((link) => quote(link)))
  importVisible.value = false
  importText.value = ''
  message.success(`已加入 ${links.length} 个节点，保存并重载后生效`)
}

function removeNode(row: NodeRow) {
  content.value = removeLine(content.value, row.entry.lineStart, row.entry.lineEnd)
}

// ---- 标签 ----
const tagTarget = ref<EntryTarget | null>(null)
const tagValue = ref('')

function openTagEditor(entry: Entry) {
  tagTarget.value = captureEntry(entry)
  tagValue.value = entry.tag || ''
}

function applyTag() {
  const target = tagTarget.value
  if (!target) return
  const tag = tagValue.value.trim()
  if (tag !== '' && !isValidTag(tag)) {
    message.error('标签只能使用字母、数字、下划线、点或横线，且以字母或下划线开头')
    return
  }
  if (!isQuotable(target.entry.value)) {
    message.error('该链接同时包含单引号和双引号，无法安全改写，请使用卡片右上角的原文编辑')
    return
  }
  const line = tag === '' ? quote(target.entry.value) : `${tag}: ${quote(target.entry.value)}`
  if (rewriteEntry(target, line)) tagTarget.value = null
}

// ---- 节点延迟(面板主机 TCP 直连握手,非 dae 内部健康检查) ----
const probing = ref(false)
const latency = ref(new Map<string, LatencyResult>())

function latencyKey(info: NodeLinkInfo): string {
  return `${info.host}:${info.port}`
}

/** 与后端 netprobe.Target.validate 一致，避免把明显非法的目标发出去。 */
function probeTarget(info: NodeLinkInfo | null): LatencyTarget | null {
  const host = info?.host
  const port = info?.port
  if (!host || host !== host.trim() || host.length > 253 || /[\s/\\]/.test(host)) return null
  if (typeof port !== 'number' || !Number.isInteger(port) || port < 1 || port > 65535) return null
  return { host, port }
}

async function probeLatency() {
  const targets = new Map<string, LatencyTarget>()
  for (const row of nodes.value) {
    const target = probeTarget(row.info)
    if (target) targets.set(`${target.host}:${target.port}`, target)
  }
  if (targets.size === 0) {
    message.warning('没有可探测的节点(需要能解析出服务器与端口)')
    return
  }
  probing.value = true
  const batch = [...targets.values()]
  const merged = new Map(latency.value)
  try {
    for (let start = 0; start < batch.length; start += 64) {
      const { results } = await postJSON<{ results: LatencyResult[] }>('/api/v1/net/latency', {
        targets: batch.slice(start, start + 64),
      })
      for (const result of results) merged.set(`${result.host}:${result.port}`, result)
      // 逐批发布，某一批失败时前面的结果仍然可见
      latency.value = new Map(merged)
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '延迟探测失败')
  } finally {
    probing.value = false
  }
}

function latencyCell(row: NodeRow) {
  if (!row.info || !probeTarget(row.info)) return h(NText, { depth: 3 }, { default: () => '—' })
  const result = latency.value.get(latencyKey(row.info))
  if (!result) return h(NText, { depth: 3 }, { default: () => '未测' })
  if (!result.reachable) {
    return h(NTooltip, null, {
      trigger: () => h(NTag, { size: 'small', type: 'error', bordered: false }, { default: () => '不可达' }),
      default: () => result.error || '连接失败',
    })
  }
  const value = result.latencyMs || 0
  const type = value < 100 ? 'success' : value < 300 ? 'warning' : 'error'
  return h(NTag, { size: 'small', type, bordered: false }, { default: () => `${value.toFixed(value < 10 ? 1 : 0)} ms` })
}

function latencyLabel(row: NodeRow): string {
  if (!row.info || !probeTarget(row.info)) return '—'
  const result = latency.value.get(latencyKey(row.info))
  if (!result) return '未测'
  if (!result.reachable) return '不可达'
  const value = result.latencyMs || 0
  return `${value.toFixed(value < 10 ? 1 : 0)} ms`
}

function latencyType(row: NodeRow): 'success' | 'warning' | 'error' | 'default' {
  if (!row.info || !probeTarget(row.info)) return 'default'
  const result = latency.value.get(latencyKey(row.info))
  if (!result) return 'default'
  if (!result.reachable) return 'error'
  const value = result.latencyMs || 0
  return value < 100 ? 'success' : value < 300 ? 'warning' : 'error'
}

const nodeColumns: DataTableColumns<NodeRow> = [
  {
    title: '名称',
    key: 'name',
    minWidth: 160,
    ellipsis: { tooltip: true },
    render: (row) => row.entry.tag || row.info?.name || h(NText, { depth: 3 }, { default: () => '未命名' }),
  },
  {
    title: '标签',
    key: 'tag',
    width: 130,
    ellipsis: { tooltip: true },
    render: (row) => row.entry.tag
      ? h(NTag, { size: 'small', bordered: false }, { default: () => row.entry.tag })
      : h(NText, { depth: 3 }, { default: () => '—' }),
  },
  {
    title: '协议',
    key: 'protocol',
    width: 110,
    render: (row) => h(NTag, { size: 'small', type: 'info', bordered: false }, { default: () => row.info?.protocol || '未知' }),
  },
  {
    title: '服务器',
    key: 'host',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'mono' }, row.info?.host || '—'),
  },
  {
    title: '端口',
    key: 'port',
    width: 90,
    render: (row) => row.info?.port ?? '—',
  },
  {
    title: () => h(NTooltip, null, {
      trigger: () => h('span', { class: 'column-hint' }, '直连延迟'),
      default: () => '面板主机到该服务器的 TCP 握手耗时，域名目标包含解析时间。这不是 dae 的健康检查结果，也不是 dae 选路所用的延迟；dae 开启 wan_interface 时会劫持本机流量，此时该连接同样由 dae 按路由规则转发。',
    }),
    key: 'latency',
    width: 110,
    render: latencyCell,
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    fixed: 'right',
    render: (row) => entryActions(row.entry, [
      { title: '打标签', icon: PricetagOutline, onClick: () => openTagEditor(row.entry) },
      { title: '移除', icon: TrashOutline, type: 'error', onClick: () => removeNode(row) },
    ]),
  },
]
</script>

<template>
  <NCard title="节点" class="panel-card" content-style="padding: 0;" data-testid="nodes-card">
    <template #header-extra>
      <NSpace size="small">
        <NTag size="small" :bordered="false">{{ nodes.length }} 个</NTag>
        <NButton size="small" secondary :loading="probing" :disabled="nodes.length === 0" @click="probeLatency">
          <template #icon><NIcon><FlashOutline /></NIcon></template>测试直连延迟
        </NButton>
        <NButton size="small" type="primary" @click="importVisible = true">
          <template #icon><NIcon><DownloadOutline /></NIcon></template>导入节点
        </NButton>
        <NButton size="small" quaternary @click="sourceVisible = true">
          <template #icon><NIcon><CreateOutline /></NIcon></template>编辑原文
        </NButton>
      </NSpace>
    </template>
    <NDataTable
      v-if="!mobile"
      :columns="nodeColumns"
      :data="nodes"
      :row-key="(row: NodeRow) => row.entry.lineStart"
      :scroll-x="960"
      :bordered="false"
      size="small"
    >
      <template #empty>
        <div class="orchestrate-empty">
          <NText depth="3">还没有手工节点。粘贴分享链接导入，或使用订阅。</NText>
        </div>
      </template>
    </NDataTable>
    <template v-else>
      <div v-if="nodes.length" class="mobile-record-list" data-testid="mobile-node-list">
        <article v-for="row in nodes" :key="row.entry.lineStart" class="mobile-record">
          <div class="mobile-record-head">
            <div class="mobile-record-title">
              <span>{{ row.entry.tag || row.info?.name || '未命名' }}</span>
              <NTag size="tiny" type="info" :bordered="false">{{ row.info?.protocol || '未知' }}</NTag>
            </div>
            <NTag size="small" :type="latencyType(row)" :bordered="false">{{ latencyLabel(row) }}</NTag>
          </div>
          <p class="mobile-record-description mono">
            {{ row.info?.host || '无法解析服务器' }}<template v-if="row.info?.port">:{{ row.info.port }}</template>
          </p>
          <div class="mobile-record-meta">
            <span v-if="row.entry.tag">标签<strong class="mono">{{ row.entry.tag }}</strong></span>
            <span>配置行<strong>{{ row.entry.lineStart + 1 }}</strong></span>
          </div>
          <div class="mobile-action-row">
            <NButton secondary :disabled="!row.entry.editable" @click="openTagEditor(row.entry)">
              <template #icon><NIcon><PricetagOutline /></NIcon></template>标签
            </NButton>
            <NButton secondary type="error" :disabled="!row.entry.editable" @click="removeNode(row)">
              <template #icon><NIcon><TrashOutline /></NIcon></template>移除
            </NButton>
          </div>
        </article>
      </div>
      <NEmpty v-else description="还没有手工节点。粘贴分享链接导入，或使用订阅。" class="mobile-empty" />
    </template>
  </NCard>

  <NModal v-model:show="importVisible" preset="card" title="导入节点" class="orchestrate-modal">
    <NText depth="3">
      每行一个分享链接，支持 vmess / vless / ss / ssr / trojan / tuic / juicity / hysteria2 / anytls / socks5 / http(s)。
      节点名称取自链接自身，导入后可单独打标签。
    </NText>
    <NInput
      v-model:value="importText"
      type="textarea"
      class="mono"
      :rows="8"
      placeholder="vmess://…&#10;vless://…&#10;hysteria2://…"
      spellcheck="false"
    />
    <template #footer>
      <NSpace justify="end">
        <NButton @click="importVisible = false">取消</NButton>
        <NButton type="primary" @click="importNodes">加入编排</NButton>
      </NSpace>
    </template>
  </NModal>

  <NModal :show="tagTarget !== null" preset="card" title="节点标签" class="orchestrate-modal" @update:show="tagTarget = null">
    <NText depth="3">标签用于在分组过滤中稳定引用节点，留空则恢复为匿名条目。</NText>
    <NInput v-model:value="tagValue" placeholder="如 hk_01" spellcheck="false" @keyup.enter="applyTag" />
    <template #footer>
      <NSpace justify="end">
        <NButton @click="tagTarget = null">取消</NButton>
        <NButton type="primary" @click="applyTag">确定</NButton>
      </NSpace>
    </template>
  </NModal>

  <SectionEditorModal
    v-model:show="sourceVisible"
    v-model:content="content"
    section="node"
    title="编辑节点原文"
    description="这里只替换 node 节内部内容，适合处理跨行节点或无法用表格安全改写的链接。其他配置保持不变。"
  />
</template>
