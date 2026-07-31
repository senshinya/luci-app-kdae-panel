<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NAvatar,
  NButton,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  NText,
  useMessage,
  type MenuOption,
} from 'naive-ui'
import {
  ArchiveOutline,
  CodeSlashOutline,
  CubeOutline,
  DocumentTextOutline,
  EarthOutline,
  GitNetworkOutline,
  GridOutline,
  LogOutOutline,
  ReaderOutline,
  SettingsOutline,
} from '@vicons/ionicons5'
import { getJSON } from '../api/client'
import type { PanelUpdatePayload, PanelUpdateStatus } from '../types/api'
import PanelUpdateAction from '../components/PanelUpdateAction.vue'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const message = useMessage()
const collapsed = ref(window.innerWidth < 900)

function menuLink(label: string, name: string, icon: typeof GridOutline): MenuOption {
  return {
    label: () => h(RouterLink, { to: { name } }, { default: () => label }),
    key: name,
    icon: () => h(NIcon, null, { default: () => h(icon) }),
  }
}

const menuOptions: MenuOption[] = [
  menuLink('运行概览', 'dashboard', GridOutline),
  menuLink('代理编排', 'orchestration', GitNetworkOutline),
  menuLink('配置管理', 'config', DocumentTextOutline),
  menuLink('配置能力', 'schema', CodeSlashOutline),
  menuLink('dae 版本', 'versions', CubeOutline),
  menuLink('Geo 数据', 'geo', EarthOutline),
  menuLink('运行日志', 'logs', ReaderOutline),
  menuLink('配置备份', 'backups', ArchiveOutline),
  menuLink('面板设置', 'settings', SettingsOutline),
]

const selectedKey = computed(() => String(route.name || 'dashboard'))
const title = computed(() => String(route.meta.title || 'kdae-panel'))

async function logout() {
  try {
    await auth.logout()
    await router.replace({ name: 'login' })
  } catch (error) {
    message.error(error instanceof Error ? error.message : '退出登录失败')
  }
}

function handleExpired() {
  auth.clearSession()
  void router.replace({ name: 'login' })
  message.warning('登录会话已过期，请重新登录')
}

function handleResize() {
  if (window.innerWidth < 900) collapsed.value = true
}

// 新版本提醒：后端带缓存，这里每次进入布局查一次即可。
// 检查失败保持沉默——提醒是锦上添花，不该因为 GitHub 不可达而打扰使用。
const update = ref<PanelUpdatePayload | null>(null)
const updateDismissed = ref(false)

async function checkUpdate() {
  try {
    update.value = await getJSON<PanelUpdatePayload>('/api/v1/panel/update')
  } catch {
    update.value = null
  }
}

function handleSelfUpdateChanged(event: Event) {
  const status = (event as CustomEvent<PanelUpdateStatus>).detail
  if (update.value && status) update.value.status = status
}

onMounted(() => {
  window.addEventListener('kdae-panel:auth-expired', handleExpired)
  window.addEventListener('kdae-panel:self-update-changed', handleSelfUpdateChanged)
  window.addEventListener('resize', handleResize)
  void checkUpdate()
})
onBeforeUnmount(() => {
  window.removeEventListener('kdae-panel:auth-expired', handleExpired)
  window.removeEventListener('kdae-panel:self-update-changed', handleSelfUpdateChanged)
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <NLayout has-sider class="app-shell">
    <NLayoutSider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="236"
      :collapsed="collapsed"
      show-trigger="bar"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="brand" :class="{ compact: collapsed }">
        <div class="brand-mark">K</div>
        <div v-if="!collapsed" class="brand-copy">
          <strong>kdae-panel</strong>
          <span>零侵入管理面板</span>
        </div>
      </div>
      <NMenu :value="selectedKey" :collapsed="collapsed" :collapsed-width="64" :collapsed-icon-size="22" :options="menuOptions" />
    </NLayoutSider>

    <NLayout>
      <NLayoutHeader bordered class="app-header">
        <div>
          <NText depth="3" class="eyebrow">KDAE CONTROL PLANE</NText>
          <h1>{{ title }}</h1>
        </div>
        <div class="account">
          <NAvatar round size="small">{{ auth.user?.username?.slice(0, 1).toUpperCase() }}</NAvatar>
          <div class="account-copy">
            <strong>{{ auth.user?.username }}</strong>
            <span>管理员</span>
          </div>
          <NButton quaternary circle title="退出登录" @click="logout">
            <template #icon><NIcon><LogOutOutline /></NIcon></template>
          </NButton>
        </div>
      </NLayoutHeader>
      <NLayoutContent class="app-content" content-style="padding: 28px;">
        <NAlert
          v-if="update?.check.updateAvailable && !updateDismissed"
          type="info"
          closable
          class="update-banner"
          @close="updateDismissed = true"
        >
          <div class="update-banner-body">
            <span>
              面板有新版本 <strong>{{ update.check.latest }}</strong>（当前 {{ update.check.current }}）。
              <template v-if="update.status?.enabled && update.status.updatable">升级会替换面板二进制并重启自身，配置与账号数据都会保留。</template>
              <template v-else-if="update.status && !update.status.enabled">可直接在这里启用一键升级，不需要 SSH。</template>
              <template v-else-if="update.status?.problem">当前无法一键升级：{{ update.status.problem }}</template>
              <template v-else>当前部署不支持一键升级，可重新执行一键部署命令。</template>
              <a href="https://github.com/tuoro/kdae-panel/releases/latest" target="_blank" rel="noopener">查看发布说明</a>
            </span>
            <PanelUpdateAction :payload="update" label="立即升级" />
          </div>
        </NAlert>
        <RouterView />
      </NLayoutContent>
    </NLayout>
  </NLayout>
</template>
