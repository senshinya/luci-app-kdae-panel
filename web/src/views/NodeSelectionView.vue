<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import {
  NAlert,
  NButton,
  NCard,
  NEmpty,
  NIcon,
  NInputNumber,
  NSpin,
  NTag,
  NText,
  useMessage,
} from 'naive-ui'
import { CheckmarkCircleOutline, FlashOutline, RefreshOutline } from '@vicons/ionicons5'
import { APIError, getJSON, putJSON } from '../api/client'
import type { ConfigDocument, ConfigSaveResult, SubscriptionNodeSource } from '../types/api'
import { parseGroups, type Group } from '../utils/daeconf'
import { parseFixedIndex, resolveFixedCandidates, type FixedCandidate, type FixedCandidates } from '../utils/fixedgroup'
import { useLatencyProbe } from '../composables/useLatencyProbe'

interface FixedGroupView {
  group: Group
  /** 当前 policy 里 fixed(n) 的 n，只用于展示与越界判断，不作为可编辑的本地状态。 */
  index: number
  candidates: FixedCandidates
}

const message = useMessage()
const { probing: latencyProbing, probe, label: latencyLabelOf, type: latencyTypeOf, title: latencyTitleOf } = useLatencyProbe(message)

const loading = ref(true)
const loadError = ref('')
const document = ref<ConfigDocument | null>(null)
const sources = ref<SubscriptionNodeSource[]>([])
const sourcesLoaded = ref(false)

// 当前正在提交切换的分组名；非空时整页禁用交互。绝不维护候选节点的本地可变选中态——
// 界面始终从 document/content 派生，请求失败时天然回到原值，不需要手动回滚。
const switching = ref<string | null>(null)
const busy = computed(() => switching.value !== null)
// 当前正在测延迟的分组名，只用于该分组按钮的 loading 态展示。
const probingGroup = ref<string | null>(null)

const content = computed(() => document.value?.content ?? '')

const fixedGroups = computed<FixedGroupView[]>(() => {
  const text = content.value
  const result: FixedGroupView[] = []
  for (const group of parseGroups(text)) {
    const index = parseFixedIndex(group.policy?.value)
    if (index === null) continue
    result.push({
      group,
      index,
      candidates: resolveFixedCandidates(text, group, sources.value, sourcesLoaded.value),
    })
  }
  return result
})

/** 协议·主机描述；两者都取不到时省略，不展示占位符。 */
function describeCandidate(node: FixedCandidate): string {
  return [node.protocol, node.host].filter(Boolean).join(' · ')
}

/** 候选可解时给出当前节点名；不可解或索引越界时返回 null，绝不展示可能错误的名字。 */
function currentNodeName(entry: FixedGroupView): string | null {
  if (!entry.candidates.resolvable) return null
  return entry.candidates.nodes[entry.index]?.name ?? null
}

function describeTarget(entry: FixedGroupView, targetIndex: number): string {
  if (entry.candidates.resolvable) return entry.candidates.nodes[targetIndex]?.name || `fixed(${targetIndex})`
  return `fixed(${targetIndex})`
}

async function refresh() {
  loading.value = true
  const [configResult, sourcesResult] = await Promise.allSettled([
    getJSON<ConfigDocument>('/api/v1/config'),
    getJSON<{ sources: SubscriptionNodeSource[] }>('/api/v1/subscriptions/nodes'),
  ])
  if (configResult.status === 'fulfilled') {
    document.value = configResult.value
    loadError.value = ''
  } else if (configResult.reason instanceof APIError && configResult.reason.status === 404) {
    // 配置尚不存在等价于没有任何分组，走空状态而不是报错
    document.value = null
    loadError.value = ''
  } else {
    loadError.value = configResult.reason instanceof Error ? configResult.reason.message : '读取配置失败'
  }
  sources.value = sourcesResult.status === 'fulfilled' ? sourcesResult.value.sources : []
  sourcesLoaded.value = true
  loading.value = false
}

/** 切换成功或遇到 409 后必须重新读取，否则 expectedHash 立刻过期，下一次切换必然冲突。 */
async function reloadConfig() {
  try {
    document.value = await getJSON<ConfigDocument>('/api/v1/config')
    loadError.value = ''
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : '重新读取配置失败'
  }
}

async function switchTo(entry: FixedGroupView, targetIndex: number) {
  if (busy.value) return
  if (targetIndex === entry.index) return // 点选当前已选中的节点：无操作，不写配置也不重载
  switching.value = entry.group.name
  try {
    const result = await putJSON<ConfigSaveResult>(
      `/api/v1/groups/${encodeURIComponent(entry.group.name)}/policy`,
      { policy: `fixed(${targetIndex})`, expectedHash: document.value?.hash || '' },
    )
    const label = describeTarget(entry, targetIndex)
    message.success(result.deferred
      ? `已把 ${entry.group.name} 切到 ${label}；dae 未运行，下次启动生效`
      : `已把 ${entry.group.name} 切到 ${label} 并完成无损重载`)
    await reloadConfig()
  } catch (error) {
    if (error instanceof APIError && error.status === 409) {
      message.warning('配置已在别处发生变化，已自动重新读取')
      await reloadConfig()
    } else {
      message.error(error instanceof Error ? error.message : `切换 ${entry.group.name} 的节点失败`)
    }
  } finally {
    switching.value = null
  }
}

async function probeGroup(entry: FixedGroupView) {
  if (!entry.candidates.resolvable || busy.value || latencyProbing.value) return
  probingGroup.value = entry.group.name
  try {
    await probe(entry.candidates.nodes)
  } finally {
    probingGroup.value = null
  }
}

onMounted(() => void refresh())
</script>

<template>
  <div class="page-stack">
    <div class="page-toolbar">
      <div>
        <h2>节点选择</h2>
        <NText depth="3">点选卡片中的节点瓦片，即可把对应的 fixed(n) 分组切到该节点并对 dae 执行无损重载</NText>
      </div>
      <NButton secondary :loading="loading" :disabled="busy" @click="refresh">
        <template #icon><NIcon><RefreshOutline /></NIcon></template>重新读取
      </NButton>
    </div>

    <NAlert v-if="loadError" type="error" closable @close="loadError = ''">{{ loadError }}</NAlert>
    <NAlert v-if="busy" type="info" :bordered="false">
      正在把 <code>{{ switching }}</code> 切换到新的节点并重载 dae，请稍候…
    </NAlert>
    <NText v-if="!loading && fixedGroups.length" depth="3" class="node-selection-hint">
      延迟标签测的是节点主机的入口网络延迟（公网 ICMP 中位数、内网 TCP 握手），不经过 dae，不验证代理端口或协议是否可用。
    </NText>

    <NSpin :show="loading">
      <NCard v-if="!loading && fixedGroups.length === 0" class="panel-card">
        <NEmpty class="empty-state" data-testid="fixed-group-empty" description="还没有 fixed 分组。把某个分组的策略设为 fixed(n) 后，可以在这里直接点选候选节点快速切换。">
          <template #extra>
            <RouterLink :to="{ name: 'orchestration' }" custom>
              <template #default="{ navigate }">
                <NButton type="primary" @click="navigate">前往代理编排</NButton>
              </template>
            </RouterLink>
          </template>
        </NEmpty>
      </NCard>

      <div v-else class="node-selection-list" data-testid="fixed-group-list">
        <NCard
          v-for="entry in fixedGroups"
          :key="entry.group.name"
          class="panel-card fixed-group-card"
          :data-testid="`fixed-group-card-${entry.group.name}`"
        >
          <template #header>
            <div class="fixed-group-head">
              <code>{{ entry.group.name }}</code>
              <span v-if="currentNodeName(entry)" class="fixed-group-current">当前 {{ currentNodeName(entry) }}</span>
              <NTag size="small" :bordered="false" type="info">fixed({{ entry.index }})</NTag>
              <NTag v-if="switching === entry.group.name" size="small" type="warning" :bordered="false">切换中…</NTag>
            </div>
          </template>
          <template #header-extra>
            <NButton
              v-if="entry.candidates.resolvable"
              size="small"
              secondary
              :loading="probingGroup === entry.group.name"
              :disabled="busy || latencyProbing"
              @click="probeGroup(entry)"
            >
              <template #icon><NIcon><FlashOutline /></NIcon></template>测入口延迟
            </NButton>
          </template>

          <NSpin :show="switching === entry.group.name">
            <div
              v-if="entry.candidates.resolvable"
              class="node-tile-grid"
              :data-testid="`fixed-group-grid-${entry.group.name}`"
            >
              <button
                v-for="(node, idx) in entry.candidates.nodes"
                :key="idx"
                type="button"
                class="node-tile"
                :class="{ selected: idx === entry.index }"
                :disabled="busy"
                :aria-pressed="idx === entry.index"
                :data-testid="`fixed-group-tile-${entry.group.name}-${idx}`"
                @click="switchTo(entry, idx)"
              >
                <NIcon v-if="idx === entry.index" class="node-tile-selected-mark"><CheckmarkCircleOutline /></NIcon>
                <span class="node-tile-name" :title="node.name">{{ node.name }}</span>
                <div class="node-tile-meta">
                  <span class="node-tile-desc">{{ describeCandidate(node) }}</span>
                  <NTag size="tiny" :type="latencyTypeOf(node)" :bordered="false" :title="latencyTitleOf(node)">
                    {{ latencyLabelOf(node) }}
                  </NTag>
                </div>
              </button>
            </div>
            <div v-else class="fixed-group-fallback">
              <NAlert type="warning" :bordered="false">{{ entry.candidates.reason }}</NAlert>
              <div class="fixed-group-index-row">
                <NText depth="3">按索引切换</NText>
                <NInputNumber
                  :value="entry.index"
                  :min="0"
                  :precision="0"
                  :disabled="busy"
                  :data-testid="`fixed-group-index-${entry.group.name}`"
                  @update:value="(value: number | null) => value !== null && switchTo(entry, value)"
                />
              </div>
            </div>
          </NSpin>
        </NCard>
      </div>
    </NSpin>
  </div>
</template>
