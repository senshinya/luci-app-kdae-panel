// Package host 把"控制 dae 服务、读它的日志"这件事收在一处，
// 对上层只暴露 Manager 一个接口，具体是 systemd 还是 procd 由本包决定。
package host

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tuoro/kdae-panel/internal/command"
)

// Manager 是主机服务管理器，systemd 与 procd 两个后端实现它。
type Manager interface {
	Status(ctx context.Context) (Status, error)
	Action(ctx context.Context, action Action) error
	RestartSelf(ctx context.Context) error
	Logs(ctx context.Context, limit int) ([]LogEntry, error)
	Interfaces(ctx context.Context) ([]NetworkInterface, error)
}

// Backend 指明用哪一套系统接口管理服务。
type Backend string

const (
	// BackendAuto 按机器实际情况二选一。
	BackendAuto    Backend = "auto"
	BackendSystemd Backend = "systemd"
	BackendProcd   Backend = "procd"
)

// procdMarker 是 OpenWrt/ImmortalWrt 的进程管理守护进程。它存在就说明这台
// 机器没有 systemd，自动探测据此二选一。做成变量是给测试留的缝。
var procdMarker = "/sbin/procd"

const defaultServiceName = "dae"

// Resolve 把 auto 落到具体后端；显式指定的原样返回，未知值直接报错。
//
// 不把探测藏在 New 里，是为了让"这台机器被判成了哪个后端"可以被日志、
// 健康检查和测试直接问出来——错配的症状（服务控制全部失败）离原因很远。
func (b Backend) Resolve() (Backend, error) {
	switch b {
	case "", BackendAuto:
		if _, err := os.Stat(procdMarker); err == nil {
			return BackendProcd, nil
		}
		return BackendSystemd, nil
	case BackendSystemd, BackendProcd:
		return b, nil
	default:
		return "", fmt.Errorf("未知的服务后端 %q，可选 auto、systemd、procd", string(b))
	}
}

type Options struct {
	Backend     Backend
	ServiceName string
	// DaeBinary 是 procd 后端 ExecStartPath 的回退值。procd 在服务停止时
	// 拿不到命令行，而调用方要靠这个字段判断"这台机器上有没有 dae"。
	DaeBinary string
	// Systemctl、Journalctl 只有 systemd 后端使用。
	Systemctl  string
	Journalctl string
	Runner     command.Runner
	Timeout    time.Duration
}

// New 按 Options.Backend 构造对应后端。
func New(options Options) (Manager, error) {
	backend, err := options.Backend.Resolve()
	if err != nil {
		return nil, err
	}
	if options.ServiceName == "" {
		options.ServiceName = defaultServiceName
	}
	if !validUnitName(options.ServiceName) {
		return nil, fmt.Errorf("服务名 %q 无效", options.ServiceName)
	}
	if options.Runner == nil {
		options.Runner = command.ExecRunner{}
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	switch backend {
	case BackendProcd:
		return newProcdManager(options)
	default:
		if options.Systemctl == "" {
			options.Systemctl = "systemctl"
		}
		if options.Journalctl == "" {
			options.Journalctl = "journalctl"
		}
		return &systemdManager{
			serviceName: options.ServiceName,
			systemctl:   options.Systemctl,
			journalctl:  options.Journalctl,
			runner:      options.Runner,
			timeout:     options.Timeout,
		}, nil
	}
}
