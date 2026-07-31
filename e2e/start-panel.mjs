// 启动被测面板：全新工作目录 + 桩二进制，供 playwright 的 webServer 使用。
// 二进制需要事先构建（见 ci.yml 的 e2e 作业；本地同样四条 go build）。
import { existsSync, mkdirSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const exe = process.platform === 'win32' ? '.exe' : ''
const bin = (name) => join(here, 'bin', name + exe)

for (const name of ['kdae-panel', 'systemctl', 'dae', 'journalctl']) {
  const extensionless = join(here, 'bin', name)
  // Windows 会在创建进程时自动补 .exe。若开发者照抄 Linux/CI 命令生成了更新的
  // 无扩展名文件，继续测试只会悄悄启动旧 .exe，形成假绿色，必须明确拒绝。
  if (exe && existsSync(extensionless) && existsSync(bin(name))
      && statSync(extensionless).mtimeMs > statSync(bin(name)).mtimeMs) {
    console.error(`${extensionless} 比 ${bin(name)} 新；Windows 本地构建目标必须带 .exe`)
    process.exit(1)
  }
  if (!existsSync(bin(name))) {
    console.error(`缺少 ${bin(name)}，请先构建（参照 .github/workflows/ci.yml 的 e2e 作业）`)
    process.exit(1)
  }
}

const work = join(here, '.work')
rmSync(work, { recursive: true, force: true })
mkdirSync(join(work, 'backups'), { recursive: true })

// 最小可校验配置：测试从"几乎为空"起步，导入节点走的正是真实的编排链路
writeFileSync(join(work, 'config.dae'), `global {
    log_level: info
}

node {
}

routing {
    fallback: direct
}
`)

const child = spawn(bin('kdae-panel'), [
  '-listen', '127.0.0.1:21323',
  '-bootstrap-token', 'e2e-bootstrap',
  '-dae-binary', bin('dae'),
  '-systemctl', bin('systemctl'),
  '-journalctl', bin('journalctl'),
  '-dae-config', join(work, 'config.dae'),
  '-backup-dir', join(work, 'backups'),
  '-database', join(work, 'panel.db'),
  '-schedule-file', join(work, 'schedule.json'),
  '-geo-schedule-file', join(work, 'geo-schedule.json'),
  '-geo-sources-file', join(work, 'geo-sources.json'),
  '-install-state-file', join(work, 'install.json'),
  '-github-token-file', join(work, 'github-token'),
  '-panel-backup-file', join(work, 'kdae-panel.previous'),
  '-geo-state-file', join(work, 'geo.json'),
  // E2E 必须完全离线可重复，不向 GitHub 发任何请求
  '-disable-update-check',
], { stdio: 'inherit' })
child.on('exit', (code) => process.exit(code ?? 1))
