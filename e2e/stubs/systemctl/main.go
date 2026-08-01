// E2E 冒烟测试的 systemctl 桩：让面板在无 systemd 的环境里
// 看到一台健康主机。show 之外的子命令（start/stop/reload…）静默成功。
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "show" {
		fmt.Print(`Id=dae.service
Description=dae Service
LoadState=loaded
ActiveState=active
SubState=running
UnitFileState=enabled
MainPID=1487
ExecMainStatus=0
ActiveEnterTimestamp=Fri 2026-07-24 21:18:06 UTC
ExecMainStartTimestamp=Fri 2026-07-24 21:18:05 UTC
MemoryCurrent=87345152
TasksCurrent=17
NRestarts=0
FragmentPath=/etc/systemd/system/dae.service
ExecStart={ path=/usr/local/bin/dae ; argv[]=/usr/local/bin/dae run -c /etc/dae/config.dae ; ignore_errors=no }
Environment=
`)
	}
}
