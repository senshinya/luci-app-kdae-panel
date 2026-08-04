<script setup lang="ts">
import { computed, ref } from 'vue'
import { NAlert, NCard, NInputNumber, NSelect, NSpin, NTag, NText, useMessage } from 'naive-ui'
import { APIError, putJSON } from '../../api/client'
import type { ConfigSaveResult, SubscriptionNodeSource } from '../../types/api'
import { parseGroups, type Group } from '../../utils/daeconf'
import { parseFixedIndex, resolveFixedCandidates, type FixedCandidates } from '../../utils/fixedgroup'

// 概览页的快速切换卡片：点选即提交，服务端直接写配置并重载 dae，不经过编排页的
// 本地缓冲，也不产生自动备份。候选顺序不可解时退回索引数字框，绝不展示可能错误的节点名。
const props = defineProps<{
  content: string
  hash: string
  sources: SubscriptionNodeSource[]
  sourcesLoaded: boolean
}>()
const emit = defineEmits<{ switched: []; conflict: [] }>()
const message = useMessage()

// 记录正在提交的分组名；非空时整卡视为禁用，避免并发提交。
const pending = ref('')

interface FixedGroupRow {
  group: Group
  index: number
  candidates: FixedCandidates
}

const rows = computed<FixedGroupRow[]>(() => {
  const result: FixedGroupRow[] = []
  for (const group of parseGroups(props.content)) {
    const index = parseFixedIndex(group.policy?.value)
    if (index === null) continue
    result.push({ group, index, candidates: resolveFixedCandidates(props.content, group, props.sources, props.sourcesLoaded) })
  }
  return result
})

/** 候选可解时取出节点名列表，供模板窄化联合类型，避免在 template 里做类型断言。 */
function candidateNodes(candidates: FixedCandidates): string[] {
  return candidates.resolvable ? candidates.nodes : []
}

/** 候选不可解时取出原因文案；可解时返回空字符串。 */
function candidateReason(candidates: FixedCandidates): string {
  return candidates.resolvable ? '' : candidates.reason
}

function optionsOf(candidates: FixedCandidates) {
  return candidateNodes(candidates).map((name, index) => ({ label: name, value: index }))
}

function describeTarget(row: FixedGroupRow, index: number): string {
  return candidateNodes(row.candidates)[index] || `第 ${index} 个节点`
}

async function switchTo(row: FixedGroupRow, index: number | null) {
  // NInputNumber 在清空重输时，失焦那一刻可能先把值汇报为 null；
  // 绝不能把它当成 0 处理，否则清空数字框再点别处就会误切到 fixed(0)。
  if (index === null || pending.value !== '' || index === row.index) return
  if (!Number.isInteger(index) || index < 0) {
    message.error('fixed(n) 的索引必须是从 0 开始的整数')
    return
  }
  pending.value = row.group.name
  try {
    const result = await putJSON<ConfigSaveResult>(`/api/v1/groups/${encodeURIComponent(row.group.name)}/policy`, {
      policy: `fixed(${index})`,
      expectedHash: props.hash,
    })
    message.success(result.deferred
      ? `已把 ${row.group.name} 切到 ${describeTarget(row, index)}；dae 未运行，下次启动生效`
      : `已把 ${row.group.name} 切到 ${describeTarget(row, index)} 并完成无损重载`)
    // 切换成功后配置哈希已变化，必须让父组件重新读取，否则下一次切换的 expectedHash 必然过期。
    emit('switched')
  } catch (error) {
    if (error instanceof APIError && error.status === 409) {
      message.error('配置已在别处变化，已重新读取，请再试一次')
      emit('conflict')
    } else {
      message.error(error instanceof Error ? error.message : '切换分组节点失败')
    }
    // 不手动回滚选择器：content/hash 均未改变，下面按 props 重新渲染的值本就是原值。
  } finally {
    pending.value = ''
  }
}
</script>

<template>
  <NCard v-if="rows.length" title="分组选择" class="panel-card" data-testid="fixed-group-card">
    <template #header-extra>
      <NTag size="small" :bordered="false">{{ rows.length }} 个固定分组</NTag>
    </template>
    <NSpin :show="pending !== ''">
      <div class="fixed-group-list">
        <div v-for="row in rows" :key="row.group.name" class="fixed-group-row" :data-testid="`fixed-group-row-${row.group.name}`">
          <code>{{ row.group.name }}</code>
          <NSelect
            v-if="row.candidates.resolvable"
            size="small"
            :value="row.index"
            :options="optionsOf(row.candidates)"
            :loading="pending === row.group.name"
            :disabled="pending !== ''"
            :consistent-menu-width="false"
            :data-testid="`fixed-group-select-${row.group.name}`"
            @update:value="(value: number) => switchTo(row, value)"
          />
          <template v-else>
            <NInputNumber
              size="small"
              class="fixed-group-index"
              :min="0"
              :precision="0"
              :value="row.index"
              :loading="pending === row.group.name"
              :disabled="pending !== ''"
              :data-testid="`fixed-group-index-${row.group.name}`"
              @update:value="(value: number | null) => switchTo(row, value)"
            />
            <NText depth="3" class="fixed-group-reason">{{ candidateReason(row.candidates) }}</NText>
          </template>
        </div>
      </div>
    </NSpin>
    <NAlert type="info" :bordered="false" class="inline-alert">
      切换会直接写入配置并重载 dae，不产生自动备份。
    </NAlert>
  </NCard>
</template>

<style scoped>
.fixed-group-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.fixed-group-row {
  display: grid;
  grid-template-columns: minmax(80px, 160px) minmax(0, 1fr);
  align-items: center;
  gap: 8px 12px;
}

.fixed-group-index {
  width: 120px;
}

.fixed-group-reason {
  grid-column: 2;
  font-size: 12px;
}

@media (max-width: 767px) {
  .fixed-group-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .fixed-group-reason {
    grid-column: 1;
  }
}
</style>
