export function formatBytes(value?: number): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  if (value < 1024) return `${value} B`
  const units = ['KiB', 'MiB', 'GiB', 'TiB']
  let current = value / 1024
  let unit = units[0]
  for (let index = 1; index < units.length && current >= 1024; index += 1) {
    current /= 1024
    unit = units[index]
  }
  return `${current >= 100 ? current.toFixed(0) : current.toFixed(1)} ${unit}`
}

export function formatDurationNanoseconds(value?: number): string {
  if (value === undefined || !Number.isFinite(value)) return '—'
  const seconds = Math.floor(value / 1_000_000_000)
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remaining = seconds % 60
  if (hours > 0) return `${hours} 小时 ${minutes} 分`
  if (minutes > 0) return `${minutes} 分 ${remaining} 秒`
  return `${remaining} 秒`
}

// formatUptime 把"已运行秒数"渲染成人读的时长。
//
// 与 formatDurationNanoseconds 分开：那个渲染的是累计 CPU 时间，几小时就算多；
// 运行时长动辄以天计，套用同一套单位会出现"8760 小时 0 分"这种没法一眼读的值。
export function formatUptime(value?: number): string {
  if (value === undefined || !Number.isFinite(value) || value < 0) return '—'
  const seconds = Math.floor(value)
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时`
  if (hours > 0) return `${hours} 小时 ${minutes} 分`
  if (minutes > 0) return `${minutes} 分`
  return `${seconds} 秒`
}

export function formatDateTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(date)
}

export function shortHash(value?: string): string {
  return value ? value.slice(0, 12) : '—'
}

