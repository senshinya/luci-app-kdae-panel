<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NInput,
  NInputGroup,
  NModal,
  NSelect,
  NSpace,
  NSwitch,
  NTag,
  NText,
  NTooltip,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { AddOutline, CreateOutline, RefreshOutline, TimerOutline, TrashOutline } from '@vicons/ionicons5'
import { getJSON, postJSON, putJSON } from '../../api/client'
import { useMobileViewport } from '../../composables/useMobileViewport'
import type { ScheduleStatus } from '../../types/api'
import { appendToSection, isQuotable, isValidTag, quote, readSection, removeLine, type Entry } from '../../utils/daeconf'
import { parseScheme, supportsPersistence, togglePersistence } from '../../utils/subscription'
import { formatDateTime } from '../../utils/format'
import { entryActions, useEntryRewrite, type EntryTarget } from './entry'
import SectionEditorModal from './SectionEditorModal.vue'

// 订阅内容始终由 dae 在 reload 时拉取，面板只负责维护配置里的那一行。
const content = defineModel<string>({ required: true })
const props = defineProps<{
  /** 有未保存的编排时不允许"立即刷新"：重载应用的是磁盘配置，会绕过这些改动。 */
  dirty: boolean
}>()

const message = useMessage()
const mobile = useMobileViewport()
const dialog = useDialog()
const { captureEntry, rewriteEntry } = useEntryRewrite(content, message)

const subscriptions = computed<Entry[]>(() => readSection(content.value, 'subscription').entries)
const sourceVisible = ref(false)

// ---- 添加 ----
const subscriptionTag = ref('')
const subscriptionURL = ref('')
const subscriptionPersist = ref(true)

function subscriptionLine(tag: string, url: string): string {
  return tag === '' ? quote(url) : `${tag}: ${quote(url)}`
}

function validSubscription(tag: string, url: string): boolean {
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(url) || !isQuotable(url)) {
    message.error('请输入完整的订阅链接，且不能同时包含单引号和双引号')
    return false
  }
  if (tag !== '' && !isValidTag(tag)) {
    message.error('订阅标签只能使用字母、数字、下划线、点或横线，且以字母或下划线开头')
    return false
  }
  return true
}

function addSubscription() {
  const tag = subscriptionTag.value.trim()
  let url = subscriptionURL.value.trim()
  if (!validSubscription(tag, url)) return
  // 开关是双向的：粘贴的链接已带 -file 而开关关闭时，同样要剥掉后缀
  if (supportsPersistence(url) && parseScheme(url)?.persistent !== subscriptionPersist.value) {
    url = togglePersistence(url)
  }
  content.value = appendToSection(content.value, 'subscription', [subscriptionLine(tag, url)])
  subscriptionTag.value = ''
  subscriptionURL.value = ''
}

function setPersistence(entry: Entry, persistent: boolean) {
  const next = togglePersistence(entry.value)
  // 开关状态与目标不符时说明重复触发或链接形态不支持，直接忽略
  if (next === entry.value || parseScheme(next)?.persistent !== persistent) return
  // 值与标签来自配置本身，仍需确认它们能被无损写回
  if (!validSubscription(entry.tag || '', next)) return
  rewriteEntry(captureEntry(entry), subscriptionLine(entry.tag || '', next))
}

// ---- 编辑 ----
const editTarget = ref<EntryTarget | null>(null)
const editTag = ref('')
const editURL = ref('')

function openSubscriptionEditor(entry: Entry) {
  editTarget.value = captureEntry(entry)
  editTag.value = entry.tag || ''
  editURL.value = entry.value
}

function applySubscriptionEdit() {
  const target = editTarget.value
  if (!target) return
  const tag = editTag.value.trim()
  const url = editURL.value.trim()
  if (!validSubscription(tag, url)) return
  if (rewriteEntry(target, subscriptionLine(tag, url))) editTarget.value = null
}

const subscriptionColumns: DataTableColumns<Entry> = [
  {
    title: '标签',
    key: 'tag',
    width: 110,
    ellipsis: { tooltip: true },
    render: (row) => row.tag
      ? h(NTag, { size: 'small', bordered: false }, { default: () => row.tag })
      : h(NText, { depth: 3 }, { default: () => '—' }),
  },
  {
    title: '订阅链接',
    key: 'value',
    minWidth: 200,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'mono' }, row.value),
  },
  {
    title: () => h(NTooltip, null, {
      trigger: () => h('span', { class: 'column-hint' }, '离线缓存'),
      default: () => '开启后 dae 会把拉取成功的订阅内容保存到 persist.d 目录，'
        + '下次拉取失败时回退使用该缓存，成功后自动更新。对 file:// 本地订阅不适用。',
    }),
    key: 'persist',
    width: 100,
    render: (row) => {
      if (!supportsPersistence(row.value)) return h(NText, { depth: 3 }, { default: () => '—' })
      return h(NSwitch, {
        size: 'small',
        value: parseScheme(row.value)?.persistent === true,
        disabled: !row.editable,
        'onUpdate:value': (value: boolean) => setPersistence(row, value),
      })
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row) => entryActions(row, [
      { title: '编辑', icon: CreateOutline, onClick: () => openSubscriptionEditor(row) },
      {
        title: '移除',
        icon: TrashOutline,
        type: 'error',
        onClick: () => { content.value = removeLine(content.value, row.lineStart, row.lineEnd) },
      },
    ], true),
  },
]

// ---- 订阅刷新:dae 只在 reload 时重新拉取订阅 ----
const refreshing = ref(false)
const scheduleVisible = ref(false)
const scheduleSaving = ref(false)
const reloadSchedule = ref<ScheduleStatus | null>(null)
const scheduleEnabled = ref(false)
const scheduleInterval = ref(1440)

const INTERVAL_OPTIONS = [
  { label: '每 30 分钟', value: 30 },
  { label: '每小时', value: 60 },
  { label: '每 6 小时', value: 360 },
  { label: '每 12 小时', value: 720 },
  { label: '每天', value: 1440 },
  { label: '每周', value: 10080 },
]

const scheduleError = ref('')

async function loadSchedule() {
  try {
    const status = await getJSON<ScheduleStatus>('/api/v1/schedule/reload')
    reloadSchedule.value = status
    scheduleEnabled.value = status.enabled
    scheduleInterval.value = status.intervalMinutes
    scheduleError.value = ''
  } catch (error) {
    // 读不到就不允许保存，否则会用表单默认值覆盖服务端的真实设置
    reloadSchedule.value = null
    scheduleError.value = error instanceof Error ? error.message : '读取自动刷新设置失败'
  }
}

// 每次打开都以服务端状态为准，避免上次取消时留下的编辑残留
async function openSchedule() {
  scheduleVisible.value = true
  await loadSchedule()
}

async function saveSchedule() {
  scheduleSaving.value = true
  try {
    reloadSchedule.value = await putJSON<ScheduleStatus>('/api/v1/schedule/reload', {
      enabled: scheduleEnabled.value,
      intervalMinutes: scheduleInterval.value,
    })
    scheduleVisible.value = false
    message.success(scheduleEnabled.value ? '已开启订阅自动刷新' : '已关闭订阅自动刷新')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存自动刷新设置失败')
  } finally {
    scheduleSaving.value = false
  }
}

function confirmRefreshNow() {
  if (props.dirty) {
    message.warning('有未保存的编排修改，请先保存并重载')
    return
  }
  dialog.info({
    title: '立即刷新订阅',
    content: 'dae 会在无损重载时重新拉取全部订阅链接，现有连接不会中断。'
      + '重载应用的是磁盘上的当前配置，因此之前"仅保存"而未应用的改动也会一并生效。',
    positiveText: '刷新并重载',
    negativeText: '取消',
    onPositiveClick: refreshNow,
  })
}

async function refreshNow() {
  refreshing.value = true
  try {
    await postJSON('/api/v1/service/actions/reload')
    message.success('已触发无损重载，订阅内容将在重载后生效')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '刷新订阅失败')
  } finally {
    refreshing.value = false
  }
}

const scheduleSummary = computed(() => {
  const status = reloadSchedule.value
  if (!status) return ''
  if (!status.enabled) return '自动刷新已关闭'
  const option = INTERVAL_OPTIONS.find((item) => item.value === status.intervalMinutes)
  return option ? option.label : `每 ${status.intervalMinutes} 分钟`
})

onMounted(() => void loadSchedule())
</script>

<template>
  <NCard title="订阅" class="panel-card" content-style="padding: 0;" data-testid="subscriptions-card">
    <template #header-extra>
      <NSpace size="small" align="center">
        <NTag size="small" :bordered="false">{{ subscriptions.length }} 个</NTag>
        <NButton size="small" quaternary :loading="refreshing" :disabled="subscriptions.length === 0" @click="confirmRefreshNow">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>立即刷新
        </NButton>
        <NButton size="small" quaternary @click="openSchedule">
          <template #icon><NIcon><TimerOutline /></NIcon></template>自动刷新
        </NButton>
        <NButton size="small" quaternary @click="sourceVisible = true">
          <template #icon><NIcon><CreateOutline /></NIcon></template>编辑原文
        </NButton>
      </NSpace>
    </template>
    <div class="orchestrate-add" data-testid="subscription-add">
      <NInputGroup>
        <NInput v-model:value="subscriptionTag" placeholder="标签(可选)" class="orchestrate-tag-input" />
        <NInput v-model:value="subscriptionURL" placeholder="https://example.com/subscription" @keyup.enter="addSubscription" />
        <NButton type="primary" ghost @click="addSubscription">
          <template #icon><NIcon><AddOutline /></NIcon></template>添加
        </NButton>
      </NInputGroup>
      <div class="orchestrate-add-hint">
        <NSwitch v-model:value="subscriptionPersist" size="small" />
        <NText depth="3">启用离线缓存（写作 https-file://，拉取失败时回退到上次成功的内容）</NText>
      </div>
      <NText v-if="scheduleSummary" depth="3" class="schedule-summary">
        订阅刷新：{{ scheduleSummary }}<template v-if="reloadSchedule?.nextRunAt">，下次 {{ formatDateTime(reloadSchedule.nextRunAt) }}</template>
      </NText>
    </div>
    <NDataTable
      v-if="!mobile"
      data-testid="subscription-list"
      :columns="subscriptionColumns"
      :data="subscriptions"
      :row-key="(row: Entry) => row.lineStart"
      :bordered="false"
      :scroll-x="620"
      size="small"
    >
      <template #empty>
        <div class="orchestrate-empty">
          <NText depth="3">还没有订阅。订阅内容由 dae 在重载时拉取。</NText>
        </div>
      </template>
    </NDataTable>
    <template v-else>
      <div v-if="subscriptions.length" class="mobile-record-list" data-testid="mobile-subscription-list">
        <article v-for="entry in subscriptions" :key="entry.lineStart" class="mobile-record">
          <div class="mobile-record-head">
            <div class="mobile-record-title">
              <span>{{ entry.tag || '未命名订阅' }}</span>
              <NTag v-if="parseScheme(entry.value)?.persistent" size="tiny" type="info" :bordered="false">离线缓存</NTag>
            </div>
          </div>
          <p class="mobile-record-description mono">{{ entry.value }}</p>
          <div v-if="supportsPersistence(entry.value)" class="mobile-record-toggle">
            <div>
              <strong>离线缓存</strong>
              <NText depth="3">拉取失败时使用上次成功内容</NText>
            </div>
            <NSwitch
              size="small"
              :value="parseScheme(entry.value)?.persistent === true"
              :disabled="!entry.editable"
              @update:value="(value: boolean) => setPersistence(entry, value)"
            />
          </div>
          <div class="mobile-action-row">
            <NButton secondary :disabled="!entry.editable" @click="openSubscriptionEditor(entry)">
              <template #icon><NIcon><CreateOutline /></NIcon></template>编辑
            </NButton>
            <NButton
              secondary
              type="error"
              :disabled="!entry.editable"
              @click="content = removeLine(content, entry.lineStart, entry.lineEnd)"
            >
              <template #icon><NIcon><TrashOutline /></NIcon></template>移除
            </NButton>
          </div>
        </article>
      </div>
      <NEmpty v-else description="还没有订阅。订阅内容由 dae 在重载时拉取。" class="mobile-empty" />
    </template>
  </NCard>

  <NModal :show="editTarget !== null" preset="card" title="编辑订阅" class="orchestrate-modal" @update:show="editTarget = null">
    <NText depth="3">修改标签或链接地址。订阅内容始终由 dae 在下次重载时重新拉取。</NText>
    <NInput v-model:value="editTag" placeholder="标签(可选)" spellcheck="false" />
    <NInput v-model:value="editURL" class="mono" placeholder="https://example.com/subscription" spellcheck="false" @keyup.enter="applySubscriptionEdit" />
    <template #footer>
      <NSpace justify="end">
        <NButton @click="editTarget = null">取消</NButton>
        <NButton type="primary" @click="applySubscriptionEdit">确定</NButton>
      </NSpace>
    </template>
  </NModal>

  <NModal v-model:show="scheduleVisible" preset="card" title="订阅自动刷新" class="orchestrate-modal">
    <NText depth="3">
      dae 只在无损重载时重新拉取订阅，因此自动刷新的实现就是按间隔执行一次 dae reload。
      面板有其他控制操作正在执行时会跳过当轮，不会与之交叉。
    </NText>
    <NAlert v-if="scheduleError" type="error" :bordered="false" class="card-alert schedule-alert">
      {{ scheduleError }}
    </NAlert>
    <div class="schedule-row">
      <NSwitch v-model:value="scheduleEnabled" :disabled="scheduleError !== ''" />
      <NText>{{ scheduleEnabled ? '已开启' : '已关闭' }}</NText>
    </div>
    <NSelect v-model:value="scheduleInterval" :options="INTERVAL_OPTIONS" :disabled="!scheduleEnabled || scheduleError !== ''" />
    <dl v-if="reloadSchedule" class="details-list schedule-details">
      <div>
        <dt>上次执行</dt>
        <dd>{{ reloadSchedule.lastRunAt ? formatDateTime(reloadSchedule.lastRunAt) : '尚未执行' }}</dd>
      </div>
      <div v-if="reloadSchedule.nextRunAt">
        <dt>下次执行</dt>
        <dd>{{ formatDateTime(reloadSchedule.nextRunAt) }}</dd>
      </div>
      <div v-if="reloadSchedule.lastError">
        <dt>上次结果</dt>
        <dd>{{ reloadSchedule.lastError }}</dd>
      </div>
    </dl>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="scheduleVisible = false">取消</NButton>
        <NButton type="primary" :loading="scheduleSaving" :disabled="scheduleError !== ''" @click="saveSchedule">保存</NButton>
      </NSpace>
    </template>
  </NModal>

  <SectionEditorModal
    v-model:show="sourceVisible"
    v-model:content="content"
    section="subscription"
    title="编辑订阅原文"
    description="这里只替换 subscription 节内部内容，其他配置与注释保持不变。应用后仍需在页面顶部保存或保存并重载。"
  />
</template>
