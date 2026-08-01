<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NGrid,
  NGridItem,
  NIcon,
  NPopconfirm,
  NSkeleton,
  NSpace,
  NTag,
  NText,
  useMessage,
} from 'naive-ui'
import {
  ArrowForwardOutline,
  PauseOutline,
  PlayOutline,
  RefreshOutline,
  ReloadOutline,
  StopOutline,
} from '@vicons/ionicons5'
import { RouterLink } from 'vue-router'
import { APIError, getJSON, postJSON } from '../api/client'
import type { ConfigDocument, DaeReport, ServiceStatus } from '../types/api'
import { formatBytes, formatElapsedSince } from '../utils/format'
import { parseGroups, parseRoutingRules, readSection } from '../utils/daeconf'
import { useBackendStore } from '../stores/backend'

const message = useMessage()
const loading = ref(true)
const refreshing = ref(false)
const actionLoading = ref('')
const service = ref<ServiceStatus | null>(null)
const dae = ref<DaeReport | null>(null)
const configContent = ref<string | null>(null)
const serviceError = ref('')
const daeError = ref('')
const configError = ref('')
const now = ref(Date.now())
let clockTimer: number | undefined

const running = computed(() => service.value?.activeState === 'active')
const suspended = computed(() => service.value?.suspended === true)
const statusType = computed(() => suspended.value ? 'warning' : running.value ? 'success' : service.value?.activeState === 'failed' ? 'error' : 'warning')
const statusLabel = computed(() => suspended.value ? '已暂停' : running.value ? '运行中' : service.value?.activeState === 'failed' ? '运行失败' : '未运行')
const supportedCommands = computed(() => Object.entries(dae.value?.commands || {}).filter(([, enabled]) => enabled).map(([name]) => name))
const uptime = computed(() => running.value
  ? formatElapsedSince(service.value?.startedAt || service.value?.activeSince, now.value)
  : '—')
const startsOnBoot = computed(() => service.value?.unitFileState === 'enabled')

const orchestration = computed(() => {
  const text = configContent.value
  if (text === null) return null
  const count = (name: string) => readSection(text, name).entries.length
  return {
    nodes: count('node'),
    subscriptions: count('subscription'),
    groups: parseGroups(text),
    rules: parseRoutingRules(text).length,
  }
})

async function refresh(silent = false) {
  if (silent) refreshing.value = true
  else loading.value = true
  const [serviceResult, daeResult, configResult] = await Promise.allSettled([
    getJSON<ServiceStatus>('/api/v1/service'),
    getJSON<DaeReport>('/api/v1/dae/capabilities'),
    getJSON<ConfigDocument>('/api/v1/config'),
  ])
  if (serviceResult.status === 'fulfilled') {
    service.value = serviceResult.value
    serviceError.value = ''
  } else {
    serviceError.value = serviceResult.reason instanceof Error ? serviceResult.reason.message : '无法读取服务状态'
  }
  if (daeResult.status === 'fulfilled') {
    dae.value = daeResult.value
    daeError.value = ''
  } else {
    daeError.value = daeResult.reason instanceof Error ? daeResult.reason.message : '无法探测 dae'
  }
  if (configResult.status === 'fulfilled') {
    configContent.value = configResult.value.content
    configError.value = ''
  } else if (configResult.reason instanceof APIError && configResult.reason.status === 404) {
    // 只有 404 才代表配置确实不存在；其他失败保留上次读到的内容并报错
    configContent.value = null
    configError.value = ''
  } else {
    configError.value = configResult.reason instanceof Error ? configResult.reason.message : '读取配置失败'
  }
  loading.value = false
  refreshing.value = false
}

async function runAction(action: string) {
  actionLoading.value = action
  try {
    const result = await postJSON<{ message?: string }>(`/api/v1/service/actions/${action}`)
    message.success(result.message || `${actionName(action)}已执行`)
    await new Promise((resolve) => window.setTimeout(resolve, 500))
    await refresh(true)
  } catch (error) {
    message.error(error instanceof Error ? error.message : `${actionName(action)}失败`)
  } finally {
    actionLoading.value = ''
  }
}

function actionName(action: string): string {
  return ({ start: '启动', stop: '停止', restart: '重启', reload: '重载', suspend: '暂停' } as Record<string, string>)[action] || action
}

// 数据来自 host.Manager，具体是 systemd 还是 procd 取决于部署，副标题跟着说。
const backend = useBackendStore()

onMounted(() => {
  void backend.ensure()
  void refresh()
  clockTimer = window.setInterval(() => { now.value = Date.now() }, 1000)
})

onBeforeUnmount(() => {
  if (clockTimer !== undefined) window.clearInterval(clockTimer)
})
</script>

<template>
  <div class="page-stack">
    <div class="page-toolbar">
      <div>
        <h2>运行状态</h2>
        <NText depth="3">{{ backend.isProcd ? '来自服务管理器与当前安装的 dae 二进制' : '来自 systemd 与当前安装的 dae 二进制' }}</NText>
      </div>
      <NButton secondary :loading="refreshing" @click="refresh(true)">
        <template #icon><NIcon><RefreshOutline /></NIcon></template>
        刷新
      </NButton>
    </div>

    <NAlert v-if="serviceError" type="error" closable @close="serviceError = ''">{{ serviceError }}</NAlert>
    <NAlert v-if="daeError" type="warning" closable @close="daeError = ''">{{ daeError }}</NAlert>
    <NAlert v-if="suspended" type="warning" :bordered="false" class="service-suspended-alert">
      <div class="service-suspended-copy">
        <strong>dae 已暂停</strong>
        <span>代理流量处理已停止，但 dae 进程仍在运行；点击“无损重载”即可恢复。</span>
      </div>
    </NAlert>

    <NGrid responsive="screen" cols="1 s:3" :x-gap="14" :y-gap="14">
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">服务状态</NText>
          <NSkeleton v-if="loading" text style="width: 60%" />
          <div v-else class="metric-value"><NTag :type="statusType" round>{{ statusLabel }}</NTag></div>
          <small>{{ suspended ? '代理流量处理已暂停' : service?.subState || '等待状态数据' }}</small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">内存占用</NText>
          <NSkeleton v-if="loading" text style="width: 65%" />
          <strong v-else class="metric-value">{{ formatBytes(service?.memoryBytes, 1) }}</strong>
          <small>{{ service?.tasks ?? '—' }} 个任务</small>
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard class="metric-card" size="small">
          <NText depth="3">本次运行时长</NText>
          <NSkeleton v-if="loading" text style="width: 65%" />
          <strong v-else class="metric-value">{{ uptime }}</strong>
          <small>主进程 PID {{ service?.mainPid || '—' }}</small>
        </NCard>
      </NGridItem>
    </NGrid>

    <NGrid class="equal-height-grid" responsive="screen" cols="1 l:2 xl:3" :x-gap="16" :y-gap="16">
      <NGridItem>
        <NCard title="服务控制" class="panel-card">
          <template #header-extra>
            <NSpace size="small" align="center">
              <NTag size="small" :type="startsOnBoot ? 'success' : 'default'">
                {{ startsOnBoot ? '随系统启动' : '不随系统启动' }}
              </NTag>
              <NTag size="small" :type="statusType">{{ service?.name || 'dae' }}</NTag>
            </NSpace>
          </template>
          <NSpace wrap>
            <NButton type="success" :disabled="actionLoading !== '' || running" :loading="actionLoading === 'start'" @click="runAction('start')">
              <template #icon><NIcon><PlayOutline /></NIcon></template>启动
            </NButton>
            <NButton type="primary" :disabled="actionLoading !== '' || !running" :loading="actionLoading === 'reload'" @click="runAction('reload')">
              <template #icon><NIcon><ReloadOutline /></NIcon></template>无损重载
            </NButton>
            <NPopconfirm positive-text="确认重启" negative-text="取消" @positive-click="runAction('restart')">
              <template #trigger>
                <NButton :disabled="actionLoading !== '' || !running" :loading="actionLoading === 'restart'">
                  <template #icon><NIcon><RefreshOutline /></NIcon></template>重启
                </NButton>
              </template>
              重启会中断现有连接，确认继续？
            </NPopconfirm>
            <NPopconfirm positive-text="确认暂停" negative-text="取消" @positive-click="runAction('suspend')">
              <template #trigger>
                <NButton :disabled="actionLoading !== '' || !running || suspended" :loading="actionLoading === 'suspend'">
                  <template #icon><NIcon><PauseOutline /></NIcon></template>暂停
                </NButton>
              </template>
              暂停后可通过无损重载恢复。
            </NPopconfirm>
            <NPopconfirm positive-text="确认停止" negative-text="取消" @positive-click="runAction('stop')">
              <template #trigger>
                <NButton type="error" ghost :disabled="actionLoading !== '' || !running" :loading="actionLoading === 'stop'">
                  <template #icon><NIcon><StopOutline /></NIcon></template>停止
                </NButton>
              </template>
              停止 dae 后代理流量将不可用。
            </NPopconfirm>
          </NSpace>
        </NCard>
      </NGridItem>

      <NGridItem>
        <NCard title="代理编排" class="panel-card">
          <template #header-extra>
            <RouterLink :to="{ name: 'orchestration' }" custom>
              <template #default="{ navigate }">
                <NButton size="small" quaternary type="primary" icon-placement="right" @click="navigate">
                  <template #icon><NIcon><ArrowForwardOutline /></NIcon></template>前往编排
                </NButton>
              </template>
            </RouterLink>
          </template>
          <NSkeleton v-if="loading" text :repeat="3" />
          <template v-else>
            <NAlert v-if="configError" type="warning" :bordered="false" class="card-alert">
              读取配置失败：{{ configError }}
            </NAlert>
            <template v-if="orchestration">
              <div class="orchestration-stats">
                <div><strong>{{ orchestration.nodes }}</strong><span>手工节点</span></div>
                <div><strong>{{ orchestration.subscriptions }}</strong><span>订阅</span></div>
                <div><strong>{{ orchestration.groups.length }}</strong><span>分组</span></div>
                <div><strong>{{ orchestration.rules }}</strong><span>路由规则</span></div>
              </div>
              <NSpace v-if="orchestration.groups.length" size="small" wrap>
                <NTag v-for="group in orchestration.groups" :key="group.name" size="small" :bordered="false" type="info">
                  {{ group.name }} · {{ group.policy?.value || 'min_moving_avg' }}
                </NTag>
              </NSpace>
            </template>
            <NText v-else-if="!configError" depth="3">入口配置尚不存在，可以在代理编排页从零创建。</NText>
          </template>
        </NCard>
      </NGridItem>

      <NGridItem>
        <NCard title="dae 能力" class="panel-card">
          <template #header-extra>
            <NTag size="small" :type="dae?.available ? 'success' : 'error'">{{ dae?.available ? '已发现' : '不可用' }}</NTag>
          </template>
          <dl class="details-list">
            <div><dt>版本</dt><dd>{{ dae?.version || '—' }}</dd></div>
            <div><dt>二进制</dt><dd class="mono">{{ dae?.binary || '—' }}</dd></div>
            <div><dt>配置结构</dt><dd>{{ dae?.outlineSupported ? `支持（${dae.outlineVersion || '未知版本'}）` : '不支持' }}</dd></div>
          </dl>
          <NSpace v-if="supportedCommands.length" size="small" wrap>
            <NTag v-for="command in supportedCommands" :key="command" size="small" :bordered="false">{{ command }}</NTag>
          </NSpace>
          <NAlert v-if="dae?.problem" type="warning" :bordered="false" class="inline-alert">{{ dae.problem }}</NAlert>
        </NCard>
      </NGridItem>
    </NGrid>
  </div>
</template>
