package app

import (
	"context"
	"fmt"

	"github.com/tuoro/kdae-panel/internal/configstore"
)

type pidReloader interface {
	ReloadPID(ctx context.Context, pid int) error
}

type configValidator interface {
	Validate(ctx context.Context, configPath string) error
}

// pidReloadDaeService 让所有面板入口都用主进程号重载 dae。
// 原始 DaeService 仍负责探测、校验、暂停与诊断，只有 Reload 被这里覆盖。
//
// 主进程号取自 host.Manager，systemd 与 procd 都填，因此这条重载路径与后端无关，
// 名字里也就不该出现 systemd——在 OpenWrt 上它走的是 procd。
type pidReloadDaeService struct {
	DaeService
	reloader  pidReloader
	validator configValidator
	host      HostService
}

func newPIDReloadDaeService(
	service DaeService,
	reloader pidReloader,
	validator configValidator,
	host HostService,
) *pidReloadDaeService {
	return &pidReloadDaeService{
		DaeService: service,
		reloader:   reloader,
		validator:  validator,
		host:       host,
	}
}

func (s *pidReloadDaeService) Validate(ctx context.Context, configPath string) error {
	return s.validator.Validate(ctx, configPath)
}

func (s *pidReloadDaeService) Reload(ctx context.Context) error {
	status, err := s.host.Status(ctx)
	if err != nil {
		return fmt.Errorf("读取 dae 服务状态后再重载: %w", err)
	}
	if status.ActiveState == "inactive" || status.ActiveState == "failed" {
		return configstore.ErrReloadDeferred
	}
	if status.ActiveState != "active" {
		return fmt.Errorf("dae 服务状态为 %s/%s，暂时不能重载", status.ActiveState, status.SubState)
	}
	if status.MainPID <= 0 {
		return fmt.Errorf("dae 正在运行但主进程号无效: %d", status.MainPID)
	}
	return s.reloader.ReloadPID(ctx, status.MainPID)
}
