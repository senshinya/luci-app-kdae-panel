import { describe, expect, it } from 'vitest'
import { readDaeLogLevel, replaceDaeLogLevel } from './loglevel'

describe('dae 日志级别配置', () => {
  it('读取显式值与默认值', () => {
    expect(readDaeLogLevel('global {\n  log_level: warn\n}\n')).toBe('warn')
    expect(readDaeLogLevel('global {\n  lan_interface: auto\n}\n')).toBe('info')
  })

  it('拒绝歧义值', () => {
    expect(readDaeLogLevel('global {\n  log_level: info\n  log_level: debug\n}\n')).toBeNull()
    expect(readDaeLogLevel('global {\n  log_level: verbose\n}\n')).toBeNull()
  })

  it('只改目标声明并保留注释', () => {
    const original = 'global {\n  log_level: warn # 运维备注\n  lan_interface: auto\n}\n'
    expect(replaceDaeLogLevel(original, 'info')).toBe(
      'global {\n  log_level: info # 运维备注\n  lan_interface: auto\n}\n',
    )
  })
})
