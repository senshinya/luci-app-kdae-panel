<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref, type VNode } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NIcon,
  NRadioButton,
  NRadioGroup,
  NSpace,
  NTag,
  NText,
  NTooltip,
  useDialog,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  CloudDownloadOutline,
  RefreshOutline,
  ReturnUpBackOutline,
  SwapHorizontalOutline,
  TrashOutline,
} from '@vicons/ionicons5'
import { APIError, deleteJSON, getJSON, postJSON } from '../api/client'
import type {
  InstallJob,
  InstallProvision,
  InstallStatus,
  UpstreamSource,
  UpstreamVersion,
} from '../types/api'
import { formatBytes, formatDateTime } from '../utils/format'
import { useBackendStore } from '../stores/backend'
import { useJobPolling } from '../composables/useJobPolling'
import { SOURCES } from '../components/versions/sources'
import InstallStatusCard from '../components/versions/InstallStatusCard.vue'
import GeoCard from '../components/versions/GeoCard.vue'

const message = useMessage()
const dialog = useDialog()
// 卸载与首次安装的确认框描述的是不可逆操作，两套 init 系统下面板做的事不同，
// 措辞必须跟着后端走，否则用户是照着不存在的行为在做决定。
const backend = useBackendStore()
const loading = ref(true)
const listing = ref(false)
// loading 只在首屏为真，之后再也不会回到 true，因此它挡不住刷新按钮被连点。
const refreshing = ref(false)
const disabled = ref(false)
const status = ref<InstallStatus | null>(null)
const provision = ref<InstallProvision | null>(null)
const job = ref<InstallJob | null>(null)
const versions = ref<UpstreamVersion[]>([])
const source = ref<UpstreamSource>('official')
const loadError = ref('')
const listError = ref('')
const cacheDeleting = ref('')

let unmounted = false

const installPolling = useJobPolling({
  refresh: () => loadStatus(),
  phase: () => job.value?.phase,
  onSettled: (phase) => {
    if (phase === 'done') {
      message.success('已完成')
      void loadVersions()
    }
    else if (phase === 'failed') message.error(job.value?.error || '安装失败')
  },
})

const activeSource = computed(() => SOURCES.find((item) => item.value === source.value)!)
const busy = computed(() => job.value?.phase === 'downloading' || job.value?.phase === 'applying')
const installedRef = computed(() => status.value?.managed?.ref || '')
function isInstalled(version: UpstreamVersion): boolean {
  return version.ref === installedRef.value && version.source === status.value?.managed?.source
}

function versionKey(version: UpstreamVersion): string {
  return `${version.source}:${version.ref}`
}
const canUninstall = computed(() => status.value?.ready === true && status.value.managed !== undefined && !status.value.drifted)
const uninstallHint = computed(() => {
  if (status.value?.drifted) return 'dae 已在面板之外被替换，请先重装一个版本后再卸载'
  if (!status.value?.managed) return '当前 dae 没有面板安装记录，为避免误删外部安装，不能自动卸载'
  return ''
})
const purgeConfig = ref(false)
const purgeGeo = ref(false)

async function loadStatus() {
  try {
    const payload = await getJSON<{ status: InstallStatus; job: InstallJob; provision?: InstallProvision }>(
      '/api/v1/dae/install',
    )
    status.value = payload.status
    job.value = payload.job
    // 安装进行中后端会略去 provision（探测有副作用，不宜每两秒做一次）。
    // 此时沿用上一次的结果，避免界面在"首次安装"与"当前安装"之间来回跳。
    if (payload.provision) {
      provision.value = payload.provision
    } else if (payload.status.ready) {
      provision.value = null
    }
    loadError.value = ''
    disabled.value = false
  } catch (error) {
    if (error instanceof APIError && error.code === 'dae_install_disabled') {
      disabled.value = true
      loadError.value = error.message
    } else {
      loadError.value = error instanceof Error ? error.message : '读取安装状态失败'
    }
  } finally {
    loading.value = false
  }
}

// 只认最后一次发出的请求。用序号而不是比对来源：连点刷新会发出多个同来源的
// 请求，它们全都满足"来源没变"，于是先回来的那个仍会被后回来的旧结果盖掉。
let versionRequest = 0

async function loadVersions() {
  if (disabled.value) return
  const requested = source.value
  const ticket = ++versionRequest
  listing.value = true
  try {
    const payload = await getJSON<{ versions: UpstreamVersion[] }>(
      `/api/v1/dae/versions?source=${requested}&limit=30`,
    )
    if (ticket !== versionRequest || unmounted) return
    versions.value = payload.versions
    listError.value = ''
  } catch (error) {
    if (ticket !== versionRequest || unmounted) return
    versions.value = []
    listError.value = error instanceof Error ? error.message : '读取版本列表失败'
  } finally {
    if (ticket === versionRequest) listing.value = false
  }
}

function changeSource(next: UpstreamSource) {
  if (next === source.value) return
  source.value = next
  versions.value = []
  // 上一个来源的失败与新来源无关，留着它会挂在新来源的空列表上方
  listError.value = ''
  void loadVersions()
}

async function refreshAll() {
  refreshing.value = true
  try {
    await loadStatus()
    if (unmounted) return
    await loadVersions()
  } finally {
    refreshing.value = false
  }
}

// 首次安装与升级是两件不同的事，确认框必须分别说清楚会发生什么。
const firstInstall = computed(() => provision.value?.possible === true)

async function confirmInstall(version: UpstreamVersion) {
  // 空闲时页面不轮询，firstInstall 可能是打开页面那一刻的快照——期间别人
  // 装好了 dae，确认框就会承诺写单元和 geo，而后端实际走的是替换升级。
  // 确认框描述的必须是后端此刻真正会做的事，所以先把状态取新。
  await loadStatus()
  if (unmounted) return
  if (firstInstall.value) {
    dialog.warning({
      title: `安装 ${version.label}`,
      content: `面板会下载并校验该版本，然后安装可执行文件到 ${provision.value?.binaryPath}、`
        // procd 下 Plan 返回 inPlace、Commit 是空操作：启动脚本随软件包装好，
        // 面板从不写它，承诺"写入服务单元"是假的。
        + (backend.isProcd
          ? '写入 geo 数据与种子配置（启动脚本由软件包提供，已就位）。'
          : `写入 geo 数据与服务单元 ${provision.value?.unitPath}。`)
        + '装完不会自动启动 dae——请先在配置管理页写好规则，再手动启动，'
        + '否则透明代理可能切断你当前的连接。',
      positiveText: '下载并安装',
      negativeText: '取消',
      onPositiveClick: () => install(version),
    })
    return
  }
  const local = version.cached === true
  dialog.warning({
    title: `安装 ${version.label}`,
    content: (local
      ? '面板会读取并重新校验本地版本，用它验证当前配置，然后替换二进制并重启 dae。'
      : '面板会下载并校验该版本，用它验证当前配置，然后替换二进制并重启 dae。')
      + '重启会中断现有连接；若新版本起不来，会自动回滚到当前版本。',
    positiveText: local ? '使用本地版本' : '下载并安装',
    negativeText: '取消',
    onPositiveClick: () => install(version),
  })
}

async function install(version: UpstreamVersion) {
  try {
    const payload = await postJSON<{ job: InstallJob }>('/api/v1/dae/install', {
      source: version.source,
      ref: version.ref,
      label: version.label,
    })
    job.value = payload.job
    message.info(version.cached && !firstInstall.value ? '已开始使用本地版本切换' : '已开始下载安装')
    installPolling.start()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '启动安装失败')
    // 409 说明后端已有任务在跑，界面必须同步过去而不是继续显示空闲
    if (error instanceof APIError && error.status === 409) {
      await loadStatus()
      if (busy.value) installPolling.start()
    }
  }
}

function confirmDeleteCached(version: UpstreamVersion) {
  const current = isInstalled(version)
  const size = formatBytes(version.cachedBytes)
  dialog.warning({
    title: `删除本地版本 ${version.label}`,
    content: `将删除这份 ${size} 的本地缓存。`
      + (current ? '当前运行中的 dae 不受影响；以后再次安装该版本需要重新下载。' : '当前运行文件和回滚点不受影响。'),
    positiveText: '删除缓存',
    negativeText: '取消',
    onPositiveClick: () => deleteCached(version),
  })
}

async function deleteCached(version: UpstreamVersion) {
  const key = versionKey(version)
  cacheDeleting.value = key
  try {
    await deleteJSON<void>('/api/v1/dae/cache', { source: version.source, ref: version.ref })
    message.success(`已删除 ${version.label} 的本地缓存`)
    await loadVersions()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除本地版本失败')
  } finally {
    cacheDeleting.value = ''
  }
}

function confirmRollback() {
  dialog.warning({
    title: '回滚到上一版本',
    content: '面板会把安装前备份的二进制换回去并重启 dae。',
    positiveText: '回滚',
    negativeText: '取消',
    onPositiveClick: rollback,
  })
}

async function rollback() {
  try {
    const payload = await postJSON<{ job: InstallJob }>('/api/v1/dae/rollback')
    job.value = payload.job
    message.info('已开始回滚')
    installPolling.start()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '启动回滚失败')
    if (error instanceof APIError && error.status === 409) {
      await loadStatus()
      if (busy.value) installPolling.start()
    }
  }
}

async function confirmUninstall() {
  // 卸载是破坏性操作，确认框必须依据点击当下的状态，而不是首屏快照。
  await loadStatus()
  if (unmounted) return
  if (!canUninstall.value) {
    message.warning(uninstallHint.value || '当前 dae 无法由面板卸载')
    return
  }
  purgeConfig.value = false
  purgeGeo.value = false
  dialog.warning({
    title: '卸载 dae',
    content: () => h(NSpace, { vertical: true, size: 12 }, {
      default: () => [
        h(NText, null, {
          // procd 下 RemovablePaths 返回空：/etc/init.d/dae 归软件包所有，
          // 卸载 dae 不删它——删了用户就再也没法从面板重新装回 dae。
          default: () => backend.isProcd
            ? '这会停止 dae，现有代理连接会立即中断，并删除由面板管理的可执行文件和版本回滚记录（启动脚本由软件包提供，不会删除）。'
            : '这会停止 dae，现有代理连接会立即中断，并删除由面板管理的可执行文件、systemd 服务单元和版本回滚记录。',
        }),
        h(NCheckbox, {
          checked: purgeConfig.value,
          'onUpdate:checked': (value: boolean) => { purgeConfig.value = value },
        }, { default: () => '同时删除 dae 主配置文件' }),
        h(NCheckbox, {
          checked: purgeGeo.value,
          'onUpdate:checked': (value: boolean) => { purgeGeo.value = value },
        }, { default: () => '同时删除面板可见的全部 geo 数据副本' }),
        h(NText, { depth: 3 }, {
          // 那句限定只在面板被沙箱挡住时成立。procd 部署没有 ProtectHome，
          // /root/.local/share/dae 对面板完全可见，卸载会把它一并删掉。
          default: () => backend.isProcd
            ? '未勾选的数据会保留，之后可在本页重新安装。'
            : '未勾选的数据会保留，之后可在本页重新安装。面板沙箱看不到的 /root/.local/share/dae 不会被删除。',
        }),
      ],
    }),
    positiveText: '停止并卸载',
    negativeText: '取消',
    onPositiveClick: () => uninstall({ purgeConfig: purgeConfig.value, purgeGeo: purgeGeo.value }),
  })
}

async function uninstall(options: { purgeConfig: boolean; purgeGeo: boolean }) {
  try {
    const payload = await postJSON<{ job: InstallJob }>('/api/v1/dae/uninstall', options)
    job.value = payload.job
    message.info('已开始卸载 dae')
    installPolling.start()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '启动卸载失败')
    if (error instanceof APIError && error.status === 409) {
      await loadStatus()
      if (busy.value) installPolling.start()
    }
  }
}

const columns = computed<DataTableColumns<UpstreamVersion>>(() => [
  {
    title: '版本',
    key: 'label',
    width: 170,
    render: (row) => h(NSpace, { size: 4, align: 'center', wrap: false }, {
      default: () => [
        h('span', { class: 'mono version-label' }, row.label),
        isInstalled(row)
          ? h(NTag, { size: 'tiny', type: 'success', bordered: false }, { default: () => '当前' })
          : null,
        row.cached
          ? h(NTooltip, null, {
              trigger: () => h(NTag, { size: 'tiny', type: 'info', bordered: false }, { default: () => '已下载' }),
              default: () => `本地缓存 ${formatBytes(row.cachedBytes)} · ${formatDateTime(row.cachedAt)}`,
            })
          : null,
        row.prerelease
          ? h(NTag, { size: 'tiny', type: 'warning', bordered: false }, { default: () => '预发布' })
          : null,
      ].filter(Boolean),
    }),
  },
  {
    title: source.value === 'kdae' ? '提交说明' : '发布名称',
    key: 'description',
    minWidth: 220,
    ellipsis: { tooltip: true },
    render: (row) => row.description || h(NText, { depth: 3 }, { default: () => '—' }),
  },
  {
    title: source.value === 'kdae' ? '构建时间' : '发布时间',
    key: 'publishedAt',
    width: 200,
    render: (row) => {
      const published = formatDateTime(row.publishedAt)
      if (!row.expiresAt || !row.installable) return published
      // 临近过期的 CI 构建要看得出来，否则它和新构建长得一模一样
      const days = Math.ceil((new Date(row.expiresAt).getTime() - Date.now()) / 86400000)
      if (!Number.isFinite(days) || days > 14) return published
      // 后端按 CreatedAt+90 天推算过期，GitHub 的实际保留期可能更短，
      // 于是会出现 installable 仍为真但已过期的行——别渲染成"-3 天后过期"。
      return h(NSpace, { size: 4, align: 'center', wrap: false }, {
        default: () => [
          published,
          h(NTag, { size: 'tiny', type: days > 0 ? 'warning' : 'error', bordered: false }, {
            default: () => days > 0 ? `${days} 天后过期` : '可能已过期',
          }),
        ],
      })
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 158,
    fixed: 'right',
    render: (row) => {
      if (!row.installable) {
        return h(NTooltip, null, {
          trigger: () => h(NTag, { size: 'small', type: 'error', bordered: false }, { default: () => '已过期' }),
          default: () => row.note || '该版本无法安装',
        })
      }
      const actions: VNode[] = []
      // 磁盘被外部替换过时，"已安装"这条也要能重装回来修复漂移
      if (isInstalled(row) && !status.value?.drifted) {
        actions.push(h(NText, { depth: 3 }, { default: () => '已安装' }))
      } else {
        const local = row.cached && !firstInstall.value
        actions.push(h(NButton, {
          size: 'small',
          secondary: true,
          type: 'primary',
          disabled: busy.value || !(status.value?.ready || firstInstall.value),
          onClick: () => void confirmInstall(row),
        }, {
          icon: () => h(NIcon, null, { default: () => h(local ? SwapHorizontalOutline : CloudDownloadOutline) }),
          default: () => firstInstall.value ? '安装' : '切换',
        }))
      }
      if (row.cached) {
        actions.push(h(NTooltip, null, {
          trigger: () => h(NButton, {
            size: 'small',
            quaternary: true,
            type: 'error',
            'aria-label': `删除 ${row.label} 的本地缓存`,
            loading: cacheDeleting.value === versionKey(row),
            disabled: busy.value || cacheDeleting.value !== '',
            onClick: () => confirmDeleteCached(row),
          }, { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => isInstalled(row) ? '删除缓存，不影响当前运行' : '删除本地缓存',
        }))
      }
      return h(NSpace, { size: 4, align: 'center', wrap: false }, { default: () => actions })
    },
  },
])

onMounted(async () => {
  // 不 await：后端只影响文案，让它去挡首屏数据加载得不偿失。
  void backend.ensure()
  await loadStatus()
  if (unmounted) return
  await loadVersions()
  if (unmounted) return
  // 任务可能在本页打开之前就已在跑，这时也要接上轮询
  if (busy.value) installPolling.start()
})
// 轮询的卸载清理由 useJobPolling 自己挂钩，这里只管本组件的加载链
onBeforeUnmount(() => {
  unmounted = true
})
</script>

<template>
  <div class="page-stack">
    <div class="page-toolbar">
      <div>
        <h2>dae 版本</h2>
        <NText depth="3">在官方发布与 kdae 构建之间切换，安装前校验、失败自动回滚</NText>
      </div>
      <NSpace>
        <NButton
          secondary
          :loading="listing || loading || refreshing"
          :disabled="disabled || refreshing"
          @click="refreshAll"
        >
          <template #icon><NIcon><RefreshOutline /></NIcon></template>刷新
        </NButton>
        <NButton
          v-if="status?.rollbackAvailable"
          :disabled="busy || !status?.ready"
          @click="confirmRollback"
        >
          <template #icon><NIcon><ReturnUpBackOutline /></NIcon></template>回滚上一版
        </NButton>
        <NTooltip v-if="status?.ready" :disabled="canUninstall">
          <template #trigger>
            <span>
              <NButton type="error" secondary :disabled="busy || !canUninstall" @click="confirmUninstall">
                <template #icon><NIcon><TrashOutline /></NIcon></template>卸载 dae
              </NButton>
            </span>
          </template>
          {{ uninstallHint }}
        </NTooltip>
      </NSpace>
    </div>

    <NAlert v-if="disabled" type="info" :bordered="false">
      {{ loadError }}
    </NAlert>
    <NAlert v-else-if="loadError" type="error" :bordered="false">{{ loadError }}</NAlert>

    <template v-if="!disabled">
      <NAlert v-if="job?.phase === 'downloading'" type="info" :bordered="false">
        正在下载并校验 {{ job.label || job.ref }}…
      </NAlert>
      <NAlert v-else-if="job?.phase === 'applying'" type="warning" :bordered="false">
        <template v-if="job.label === '卸载 dae'">正在停止服务并卸载 dae，数据将按确认时的选择处理…</template>
        <template v-else-if="job.cached">正在使用本地版本替换二进制并重启 dae，期间连接会短暂中断…</template>
        <template v-else>正在替换二进制并重启 dae，期间连接会短暂中断…</template>
      </NAlert>
      <NAlert v-else-if="job?.phase === 'failed'" type="error" :bordered="false">
        上次操作失败：{{ job.error }}
      </NAlert>

      <InstallStatusCard :loading="loading" :busy="busy" :status="status" :provision="provision" />

      <NCard class="panel-card" content-style="padding: 0;">
        <template #header>
          <NRadioGroup :value="source" size="small" @update:value="changeSource">
            <NRadioButton v-for="item in SOURCES" :key="item.value" :value="item.value">
              {{ item.label }}
            </NRadioButton>
          </NRadioGroup>
        </template>
        <div class="source-hint">
          <NText depth="3">{{ activeSource.hint }}</NText>
        </div>
        <NAlert v-if="listError" type="error" :bordered="false" class="source-hint">{{ listError }}</NAlert>
        <NDataTable
          :columns="columns"
          :data="versions"
          :loading="listing"
          :row-key="(row: UpstreamVersion) => row.ref"
          :scroll-x="820"
          :bordered="false"
          size="small"
        >
          <template #empty>
            <div class="orchestrate-empty">
              <NText depth="3">没有可用版本</NText>
            </div>
          </template>
        </NDataTable>
      </NCard>
    </template>

    <!-- geo 数据是独立开关，因此不放在 dae 版本管理的 v-if 里：
         只开其中一个的部署是正常情况 -->
    <GeoCard />
  </div>
</template>
