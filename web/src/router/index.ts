import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/setup',
      name: 'setup',
      component: () => import('../views/SetupView.vue'),
      meta: { public: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('../layouts/AppLayout.vue'),
      children: [
        { path: '', name: 'dashboard', component: () => import('../views/DashboardView.vue'), meta: { title: '运行概览' } },
        { path: 'connections', name: 'connections', component: () => import('../views/ConnectionsView.vue'), meta: { title: '连接活动' } },
        { path: 'proxy', name: 'orchestration', component: () => import('../views/OrchestrationView.vue'), meta: { title: '代理编排' } },
        { path: 'config', name: 'config', component: () => import('../views/ConfigView.vue'), meta: { title: '配置管理' } },
        { path: 'schema', name: 'schema', component: () => import('../views/SchemaView.vue'), meta: { title: '配置能力' } },
        { path: 'versions', name: 'versions', component: () => import('../views/VersionsView.vue'), meta: { title: 'dae 版本' } },
        { path: 'geo', name: 'geo', component: () => import('../views/GeoView.vue'), meta: { title: 'Geo 数据' } },
        { path: 'diagnostics', name: 'diagnostics', component: () => import('../views/DiagnosticsView.vue'), meta: { title: '故障诊断' } },
        { path: 'logs', name: 'logs', component: () => import('../views/LogsView.vue'), meta: { title: '运行日志' } },
        { path: 'backups', name: 'backups', component: () => import('../views/BackupsView.vue'), meta: { title: '配置备份' } },
        { path: 'settings', name: 'settings', component: () => import('../views/SettingsView.vue'), meta: { title: '面板设置' } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  try {
    await auth.bootstrap()
  } catch {
    if (to.name !== 'login') return { name: 'login' }
    return true
  }

  if (!auth.initialized && to.name !== 'setup') {
    return { name: 'setup' }
  }
  if (auth.initialized && to.name === 'setup') {
    return auth.authenticated ? { name: 'dashboard' } : { name: 'login' }
  }
  if (auth.initialized && !auth.authenticated && !to.meta.public) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (auth.authenticated && (to.name === 'login' || to.name === 'setup')) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
