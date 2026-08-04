import { h, ref } from 'vue'
import { NTag, NText, NTooltip, type MessageApi } from 'naive-ui'
import { postJSON } from '../api/client'
import type { LatencyResult, LatencyTarget } from '../types/api'

/**
 * 节点入口延迟探测（公网 ICMP、内网 TCP，不经过 dae，不验证代理端口或协议可用性）。
 * 从 NodesCard.vue 抽出，供节点表格与后续的节点瓦片共用同一套阈值与文案。
 *
 * 探测的是「主机:端口」这个入口本身，因此调用方只需要提供这两个字段——本地节点用
 * parseNodeLink 的结果，订阅节点/fixed 候选同理，字段名恰好一致。
 */
export interface LatencyProbeSource {
  host: string
  port: number | null
}

const BATCH_SIZE = 64
const FAST_MS = 100
const SLOW_MS = 300

function keyOf(host: string, port: number | null): string {
  return `${host}:${port}`
}

/** 与后端 netprobe.Target.validate 一致，避免把明显非法的目标发出去。 */
export function probeTarget(source: LatencyProbeSource | null | undefined): LatencyTarget | null {
  const host = source?.host
  const port = source?.port
  if (!host || host !== host.trim() || host.length > 253 || /[\s/\\]/.test(host)) return null
  if (typeof port !== 'number' || !Number.isInteger(port) || port < 1 || port > 65535) return null
  return { host, port }
}

export function useLatencyProbe(message: MessageApi) {
  const probing = ref(false)
  const results = ref(new Map<string, LatencyResult>())

  /** 按原始 host/port 直接查表，不经过 probeTarget 校验——未探测过的目标本来就查不到。 */
  function resultOf(source: LatencyProbeSource | null | undefined): LatencyResult | undefined {
    if (!source) return undefined
    return results.value.get(keyOf(source.host, source.port))
  }

  /** 批量探测；分批 64 个发送，逐批把结果发布出去，某一批失败不影响前面已到的结果。 */
  async function probe(sources: Array<LatencyProbeSource | null | undefined>): Promise<void> {
    const targets = new Map<string, LatencyTarget>()
    for (const source of sources) {
      const target = probeTarget(source)
      if (target) targets.set(keyOf(target.host, target.port), target)
    }
    if (targets.size === 0) {
      message.warning('没有可探测的节点(需要能解析出服务器与端口)')
      return
    }
    probing.value = true
    const batch = [...targets.values()]
    const merged = new Map(results.value)
    try {
      for (let start = 0; start < batch.length; start += BATCH_SIZE) {
        const { results: batchResults } = await postJSON<{ results: LatencyResult[] }>('/api/v1/net/latency', {
          targets: batch.slice(start, start + BATCH_SIZE),
        })
        for (const result of batchResults) merged.set(keyOf(result.host, result.port), result)
        // 逐批发布，某一批失败时前面的结果仍然可见
        results.value = new Map(merged)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '延迟探测失败')
    } finally {
      probing.value = false
    }
  }

  /** 「未测 / 无法测量 / 不可达 / N ms」文案；目标本身无法测量时给出占位符。 */
  function label(source: LatencyProbeSource | null | undefined): string {
    if (!source || !probeTarget(source)) return '—'
    const result = resultOf(source)
    if (!result) return '未测'
    if (!result.reachable) return result.method === 'icmp' ? '无法测量' : '不可达'
    const value = result.latencyMs || 0
    return `${value.toFixed(value < 10 ? 1 : 0)} ms`
  }

  /** 与 label 对应的 NTag 类型，阈值沿用 100ms / 300ms。 */
  function type(source: LatencyProbeSource | null | undefined): 'success' | 'warning' | 'error' | 'default' {
    if (!source || !probeTarget(source)) return 'default'
    const result = resultOf(source)
    if (!result) return 'default'
    if (!result.reachable) return 'error'
    const value = result.latencyMs || 0
    return value < FAST_MS ? 'success' : value < SLOW_MS ? 'warning' : 'error'
  }

  /** tooltip/title 说明文字：不可达给出错误原因，可达给出方法与解析 IP。 */
  function title(source: LatencyProbeSource | null | undefined): string {
    if (!source) return ''
    const result = resultOf(source)
    if (!result) return ''
    if (!result.reachable) return result.error || ''
    const method = result.method === 'icmp' ? 'ICMP 网络延迟' : 'TCP 握手延迟'
    return result.resolvedIp ? `${method} · ${result.resolvedIp}` : method
  }

  /** 桌面表格用的延迟单元格：NTag 包一层 NTooltip 说明这是入口延迟而非 dae 健康检查。 */
  function cell(source: LatencyProbeSource | null | undefined) {
    if (!source || !probeTarget(source)) return h(NText, { depth: 3 }, { default: () => '—' })
    const result = resultOf(source)
    if (!result) return h(NText, { depth: 3 }, { default: () => '未测' })
    if (!result.reachable) {
      const errorLabel = result.method === 'icmp' ? '无法测量' : '不可达'
      return h(NTooltip, null, {
        trigger: () => h(NTag, { size: 'small', type: 'error', bordered: false }, { default: () => errorLabel }),
        default: () => result.error || '连接失败',
      })
    }
    const value = result.latencyMs || 0
    const valueType = value < FAST_MS ? 'success' : value < SLOW_MS ? 'warning' : 'error'
    const tag = () => h(NTag, { size: 'small', type: valueType, bordered: false }, {
      default: () => `${value.toFixed(value < 10 ? 1 : 0)} ms`,
    })
    return h(NTooltip, null, {
      trigger: tag,
      default: () => [
        result.method === 'icmp' ? 'ICMP 网络延迟（不验证代理端口或协议）' : 'TCP 握手延迟',
        result.resolvedIp ? ` · ${result.resolvedIp}` : '',
      ].join(''),
    })
  }

  return { probing, probe, resultOf, label, type, title, cell }
}
