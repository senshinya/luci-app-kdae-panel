package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/tuoro/kdae-panel/internal/configstore"
	"github.com/tuoro/kdae-panel/internal/host"
)

type stubDaeController struct {
	stubDaeService
	pids          []int
	validatedPath string
	reloadPIDErr  error
}

func (s *stubDaeController) ReloadPID(_ context.Context, pid int) error {
	s.pids = append(s.pids, pid)
	return s.reloadPIDErr
}

func (s *stubDaeController) Validate(_ context.Context, path string) error {
	s.validatedPath = path
	return nil
}

func TestPIDReloadDaeServiceReloadUsesMainPID(t *testing.T) {
	controller := &stubDaeController{stubDaeService: stubDaeService{
		err: errors.New("不应调用无参数 reload"),
	}}
	service := newPIDReloadDaeService(
		controller, controller, controller,
		&stubHostService{status: host.Status{ActiveState: "active", MainPID: 4321}},
	)
	if err := service.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(controller.pids, []int{4321}) {
		t.Fatalf("重载 PID = %v，期望 [4321]", controller.pids)
	}
	if err := service.Validate(context.Background(), "/tmp/candidate.dae"); err != nil {
		t.Fatal(err)
	}
	if controller.validatedPath != "/tmp/candidate.dae" {
		t.Fatalf("校验路径 = %q", controller.validatedPath)
	}
}

func TestPIDReloadDaeServiceDefersWhenInactive(t *testing.T) {
	controller := &stubDaeController{}
	service := newPIDReloadDaeService(
		controller, controller, controller,
		&stubHostService{status: host.Status{ActiveState: "inactive"}},
	)
	if err := service.Reload(context.Background()); !errors.Is(err, configstore.ErrReloadDeferred) {
		t.Fatalf("错误 = %v，期望 ErrReloadDeferred", err)
	}
	if len(controller.pids) != 0 {
		t.Fatalf("停止状态不应调用 reload: %v", controller.pids)
	}
}

func TestPIDReloadDaeServiceRejectsUnstableStateAndInvalidPID(t *testing.T) {
	controller := &stubDaeController{}
	for _, status := range []host.Status{
		{ActiveState: "activating", SubState: "start-pre"},
		{ActiveState: "active", SubState: "running", MainPID: 0},
	} {
		service := newPIDReloadDaeService(controller, controller, controller, &stubHostService{status: status})
		if err := service.Reload(context.Background()); err == nil {
			t.Fatalf("状态 %+v 应拒绝重载", status)
		}
	}
}
