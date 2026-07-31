export interface User {
  id: number
  username: string
  createdAt: string
}

export interface AuthStatus {
  initialized: boolean
  authenticated: boolean
  user?: User
  csrfToken?: string
  expiresAt?: string
  bootstrapRequired?: boolean
  bootstrapAuthorized?: boolean
}

export interface DaeReport {
  binary: string
  available: boolean
  version?: string
  commands: Record<string, boolean>
  outlineSupported: boolean
  outlineVersion?: string
  problem?: string
  detectedAt: string
}

export interface OutlineElement {
  name?: string
  mapping?: string
  isArray?: boolean
  defaultValue?: string
  required?: boolean
  type?: string
  desc?: string
  structure?: OutlineElement[]
}

export interface DaeOutline {
  version: string
  leaves: string[]
  structure: OutlineElement[]
}

export interface ServiceStatus {
  name: string
  description?: string
  loadState?: string
  activeState?: string
  subState?: string
  unitFileState?: string
  mainPid?: number
  execMainStatus?: number
  activeSince?: string
  startedAt?: string
  memoryBytes?: number
  cpuUsageNanoseconds?: number
  tasks?: number
  restarts?: number
  unitPath?: string
}

export interface NetworkInterface {
  name: string
  addresses?: string[]
}

export interface ConfigDocument {
  path: string
  content: string
  hash: string
  size: number
  mode: string
  modifiedAt: string
}

export interface ConfigSaveResult {
  hash: string
  backupId?: string
  applied: boolean
  savedAt: string
  rolledBack: boolean
}

export interface ConfigBackup {
  id: string
  hash: string
  size: number
  createdAt: string
  sourcePath: string
}

export type UpstreamSource = 'official' | 'kdae'

export interface UpstreamVersion {
  source: UpstreamSource
  ref: string
  label: string
  description?: string
  publishedAt: string
  prerelease?: boolean
  installable: boolean
  note?: string
  expiresAt?: string
  cached?: boolean
  cachedOnly?: boolean
  cachedAt?: string
  cachedBytes?: number
}

export interface InstalledState {
  source?: UpstreamSource
  ref?: string
  label?: string
  version?: string
  installedAt?: string
  sha256?: string
}

export interface InstallStatus {
  binaryPath?: string
  platform: string
  ready: boolean
  present: boolean
  version?: string
  managed?: InstalledState
  drifted?: boolean
  rollbackAvailable: boolean
  serviceActive: boolean
  warnings?: string[]
  problem?: string
}

export interface InstallProvision {
  possible: boolean
  installed: boolean
  binaryPath: string
  configPath: string
  unitPath: string
  blockers?: string[]
  notes?: string[]
}

export type InstallPhase = 'idle' | 'downloading' | 'applying' | 'done' | 'failed'

export interface InstallJob {
  phase: InstallPhase
  source?: string
  ref?: string
  label?: string
  cached?: boolean
  startedAt?: string
  endedAt?: string
  error?: string
}

export interface GeoFile {
  name: string
  path?: string
  present: boolean
  size?: number
  modTime?: string
  /** 被 path 遮蔽的同名副本；dae 只读优先级最高的那一份。 */
  shadowed?: string[]
}

export type GeoSource = 'loyalsoldier' | 'v2fly'

export interface GeoSourceInfo {
  source: GeoSource
  label: string
  /** 如实列出全部信任根；同一来源可能横跨多个仓库。 */
  repositories: string[]
  note: string
}

export interface GeoState {
  source: GeoSource
  repositories?: string[]
  tag: string
  updatedAt: string
}

export interface GeoStatus {
  sources: GeoSourceInfo[]
  /** 界面该预选的来源：用过就沿用上次那个，否则用内置默认。 */
  defaultSource: GeoSource
  targetDir: string
  searchPath: string[]
  files: GeoFile[]
  updatable: boolean
  problem?: string
  managed?: GeoState
  warnings?: string[]
}

/** 面板自身的新版本检查结果；dev 构建或检查被关闭时 latest 为空。 */
export interface PanelUpdateCheck {
  current: string
  latest?: string
  updateAvailable: boolean
  checkedAt: string
  error?: string
}

/** 自升级开关与可行性；正式部署始终返回，关闭时仍可从界面重新启用。 */
export interface PanelUpdateStatus {
  current: string
  binaryPath: string
  platform: string
  enabled: boolean
  updatable: boolean
  problem?: string
  previousPath?: string
}

export interface PanelUpdatePayload {
  check: PanelUpdateCheck
  status?: PanelUpdateStatus
  job?: InstallJob
}

/** 定时任务（订阅自动刷新 / geo 自动更新）的设置与执行状态，两个端点同构。 */
export interface ScheduleStatus {
  enabled: boolean
  intervalMinutes: number
  lastRunAt?: string
  lastError?: string
  nextRunAt?: string
}

export interface LatencyTarget {
  host: string
  port: number
}

export interface LatencyResult {
  host: string
  port: number
  reachable: boolean
  latencyMs?: number
  error?: string
}

export interface LogEntry {
  timestamp: string
  priority: number
  level: string
  message: string
  unit?: string
  pid?: string
}

/** GET /api/v1/health。backend 指明面板正跑在哪套 init 系统上。 */
export interface HealthStatus {
  status: string
  version?: string
  backend?: string
}
