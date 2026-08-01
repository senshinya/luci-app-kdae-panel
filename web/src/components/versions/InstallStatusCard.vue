<script setup lang="ts">
import { computed } from 'vue'
import { NAlert, NCard, NTag, NText } from 'naive-ui'
import type { InstallProvision, InstallStatus } from '../../types/api'
import { formatDateTime } from '../../utils/format'
import { useBackendStore } from '../../stores/backend'
import { sourceName } from './sources'

// 纯展示卡片：状态由 VersionsView 加载并轮询，这里只负责如实呈现。
const props = defineProps<{
  loading: boolean
  busy: boolean
  status: InstallStatus | null
  provision: InstallProvision | null
}>()

const firstInstall = computed(() => props.provision?.possible === true)
// 首次安装的说明必须与后端实际会做的事一致：procd 下启动脚本随软件包装好，
// 面板从不写它。加载由 VersionsView 触发，这里只读结果。
const backend = useBackendStore()

const installedPlatform = computed(() => {
  if (props.status?.drifted) return '无法确认（文件已被外部替换）'
  return props.status?.managed?.platform || '未知（旧记录或外部安装）'
})
</script>

<template>
  <NCard :title="firstInstall ? '尚未安装 dae' : '当前安装'" class="panel-card">
    <template v-if="provision && !status?.ready">
      <NAlert v-if="provision.possible" type="info" :bordered="false" class="card-alert">
        这台机器上还没有 dae。在下面选一个版本即可完成首次安装：面板会安装
        <code class="mono">{{ provision.binaryPath }}</code>、<template v-if="backend.isProcd">写入 geo
        数据与种子配置（启动脚本由软件包提供，已就位）。</template><template v-else>写入 geo 数据与服务单元
        <code class="mono">{{ provision.unitPath }}</code>。</template>
      </NAlert>
      <NAlert v-else type="error" :bordered="false" class="card-alert">
        <div v-for="blocker in provision.blockers || []" :key="blocker">{{ blocker }}</div>
      </NAlert>
      <ul v-if="provision.possible" class="provision-notes">
        <li v-for="note in provision.notes || []" :key="note">{{ note }}</li>
      </ul>
    </template>
    <!-- 独立成一条，不接在 provision 的 v-else-if 后面：ready 为假时后端
         几乎总会带上 provision，挂成 else 分支等于让真正的故障原因永远不显示 -->
    <NAlert v-if="!busy && status?.problem" type="warning" :bordered="false" class="card-alert">
      {{ status.problem }}
    </NAlert>
    <!-- applying 期间磁盘与账本短暂不一致、暂存回滚点存在都是事务的正常中间态。
         此时只保留页面级进度提示，不能把正常过程渲染成两条故障告警。 -->
    <template v-if="!busy">
      <NAlert
        v-for="warning in status?.warnings || []"
        :key="warning"
        type="warning"
        :bordered="false"
        class="card-alert"
      >
        {{ warning }}
      </NAlert>
    </template>
    <dl v-if="!firstInstall" class="details-list">
      <div>
        <dt>运行版本</dt>
        <dd>
          <template v-if="loading">
            <NText depth="3">读取中…</NText>
          </template>
          <template v-else>
            {{ status?.version || '—' }}
            <NTag v-if="status?.serviceActive" size="tiny" type="success" :bordered="false">运行中</NTag>
            <NTag v-else size="tiny" type="error" :bordered="false">未运行</NTag>
          </template>
        </dd>
      </div>
      <div>
        <dt>可执行文件</dt>
        <dd class="mono">{{ status?.binaryPath || '—' }}</dd>
      </div>
      <div>
        <dt>面板记录</dt>
        <dd>
          <!-- 判据是"有没有记录"，不是"记录里有没有 label"：
               回滚到面板之外装的版本时账本就没有 label，据此说"不是面板装的"是错的 -->
          <template v-if="status?.managed">
            {{ sourceName(status.managed.source) }}
            <template v-if="status.managed.label"> · {{ status.managed.label }}</template>
            <template v-else-if="status.managed.ref"> · {{ status.managed.ref }}</template>
            <NText depth="3">（{{ formatDateTime(status.managed.installedAt) }} 安装）</NText>
          </template>
          <NText v-else depth="3">该二进制不是由面板安装的</NText>
        </dd>
      </div>
      <div>
        <dt>CPU 架构</dt>
        <dd class="mono">{{ status?.architecture || '—' }}</dd>
      </div>
      <div>
        <dt>首选构建</dt>
        <dd class="mono">{{ status?.preferredPlatform || status?.platform || '—' }}</dd>
      </div>
      <div>
        <dt>当前构建</dt>
        <dd class="mono">{{ installedPlatform }}</dd>
      </div>
    </dl>
    <NAlert v-if="!busy && status?.drifted" type="warning" :bordered="false">
      磁盘上的二进制与面板记录不一致，说明它在面板之外被替换过。
    </NAlert>
  </NCard>
</template>
