<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NEmpty, NIcon, NInput, NModal, NSpace, NSpin, NTag, NText, useDialog, useMessage, type DataTableColumns } from 'naive-ui'
import { CreateOutline, GitCompareOutline, PencilOutline, RefreshOutline, ReturnUpBackOutline, TrashOutline } from '@vicons/ionicons5'
import { deleteJSON, getJSON, postJSON, putJSON } from '../api/client'
import type { ConfigBackup, ConfigBackupPreview, ConfigSaveResult } from '../types/api'
import { useMobileViewport } from '../composables/useMobileViewport'
import { formatBytes, formatDateTime, shortHash } from '../utils/format'

const message = useMessage()
const dialog = useDialog()
const mobile = useMobileViewport()
const loading = ref(true)
const restoring = ref('')
const deleting = ref('')
const backups = ref<ConfigBackup[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const editingID = ref('')
const draftName = ref('')
const draftNote = ref('')
const previewVisible = ref(false)
const previewLoading = ref('')
const preview = ref<ConfigBackupPreview | null>(null)

const columns: DataTableColumns<ConfigBackup> = [
  {
    title: '名称',
    key: 'name',
    minWidth: 170,
    render: (row) => row.name || h(NText, { depth: 3 }, { default: () => '自动备份' }),
  },
  {
    title: '备注',
    key: 'note',
    minWidth: 220,
    ellipsis: { tooltip: true },
    render: (row) => row.note || h(NText, { depth: 3 }, { default: () => '—' }),
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 180,
    render: (row) => formatDateTime(row.createdAt),
  },
  {
    title: '内容摘要',
    key: 'hash',
    minWidth: 150,
    render: (row) => h(NTag, { size: 'small', bordered: false }, { default: () => shortHash(row.hash) }),
  },
  {
    title: '大小',
    key: 'size',
    width: 110,
    render: (row) => formatBytes(row.size),
  },
  {
    title: '备份编号',
    key: 'id',
    minWidth: 320,
    ellipsis: { tooltip: true },
  },
  {
    title: '操作',
    key: 'actions',
    width: 286,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      { size: 6, align: 'center', wrap: false },
      {
        default: () => [
          h(NButton, {
            size: 'small', quaternary: true, title: '与当前配置比较',
            loading: previewLoading.value === row.id,
            disabled: Boolean(restoring.value || deleting.value || previewLoading.value),
            onClick: () => void openPreview(row),
          }, {
            icon: () => h(NIcon, null, { default: () => h(GitCompareOutline) }),
          }),
          h(NButton, {
            size: 'small', secondary: true, type: 'primary',
            loading: restoring.value === row.id,
            disabled: Boolean(restoring.value || deleting.value || previewLoading.value),
            onClick: () => void openPreview(row),
          }, {
            icon: () => h(NIcon, null, { default: () => h(ReturnUpBackOutline) }),
            default: () => '恢复',
          }),
          h(NButton, {
            size: 'small', quaternary: true, title: '编辑名称和备注',
            disabled: Boolean(restoring.value || deleting.value),
            onClick: () => openEditor(row),
          }, {
            icon: () => h(NIcon, null, { default: () => h(PencilOutline) }),
          }),
          h(NButton, {
            size: 'small', quaternary: true, type: 'error', title: '删除配置存档',
            loading: deleting.value === row.id,
            disabled: Boolean(restoring.value || deleting.value),
            onClick: () => confirmDelete(row),
          }, {
            icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
          }),
        ],
      },
    ),
  },
]

async function load() {
  loading.value = true
  try {
    backups.value = await getJSON<ConfigBackup[]>('/api/v1/config/backups')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '读取备份失败')
  } finally {
    loading.value = false
  }
}

async function openPreview(backup: ConfigBackup) {
  previewLoading.value = backup.id
  try {
    preview.value = await getJSON<ConfigBackupPreview>(
      `/api/v1/config/backups/${encodeURIComponent(backup.id)}/preview`,
    )
    previewVisible.value = true
  } catch (error) {
    message.error(error instanceof Error ? error.message : '比较配置存档失败')
  } finally {
    previewLoading.value = ''
  }
}

function openEditor(backup?: ConfigBackup) {
  editingID.value = backup?.id || ''
  draftName.value = backup?.name || ''
  draftNote.value = backup?.note || ''
  editorVisible.value = true
}

async function saveMetadata() {
  saving.value = true
  try {
    if (editingID.value) {
      await putJSON<ConfigBackup>(`/api/v1/config/backups/${encodeURIComponent(editingID.value)}`, {
        name: draftName.value,
        note: draftNote.value,
      })
      message.success('配置存档信息已更新')
    } else {
      await postJSON<ConfigBackup>('/api/v1/config/backups', {
        name: draftName.value,
        note: draftNote.value,
      })
      message.success('当前配置已保存为存档')
    }
    editorVisible.value = false
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存配置存档失败')
  } finally {
    saving.value = false
  }
}

function confirmDelete(backup: ConfigBackup) {
  dialog.warning({
    title: '删除配置存档',
    content: `将删除“${backup.name || '自动备份'}”及其配置内容，删除后无法恢复。`,
    positiveText: '删除存档',
    negativeText: '取消',
    onPositiveClick: () => deleteBackup(backup),
  })
}

async function deleteBackup(backup: ConfigBackup) {
  deleting.value = backup.id
  try {
    await deleteJSON<void>(`/api/v1/config/backups/${encodeURIComponent(backup.id)}`, {})
    message.success('配置存档已删除')
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除配置存档失败')
  } finally {
    deleting.value = ''
  }
}

async function restore(preflight: ConfigBackupPreview) {
  const backup = preflight.backup
  restoring.value = backup.id
  try {
    const result = await postJSON<ConfigSaveResult>(`/api/v1/config/backups/${encodeURIComponent(backup.id)}/restore`, {
      expectedHash: preflight.currentHash,
      apply: true,
    })
    previewVisible.value = false
    message.success(result.deferred
      ? '配置已恢复；dae 当前未运行，下次启动时生效'
      : '配置已恢复并完成无损重载')
    await load()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '恢复配置失败')
  } finally {
    restoring.value = ''
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="page-stack backups-page">
    <div class="page-toolbar">
      <div>
        <h2>配置历史</h2>
        <NText depth="3">保存当前配置或查看自动历史；恢复操作同样受并发摘要保护</NText>
      </div>
      <NSpace>
        <NButton type="primary" secondary @click="openEditor()">
          <template #icon><NIcon><CreateOutline /></NIcon></template>保存当前配置
        </NButton>
        <NButton secondary :loading="loading" @click="load">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>刷新
        </NButton>
      </NSpace>
    </div>
    <NCard content-style="padding: 0;">
      <NDataTable
        v-if="!mobile"
        :columns="columns"
        :data="backups"
        :loading="loading"
        :row-key="(row: ConfigBackup) => row.id"
        :scroll-x="920"
        :bordered="false"
      />
      <NSpin v-else :show="loading">
        <div v-if="backups.length" class="mobile-record-list" data-testid="mobile-backup-list">
          <article v-for="backup in backups" :key="backup.id" class="mobile-record">
            <div class="mobile-record-head">
              <div class="mobile-record-title">{{ backup.name || '自动备份' }}</div>
              <NTag size="small" :bordered="false">{{ shortHash(backup.hash) }}</NTag>
            </div>
            <p v-if="backup.note" class="mobile-record-description">{{ backup.note }}</p>
            <div class="mobile-record-meta">
              <span>创建<strong>{{ formatDateTime(backup.createdAt) }}</strong></span>
              <span>大小<strong>{{ formatBytes(backup.size) }}</strong></span>
            </div>
            <div class="mobile-action-row">
              <NButton
                secondary
                type="primary"
                :loading="restoring === backup.id"
                :disabled="Boolean(restoring || deleting || previewLoading)"
                @click="openPreview(backup)"
              >
                <template #icon><NIcon><ReturnUpBackOutline /></NIcon></template>恢复
              </NButton>
              <NButton
                secondary
                :loading="previewLoading === backup.id"
                :disabled="Boolean(restoring || deleting || previewLoading)"
                @click="openPreview(backup)"
              >
                <template #icon><NIcon><GitCompareOutline /></NIcon></template>比较
              </NButton>
              <NButton secondary :disabled="Boolean(restoring || deleting)" @click="openEditor(backup)">
                <template #icon><NIcon><PencilOutline /></NIcon></template>编辑
              </NButton>
              <NButton
                secondary
                type="error"
                :loading="deleting === backup.id"
                :disabled="Boolean(restoring || deleting)"
                @click="confirmDelete(backup)"
              >
                <template #icon><NIcon><TrashOutline /></NIcon></template>删除
              </NButton>
            </div>
          </article>
        </div>
        <NEmpty v-else description="还没有配置存档" class="mobile-empty" />
      </NSpin>
    </NCard>

    <NModal v-model:show="editorVisible" preset="card" :title="editingID ? '编辑配置存档' : '保存当前配置'" class="backup-editor-modal">
      <div class="backup-editor-form">
        <label>
          <NText>名称</NText>
          <NInput v-model:value="draftName" maxlength="80" show-count placeholder="例如：稳定线路" />
        </label>
        <label>
          <NText>备注</NText>
          <NInput v-model:value="draftNote" type="textarea" maxlength="500" show-count :autosize="{ minRows: 3, maxRows: 6 }" placeholder="记录这份配置的用途或适用场景" />
        </label>
      </div>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="editorVisible = false">取消</NButton>
          <NButton type="primary" :loading="saving" :disabled="!draftName.trim()" @click="saveMetadata">
            {{ editingID ? '保存修改' : '保存存档' }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal
      v-model:show="previewVisible"
      preset="card"
      :title="`配置差异 · ${preview?.backup.name || '自动备份'}`"
      class="backup-diff-modal"
    >
      <template v-if="preview">
        <NAlert v-if="!preview.valid" type="error" :bordered="false">
          这份存档无法通过当前 dae 的配置校验，不能恢复。
          <pre class="backup-validation-error">{{ preview.validationError }}</pre>
        </NAlert>
        <NAlert v-else-if="preview.same" type="success" :bordered="false">
          这份存档与当前配置内容相同，无需恢复。
        </NAlert>
        <NAlert v-else-if="preview.diffTruncated" type="warning" :bordered="false">
          配置差异较大，下面只展示有边界的结果；目标配置仍已完整通过 dae 校验。
        </NAlert>
        <div class="backup-diff-legend">
          <span class="backup-diff-remove">− 当前配置删除</span>
          <span class="backup-diff-add">+ 存档配置加入</span>
        </div>
        <div class="backup-diff" role="region" aria-label="当前配置与存档配置差异">
          <div
            v-for="(line, index) in preview.diff"
            :key="`${index}:${line.kind}:${line.oldLine}:${line.newLine}`"
            class="backup-diff-line"
            :class="`backup-diff-${line.kind}`"
          >
            <template v-if="line.kind === 'skip'">
              <span class="backup-diff-skip-copy">{{ line.text }}（{{ line.skipCount }} 行）</span>
            </template>
            <template v-else>
              <span class="backup-diff-number">{{ line.oldLine || '' }}</span>
              <span class="backup-diff-number">{{ line.newLine || '' }}</span>
              <span class="backup-diff-mark">{{ line.kind === 'add' ? '+' : line.kind === 'remove' ? '−' : ' ' }}</span>
              <code>{{ line.text || ' ' }}</code>
            </template>
          </div>
        </div>
      </template>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="previewVisible = false">关闭</NButton>
          <NButton
            type="primary"
            :loading="Boolean(preview && restoring === preview.backup.id)"
            :disabled="!preview?.valid || preview.same"
            @click="preview && restore(preview)"
          >
            <template #icon><NIcon><ReturnUpBackOutline /></NIcon></template>恢复并重载
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
