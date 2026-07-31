// E2E 冒烟测试的 dae 桩：应答面板的能力探测与配置校验。
// validate / reload 等控制子命令一律静默成功——E2E 验证的是面板这一侧
// 的完整链路，dae 本体的行为由上游契约作业对真实二进制验证。
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "--version":
		fmt.Println("dae version v1.0.6")
	case "--help":
		fmt.Print(`Usage:
  dae [command]

Available Commands:
  export      Export dae utilities
  reload      Reload config
  run         Run dae in the foreground
  suspend     Suspend dae
  sysdump     Dump system network config
  validate    Validate dae config
`)
	case "export":
		if len(args) > 1 && args[1] == "outline" {
			fmt.Print(`{"version":"v1.0.6","structure":[{"name":"Global","mapping":"global","structure":[{"name":"LogLevel","mapping":"log_level","defaultValue":"info"},{"name":"LanInterface","mapping":"lan_interface"},{"name":"WanInterface","mapping":"wan_interface"},{"name":"DialMode","mapping":"dial_mode","defaultValue":"domain"},{"name":"TproxyPort","mapping":"tproxy_port","defaultValue":"12345"},{"name":"TlsImplementation","mapping":"tls_implementation","defaultValue":"tls"},{"name":"CheckInterval","mapping":"check_interval","defaultValue":"30s"}]},{"name":"Dns","mapping":"dns","structure":[{"name":"IpVersionPrefer","mapping":"ipversion_prefer"},{"name":"FixedDomainTtl","mapping":"fixed_domain_ttl"},{"name":"Upstream","mapping":"upstream"},{"name":"Routing","mapping":"routing"},{"name":"Bind","mapping":"bind"},{"name":"OptimisticCache","mapping":"optimistic_cache","defaultValue":"true"},{"name":"OptimisticCacheTtl","mapping":"optimistic_cache_ttl","defaultValue":"60"},{"name":"MaxCacheSize","mapping":"max_cache_size","defaultValue":"65536"}]}]}`)
		}
	}
}
