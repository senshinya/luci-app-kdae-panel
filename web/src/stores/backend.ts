import { defineStore } from 'pinia'
import { getJSON } from '../api/client'
import type { HealthStatus } from '../types/api'

// 面板同一份界面跑在两套 init 系统上，而它们对"面板到底做了什么"的答案不同：
// systemd 部署由面板写入并删除 dae.service，日志来自 journald；procd 部署的
// /etc/init.d/dae 归软件包所有，面板从不写它、卸载也不删它，日志来自系统日志
// 缓冲区。文案照着 systemd 一套写，procd 用户读到的就是对不上的描述——
// 卸载和首次安装这两处尤其要命，它们描述的是不可逆操作。
//
// 后端在一次会话里不会变，因此只取一次。/api/v1/health 是公开端点，
// 未登录页面也能读到。
type Backend = '' | 'systemd' | 'procd'

// inflight 让并发调用共用同一个请求：一个页面里父子组件各调一次很常见，
// 没有它就会对同一个事实发两次查询。
let inflight: Promise<void> | null = null

export const useBackendStore = defineStore('backend', {
  state: () => ({
    backend: '' as Backend,
  }),
  getters: {
    // 取不到时按 systemd 措辞：那是本项目的原生部署，也是所有文案的原始写法。
    // 宁可对 procd 用户少说一句，也不要对 systemd 用户改口。
    isProcd: (state): boolean => state.backend === 'procd',
  },
  actions: {
    async ensure() {
      if (this.backend) return
      if (!inflight) {
        inflight = (async () => {
          try {
            const health = await getJSON<HealthStatus>('/api/v1/health')
            this.backend = health.backend === 'procd' ? 'procd' : 'systemd'
          } catch {
            // 读不到不该挡住页面：文案退回 systemd 说法，下次进页面再试。
          } finally {
            inflight = null
          }
        })()
      }
      await inflight
    },
  },
})
