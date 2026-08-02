import { getJSON, putJSON } from '../api/client'
import type { ConfigDocument, ConfigSaveResult } from '../types/api'
import { applyGlobalChanges, readGlobalState } from './global'

export const DAE_LOG_LEVELS = ['error', 'warn', 'info', 'debug', 'trace'] as const
export type DaeLogLevel = typeof DAE_LOG_LEVELS[number]

export const DAE_LOG_LEVEL_OPTIONS = [
  { label: '错误 · error', value: 'error' },
  { label: '警告 · warn', value: 'warn' },
  { label: '信息 · info', value: 'info' },
  { label: '调试 · debug', value: 'debug' },
  { label: '跟踪 · trace', value: 'trace' },
] satisfies Array<{ label: string; value: DaeLogLevel }>

const levelSet = new Set<string>(DAE_LOG_LEVELS)

/** 读取配置中的实际级别；未声明时采用 dae 的默认值 info。 */
export function readDaeLogLevel(content: string): DaeLogLevel | null {
  const state = readGlobalState(content)
  if (state.duplicateKeys.has('log_level') || state.invalidKeys.has('log_level')) return null
  const value = state.values.log_level
  if (value === null) return 'info'
  return typeof value === 'string' && levelSet.has(value) ? value as DaeLogLevel : null
}

export function replaceDaeLogLevel(content: string, level: DaeLogLevel): string {
  return applyGlobalChanges(content, [{ key: 'log_level', value: level }])
}

export async function loadDaeLogLevel(): Promise<DaeLogLevel | null> {
  const document = await getJSON<ConfigDocument>('/api/v1/config')
  return readDaeLogLevel(document.content)
}

export async function updateDaeLogLevel(level: DaeLogLevel): Promise<{
  changed: boolean
  deferred: boolean
}> {
  const document = await getJSON<ConfigDocument>('/api/v1/config')
  if (readDaeLogLevel(document.content) === level) return { changed: false, deferred: false }
  const content = replaceDaeLogLevel(document.content, level)
  const result = await putJSON<ConfigSaveResult>('/api/v1/config', {
    content,
    expectedHash: document.hash,
    apply: true,
  })
  return { changed: true, deferred: result.deferred === true }
}
