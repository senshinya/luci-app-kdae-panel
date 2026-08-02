<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NAvatar,
  NButton,
  NDrawer,
  NDrawerContent,
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
  MenuOutline,
  PulseOutline,
  ReaderOutline,
  SettingsOutline,
  SwapHorizontalOutline,
} from '@vicons/ionicons5'
import { getJSON } from '../api/client'
import type { PanelUpdatePayload, PanelUpdateStatus } from '../types/api'
import PanelUpdateAction from '../components/PanelUpdateAction.vue'
import { useMobileViewport } from '../composables/useMobileViewport'
import { useAuthStore } from '../stores/auth'
import { useBackendStore } from '../stores/backend'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const backend = useBackendStore()
const message = useMessage()
const mobile = useMobileViewport()
const drawerVisible = ref(false)
const viewportWidth = ref(window.innerWidth)
const drawerWidth = computed(() => Math.min(320, Math.round(viewportWidth.value * 0.86)))
const collapsed = ref(window.innerWidth < 1100)

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
  menuLink('故障诊断', 'diagnostics', PulseOutline),
  menuLink('连接活动', 'connections', SwapHorizontalOutline),
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
  viewportWidth.value = window.innerWidth
  if (!mobile.value && window.innerWidth < 1100) collapsed.value = true
}

watch(mobile, () => { drawerVisible.value = false })

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
  void backend.ensure()
  void checkUpdate()
})
onBeforeUnmount(() => {
  window.removeEventListener('kdae-panel:auth-expired', handleExpired)
  window.removeEventListener('kdae-panel:self-update-changed', handleSelfUpdateChanged)
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <NLayout :has-sider="!mobile" class="app-shell">
    <NLayoutSider
      v-if="!mobile"
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

    <NDrawer v-model:show="drawerVisible" placement="left" :width="drawerWidth">
      <NDrawerContent class="mobile-nav-drawer" :native-scrollbar="false" body-content-style="padding: 0;">
        <div class="brand mobile-drawer-brand">
          <div class="brand-mark">K</div>
          <div class="brand-copy">
            <strong>kdae-panel</strong>
            <span>零侵入管理面板</span>
          </div>
        </div>
        <NMenu :value="selectedKey" :options="menuOptions" @update:value="drawerVisible = false" />
        <template #footer>
          <div class="mobile-drawer-account">
            <NAvatar round size="small">{{ auth.user?.username?.slice(0, 1).toUpperCase() }}</NAvatar>
            <div class="account-copy">
              <strong>{{ auth.user?.username }}</strong>
              <span>管理员</span>
            </div>
            <NButton quaternary circle title="退出登录" aria-label="退出登录" @click="logout">
              <template #icon><NIcon><LogOutOutline /></NIcon></template>
            </NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- content-class 落到 naive-ui 自己的滚动容器上：桌面端要在这一层把顶栏钉住，
         把滚动交给下面的 .app-content，否则唯一能挂样式的就是 naive-ui 的内部类名 -->
    <NLayout content-class="app-main">
      <NLayoutHeader bordered class="app-header">
        <div class="app-header-leading">
          <NButton
            v-if="mobile"
            quaternary
            circle
            class="mobile-nav-trigger"
            title="打开导航"
            aria-label="打开导航"
            @click="drawerVisible = true"
          >
            <template #icon><NIcon><MenuOutline /></NIcon></template>
          </NButton>
          <div class="app-title">
            <NText depth="3" class="eyebrow">KDAE CONTROL PLANE</NText>
            <h1>{{ title }}</h1>
          </div>
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
      <NLayoutContent class="app-content" content-style="padding: var(--page-padding);">
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
              <!-- status 缺失 = 后端没注册自升级。procd 走 opkg，说"重新执行一键部署命令"
                   会把用户引到一条这台机器上不存在的路径上。 -->
              <template v-else-if="backend.isProcd">
                本部署由 opkg 升级：<code class="mono">opkg update &amp;&amp; opkg install kdae-panel luci-app-kdae-panel</code>。
              </template>
              <template v-else>当前部署不支持一键升级，可重新执行一键部署命令。</template>
              <a v-if="update.check.releasesUrl" :href="update.check.releasesUrl" target="_blank" rel="noopener">查看发布说明</a>
            </span>
            <PanelUpdateAction :payload="update" label="立即升级" />
          </div>
        </NAlert>
        <RouterView />
      </NLayoutContent>
    </NLayout>
  </NLayout>
</template>
