import { describe, expect, it } from 'vitest'
import { allocateNodeTags, parseNodeLink } from './nodelink'

function base64(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

describe('parseNodeLink', () => {
  it('解析 vmess base64 JSON', () => {
    const payload = base64(JSON.stringify({ add: '1.2.3.4', port: '443', ps: '香港 01' }))
    expect(parseNodeLink(`vmess://${payload}`)).toEqual({
      protocol: 'vmess', name: '香港 01', host: '1.2.3.4', port: 443,
    })
  })

  it('解析 URL 形式的 vless/trojan/tuic/hysteria2', () => {
    expect(parseNodeLink('vless://uuid@example.com:8443?security=reality#节点%20A')).toEqual({
      protocol: 'vless', name: '节点 A', host: 'example.com', port: 8443,
    })
    expect(parseNodeLink('trojan://pass@host.io#T')).toEqual({
      protocol: 'trojan', name: 'T', host: 'host.io', port: 443,
    })
    expect(parseNodeLink('hy2://auth@[2001:db8::1]:9443/?insecure=1#v6')).toEqual({
      protocol: 'hysteria2', name: 'v6', host: '2001:db8::1', port: 9443,
    })
    expect(parseNodeLink('tuic://uuid:pass@t.example:443#tuic')?.host).toBe('t.example')
  })

  it('解析两种 ss 形式', () => {
    expect(parseNodeLink('ss://YWVzOm1pbWk@5.6.7.8:8388#HK')).toEqual({
      protocol: 'ss', name: 'HK', host: '5.6.7.8', port: 8388,
    })
    const whole = btoa('aes-256-gcm:pass@9.9.9.9:443')
    expect(parseNodeLink(`ss://${whole}#SG`)).toEqual({
      protocol: 'ss', name: 'SG', host: '9.9.9.9', port: 443,
    })
  })

  it('解析 ssr 链接', () => {
    const inner = '8.8.4.4:8443:origin:aes-128-cfb:plain:' + base64('pwd') + '/?remarks=' + base64('日本')
    const info = parseNodeLink('ssr://' + base64(inner).replace(/\+/g, '-').replace(/\//g, '_'))
    expect(info).toEqual({ protocol: 'ssr', name: '日本', host: '8.8.4.4', port: 8443 })
  })

  it('非法输入不抛异常', () => {
    expect(parseNodeLink('not a link')).toBeNull()
    expect(parseNodeLink('vmess://%%%')).not.toBeNull()
    expect(parseNodeLink('unknown://a@b:1#x')?.protocol).toBe('unknown')
  })

  it('非整数或越界端口解析为 null,不会发到探测接口', () => {
    for (const port of ['443.5', '0', '70000', 'abc', '']) {
      const link = `vmess://${base64(JSON.stringify({ add: '1.2.3.4', port, ps: 'x' }))}`
      expect(parseNodeLink(link)?.port).toBeNull()
    }
    expect(parseNodeLink(`vmess://${base64(JSON.stringify({ add: '1.2.3.4', port: 443 }))}`)?.port).toBe(443)
  })
})

describe('allocateNodeTags', () => {
  it('从链接备注生成唯一且可由 dae 作为配置键的稳定标签', () => {
    expect(allocateNodeTags([
      'trojan://pass@example.com:443#dmit%20lax%20pro-dmitlaxpro',
      'vless://id@example.net:443#dmit%20lax%20pro-dmitlaxpro',
    ], ['dmit_lax_pro-dmitlaxpro'])).toEqual([
      'dmit_lax_pro-dmitlaxpro_2',
      'dmit_lax_pro-dmitlaxpro_3',
    ])
  })

  it('非 ASCII 备注回退到协议和主机，数字开头时补安全前缀', () => {
    expect(allocateNodeTags([
      'hysteria2://auth@hk.example.com:443#香港',
      'trojan://pass@example.com:443#123',
      'unknown://',
    ])).toEqual(['hysteria2_hk.example.com', 'node_123', 'unknown'])
  })
})
