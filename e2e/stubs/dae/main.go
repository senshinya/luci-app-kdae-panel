// E2E 冒烟测试的 dae 桩：应答面板的能力探测与配置校验。
// reload 必须携带 systemctl 桩给出的 MainPID，防止配置保存悄悄退回依赖
// /var/run/dae.pid 的无参数调用；dae 本体行为由上游契约作业验证。
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
	case "reload":
		if len(args) != 2 || args[1] != "1487" {
			fmt.Fprintf(os.Stderr, "reload 必须携带 MainPID 1487，实际参数: %v\n", args)
			os.Exit(2)
		}
	}
}
