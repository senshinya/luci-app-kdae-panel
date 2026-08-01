<script setup lang="ts">
import { computed, h, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NIcon,
  NInput,
  NInputGroup,
  NInputNumber,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  NText,
  NTooltip,
  useMessage,
  type SelectOption,
} from 'naive-ui'
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import {
  addGroup,
  isValidTag,
  parseGroups,
  readSection,
  removeGroup,
  setGroupFilter,
  setGroupPolicy,
  type Group,
} from '../../utils/daeconf'
import {
  createGroupFilter,
  describeGroupFilter,
  knownFixedCandidateCount,
  parseGroupFilter,
  serializeGroupFilter,
  type GroupFilterDraft,
  type GroupFilterKind,
} from '../../utils/group'
import { parseNodeLink } from '../../utils/nodelink'
import SectionEditorModal from './SectionEditorModal.vue'

const content = defineModel<string>({ required: true })
const message = useMessage()

const groups = computed<Group[]>(() => parseGroups(content.value))
const groupNames = computed(() => new Set(groups.value.map((group) => group.name)))
const sourceVisible = ref(false)

const POLICY_OPTIONS = [
  { label: 'min_moving_avg · 移动平均延迟最小', value: 'min_moving_avg' },
  { label: 'min_avg10 · 最近 10 次平均延迟最小', value: 'min_avg10' },
  { label: 'min · 最近一次延迟最小', value: 'min' },
  { label: 'random · 每次连接随机', value: 'random' },
  { label: 'fixed(n) · 固定第 n 个节点', value: 'fixed' },
]

function parsePolicy(value?: string): { name: string; index: number } {
  const fixed = /^fixed\((\d+)\)$/.exec(value || '')
  if (fixed) return { name: 'fixed', index: Number(fixed[1]) }
  return { name: value || 'min_moving_avg', index: 0 }
}

function policyOptionsFor(group: Group) {
  const current = parsePolicy(group.policy?.value).name
  if (POLICY_OPTIONS.some((option) => option.value === current)) return POLICY_OPTIONS
  return [...POLICY_OPTIONS, { label: current, value: current }]
}

const newGroupName = ref('')
const newGroupPolicy = ref('min_moving_avg')

function createGroup() {
  const name = newGroupName.value.trim()
  if (!isValidTag(name)) {
    message.error('分组名只能使用字母、数字、下划线、点或横线，且以字母或下划线开头')
    return
  }
  if (groupNames.value.has(name)) {
    message.error(`分组 ${name} 已存在`)
    return
  }
  content.value = addGroup(content.value, name, newGroupPolicy.value === 'fixed' ? 'fixed(0)' : newGroupPolicy.value)
  newGroupName.value = ''
}

function changePolicy(group: Group, name: string) {
  if (name === 'fixed' && !validFixedIndex(group, parsePolicy(group.policy?.value).index)) return
  const serialized = name === 'fixed' ? `fixed(${parsePolicy(group.policy?.value).index})` : name
  content.value = setGroupPolicy(content.value, group, serialized)
}

function changeFixedIndex(group: Group, index: number | null) {
  const normalized = index ?? 0
  if (!validFixedIndex(group, normalized)) return
  content.value = setGroupPolicy(content.value, group, `fixed(${normalized})`)
}

const FILTER_KIND_OPTIONS = [
  { label: '指定节点', value: 'nodes' },
  { label: '指定订阅', value: 'subscriptions' },
  { label: '节点名称关键词', value: 'nameKeyword' },
  { label: '节点名称正则', value: 'nameRegex' },
  { label: '高级表达式', value: 'raw' },
]

interface ResourceOption extends SelectOption {
  label: string
  value: string
  description?: string
}

const subscriptionOptions = computed<ResourceOption[]>(() => readSection(content.value, 'subscription').entries
  .filter((entry) => entry.tag)
  .map((entry) => ({ label: entry.tag!, value: entry.tag!, description: entry.value })))

const nodeOptions = computed<ResourceOption[]>(() => {
  const options: ResourceOption[] = []
  const seen = new Set<string>()
  for (const entry of readSection(content.value, 'node').entries) {
    const info = parseNodeLink(entry.value)
    const value = entry.tag?.trim() || ''
    if (value === '' || seen.has(value)) continue
    seen.add(value)
    const details = [
      entry.tag && info?.name && info.name !== entry.tag ? info.name : '',
      info?.protocol,
      info?.host,
    ].filter(Boolean).join(' · ')
    options.push({ label: value, value, description: details || undefined })
  }
  return options
})

function renderResourceLabel(option: SelectOption) {
  const label = typeof option.label === 'string' ? option.label : String(option.value || '')
  const description = typeof option.description === 'string' ? option.description : ''
  return h('div', { class: 'group-resource-option' }, [
    h('strong', label),
    description ? h('span', { class: 'group-resource-option-meta' }, description) : null,
  ])
}

const groupEditVisible = ref(false)
const groupTarget = ref<{ index: number; snapshot: string } | null>(null)
const groupPolicy = ref('min_moving_avg')
const groupFixedIndex = ref(0)
const groupFilters = ref<GroupFilterDraft[]>([])
const unknownNodeNames = computed(() => {
  const explicit = new Set(nodeOptions.value.map((option) => option.value))
  return [...new Set(groupFilters.value
    .filter((filter) => filter.kind === 'nodes')
    .flatMap((filter) => filter.values)
    .filter((value) => !explicit.has(value)))]
})

function openGroupEditor(groupIndex: number) {
  const group = groups.value[groupIndex]
  if (!group) return
  if ((group.policy && !group.policy.editable) || group.filters.some((filter) => !filter.editable)) {
    message.warning('该分组含跨行或重复声明，请使用卡片右上角的原文编辑')
    return
  }
  const policy = parsePolicy(group.policy?.value)
  groupTarget.value = {
    index: groupIndex,
    snapshot: content.value.slice(group.section.nameStart, group.section.bodyEnd + 1),
  }
  groupPolicy.value = policy.name
  groupFixedIndex.value = policy.index
  groupFilters.value = group.filters.map((filter) => parseGroupFilter(filter.value))
  groupEditVisible.value = true
}

function addFilter(kind: GroupFilterKind) {
  groupFilters.value.push(createGroupFilter(kind))
}

function applyGroupEdit() {
  const target = groupTarget.value
  if (!target) return
  const current = groups.value[target.index]
  if (!current || content.value.slice(current.section.nameStart, current.section.bodyEnd + 1) !== target.snapshot) {
    message.error('配置在编辑期间发生了变化，请关闭后重新打开')
    return
  }
  const serialized = groupFilters.value.map(serializeGroupFilter)
  if (serialized.some((value) => value === null)) {
    message.error('过滤条件不能为空；请选择节点或订阅，名称也不能同时含单双引号')
    return
  }
  if (groupPolicy.value === 'fixed') {
    if (!Number.isInteger(groupFixedIndex.value) || groupFixedIndex.value < 0) {
      message.error('fixed(n) 的索引必须是从 0 开始的整数')
      return
    }
    const candidateCount = fixedCandidateCount(groupFilters.value)
    if (candidateCount !== null && (candidateCount === 0 || groupFixedIndex.value >= candidateCount)) {
      message.error(candidateCount === 0
        ? '当前分组没有可供 fixed(0) 选择的节点'
        : `fixed(${groupFixedIndex.value}) 已越界；当前过滤条件只有 ${candidateCount} 个明确节点`)
      return
    }
  }

  let next = content.value
  for (let index = current.filters.length - 1; index >= 0; index -= 1) {
    next = setGroupFilter(next, current, index, serialized[index] || '')
  }
  for (let index = current.filters.length; index < serialized.length; index += 1) {
    const latest = parseGroups(next)[target.index]
    if (!latest) return
    next = setGroupFilter(next, latest, latest.filters.length, serialized[index]!)
  }
  const latest = parseGroups(next)[target.index]
  if (!latest) return
  const policy = groupPolicy.value === 'fixed' ? `fixed(${groupFixedIndex.value})` : groupPolicy.value
  content.value = setGroupPolicy(next, latest, policy)
  groupEditVisible.value = false
  groupTarget.value = null
}

function validFixedIndex(group: Group, index: number): boolean {
  if (!Number.isInteger(index) || index < 0) {
    message.error('fixed(n) 的索引必须是从 0 开始的整数')
    return false
  }
  const candidateCount = fixedCandidateCount(group.filters.map((filter) => parseGroupFilter(filter.value)))
  if (candidateCount === null || (candidateCount > 0 && index < candidateCount)) return true
  message.error(candidateCount === 0
    ? '当前分组没有可供 fixed(0) 选择的节点'
    : `fixed(${index}) 已越界；当前过滤条件只有 ${candidateCount} 个明确节点`)
  return false
}

function fixedCandidateCount(filters: GroupFilterDraft[]): number | null {
  const explicit = new Set(nodeOptions.value.map((option) => option.value))
  if (filters.some((filter) => filter.kind === 'nodes' && filter.values.some((value) => !explicit.has(value)))) {
    return null
  }
  return knownFixedCandidateCount(
    filters,
    readSection(content.value, 'node').entries.length,
    readSection(content.value, 'subscription').entries.length > 0,
  )
}
</script>

<template>
  <NCard title="分组" class="panel-card" data-testid="groups-card">
    <template #header-extra>
      <NSpace size="small" align="center">
        <NTag size="small" :bordered="false">{{ groups.length }} 个</NTag>
        <NButton size="small" quaternary @click="sourceVisible = true">
          <template #icon><NIcon><CreateOutline /></NIcon></template>编辑原文
        </NButton>
      </NSpace>
    </template>
    <div class="orchestrate-add inset" data-testid="group-add">
      <NInputGroup>
        <NInput v-model:value="newGroupName" placeholder="新分组名，如 proxy" @keyup.enter="createGroup" />
        <NSelect v-model:value="newGroupPolicy" :options="POLICY_OPTIONS" class="group-policy-select" />
        <NButton type="primary" ghost @click="createGroup">
          <template #icon><NIcon><AddOutline /></NIcon></template>新建
        </NButton>
      </NInputGroup>
    </div>
    <div v-if="groups.length === 0" class="orchestrate-empty" data-testid="group-list">
      <NText depth="3">还没有分组。分组是路由规则的出站目标，按策略从命中的节点中选择。</NText>
    </div>
    <div v-for="(group, groupIndex) in groups" :key="groupIndex" class="group-item" data-testid="group-list">
      <div class="group-head">
        <code>{{ group.name }}</code>
        <NSpace size="small">
          <NButton size="tiny" secondary @click="openGroupEditor(groupIndex)">
            <template #icon><NIcon><CreateOutline /></NIcon></template>编辑
          </NButton>
          <NPopconfirm positive-text="删除" negative-text="取消" @positive-click="content = removeGroup(content, group)">
            <template #trigger>
              <NButton size="tiny" quaternary type="error" title="删除分组">
                <template #icon><NIcon><TrashOutline /></NIcon></template>
              </NButton>
            </template>
            删除分组后，引用它的路由规则会校验失败，确认删除？
          </NPopconfirm>
        </NSpace>
      </div>
      <div class="group-row">
        <NText depth="3">策略</NText>
        <NSelect
          size="small"
          :value="parsePolicy(group.policy?.value).name"
          :options="policyOptionsFor(group)"
          :disabled="group.policy !== null && !group.policy.editable"
          @update:value="(value: string) => changePolicy(group, value)"
        />
        <NInputNumber
          v-if="parsePolicy(group.policy?.value).name === 'fixed'"
          size="small"
          class="group-fixed-index"
          :min="0"
          :precision="0"
          :value="parsePolicy(group.policy?.value).index"
          @update:value="(value: number | null) => changeFixedIndex(group, value)"
        />
      </div>
      <div class="group-row filters">
        <NText depth="3">过滤</NText>
        <NSpace size="small" wrap>
          <NTooltip v-for="(filter, index) in group.filters" :key="index" :disabled="filter.editable">
            <template #trigger>
              <NTag
                size="small"
                class="filter-tag mono"
                :class="{ locked: !filter.editable }"
                :closable="filter.editable"
                @close="content = setGroupFilter(content, group, index, '')"
              >
                <span class="filter-value" :title="filter.value" @click="filter.editable && openGroupEditor(groupIndex)">
                  {{ describeGroupFilter(filter.value) }}
                </span>
              </NTag>
            </template>
            该条件跨行或结构复杂，请使用卡片右上角的原文编辑。
          </NTooltip>
          <NButton size="tiny" dashed @click="openGroupEditor(groupIndex)">
            <template #icon><NIcon><AddOutline /></NIcon></template>
            {{ group.filters.length === 0 ? '全部节点，选择成员' : '编辑成员' }}
          </NButton>
        </NSpace>
      </div>
    </div>
  </NCard>

  <NModal v-model:show="groupEditVisible" preset="card" title="编辑分组" class="orchestrate-group-modal" :mask-closable="false" data-testid="group-editor-modal">
    <NSpace vertical size="small">
      <NAlert type="info" :bordered="false">
        可直接选择本地节点或订阅；多条过滤之间是“或”关系。订阅内部节点由 dae 拉取，面板按订阅整体加入。
      </NAlert>
      <NAlert v-if="unknownNodeNames.length" type="warning" :bordered="false">
        {{ unknownNodeNames.join('、') }} 不对应显式本地标签，可能是订阅节点名称或旧配置名称；面板会原样保留，但无法保证它与 dae 实际解析出的名称一致。
      </NAlert>
    </NSpace>
    <div class="group-editor-policy">
      <NText depth="3">策略</NText>
      <NSelect v-model:value="groupPolicy" :options="POLICY_OPTIONS" />
      <NInputNumber v-if="groupPolicy === 'fixed'" v-model:value="groupFixedIndex" :min="0" :precision="0" />
    </div>
    <div class="group-filter-editor">
      <div class="group-filter-editor-head">
        <div>
          <strong>分组成员</strong>
          <NText depth="3" class="group-resource-hint">
            本地节点只使用显式标签，避免分享链接备注与 dae 运行时名称不一致；未打标签的订阅也无法由 subtag 稳定引用。
          </NText>
        </div>
        <NSpace size="small" class="group-filter-actions">
          <NButton size="small" dashed :disabled="nodeOptions.length === 0" @click="addFilter('nodes')">
            <template #icon><NIcon><AddOutline /></NIcon></template>选择节点
          </NButton>
          <NButton size="small" dashed :disabled="subscriptionOptions.length === 0" @click="addFilter('subscriptions')">
            <template #icon><NIcon><AddOutline /></NIcon></template>选择订阅
          </NButton>
          <NButton size="small" quaternary @click="addFilter('nameKeyword')">
            <template #icon><NIcon><AddOutline /></NIcon></template>高级条件
          </NButton>
        </NSpace>
      </div>
      <NText v-if="groupFilters.length === 0" depth="3">不设置过滤时，该分组包含全部节点。</NText>
      <div v-for="(filter, index) in groupFilters" :key="index" class="group-filter-editor-row">
        <NSelect v-model:value="filter.kind" :options="FILTER_KIND_OPTIONS" />
        <NSelect
          v-if="filter.kind === 'nodes'"
          v-model:value="filter.values"
          :options="nodeOptions"
          :render-label="renderResourceLabel"
          multiple
          filterable
          clearable
          max-tag-count="responsive"
          :virtual-scroll="false"
          :consistent-menu-width="false"
          placeholder="选择本地节点"
          data-testid="group-node-picker"
        />
        <NSelect
          v-else-if="filter.kind === 'subscriptions'"
          v-model:value="filter.values"
          :options="subscriptionOptions"
          :render-label="renderResourceLabel"
          multiple
          filterable
          tag
          clearable
          max-tag-count="responsive"
          :virtual-scroll="false"
          :consistent-menu-width="false"
          placeholder="选择订阅"
          data-testid="group-subscription-picker"
        />
        <NInput
          v-else
          v-model:value="filter.value"
          class="mono"
          :placeholder="filter.kind === 'raw' ? 'subtag(my_sub) && !name(keyword: 过期)' : '匹配内容'"
          spellcheck="false"
        />
        <label v-if="filter.kind !== 'raw'" class="filter-exclude">
          <NSwitch v-model:value="filter.exclude" size="small" />
          <span>排除</span>
        </label>
        <span v-else class="filter-exclude-placeholder"></span>
        <NButton quaternary circle type="error" title="删除条件" @click="groupFilters.splice(index, 1)">
          <template #icon><NIcon><TrashOutline /></NIcon></template>
        </NButton>
      </div>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="groupEditVisible = false">取消</NButton>
        <NButton type="primary" @click="applyGroupEdit">应用到编排</NButton>
      </NSpace>
    </template>
  </NModal>

  <SectionEditorModal
    v-model:show="sourceVisible"
    v-model:content="content"
    section="group"
    title="编辑分组原文"
    description="这里只替换 group 节内部内容，适合处理跨行条件或分组级高级参数。其他配置保持不变。"
  />
</template>
