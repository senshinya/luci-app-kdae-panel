<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NCard, NDataTable, NIcon, NInput, NModal, NSpace, NTag, NText, useDialog, useMessage, type DataTableColumns } from 'naive-ui'
import { CreateOutline, PencilOutline, RefreshOutline, ReturnUpBackOutline, TrashOutline } from '@vicons/ionicons5'
import { APIError, deleteJSON, getJSON, postJSON, putJSON } from '../api/client'
import type { ConfigBackup, ConfigDocument, ConfigSaveResult } from '../types/api'
import { formatBytes, formatDateTime, shortHash } from '../utils/format'

const message = useMessage()
const dialog = useDialog()
const loading = ref(true)
const restoring = ref('')
const deleting = ref('')
const backups = ref<ConfigBackup[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const editingID = ref('')
const draftName = ref('')
const draftNote = ref('')

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
    width: 250,
    fixed: 'right',
    render: (row) => h(
      NSpace,
      { size: 6, align: 'center', wrap: false },
      {
        default: () => [
          h(NButton, {
            size: 'small', secondary: true, type: 'primary',
            loading: restoring.value === row.id,
            disabled: Boolean(restoring.value || deleting.value),
            onClick: () => confirmRestore(row),
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

function confirmRestore(backup: ConfigBackup) {
  dialog.warning({
    title: '恢复配置备份',
    content: `将恢复“${backup.name || '自动备份'}”（${formatDateTime(backup.createdAt)}），并执行 dae reload。当前配置也会先生成新备份。`,
    positiveText: '恢复并重载',
    negativeText: '取消',
    onPositiveClick: () => restore(backup),
  })
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

async function restore(backup: ConfigBackup) {
  restoring.value = backup.id
  try {
    let expectedHash = ''
    try {
      expectedHash = (await getJSON<ConfigDocument>('/api/v1/config')).hash
    } catch (error) {
      if (!(error instanceof APIError && error.status === 404)) throw error
    }
    await postJSON<ConfigSaveResult>(`/api/v1/config/backups/${encodeURIComponent(backup.id)}/restore`, {
      expectedHash,
      apply: true,
    })
    message.success('配置已恢复并完成无损重载')
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
        :columns="columns"
        :data="backups"
        :loading="loading"
        :row-key="(row: ConfigBackup) => row.id"
        :scroll-x="920"
        :bordered="false"
      />
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
  </div>
</template>
