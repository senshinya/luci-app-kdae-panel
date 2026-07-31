package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newRunner 用测试默认项构造,让既有用例不关心 Options 的新字段。
func newRunner(path string, task Task) (*Runner, error) {
	return New(Options{Path: path, Name: "订阅自动刷新", Task: task, Logger: discardLogger()})
}

func newTestRunner(t *testing.T, task Task) (*Runner, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schedule.json")
	runner, err := newRunner(path, task)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runner.Close() })
	return runner, path
}

// fireSoon 直接改内部状态让下一轮立即到期，避免测试等待真实间隔。
func fireSoon(t *testing.T, runner *Runner) {
	t.Helper()
	runner.mu.Lock()
	runner.settings = Settings{Enabled: true, IntervalMinutes: MinIntervalMinutes}
	runner.nextRunAt = runner.now().Add(20 * time.Millisecond)
	runner.mu.Unlock()
	select {
	case runner.reset <- struct{}{}:
	default:
	}
}

func TestDefaultsAndValidation(t *testing.T) {
	runner, _ := newTestRunner(t, func(context.Context) error { return nil })
	status := runner.Status()
	if status.Enabled || status.IntervalMinutes != defaultInterval || status.NextRunAt != nil {
		t.Fatalf("默认状态异常: %+v", status)
	}
	for _, interval := range []int{0, MinIntervalMinutes - 1, MaxIntervalMinutes + 1} {
		_, err := runner.Update(Settings{Enabled: true, IntervalMinutes: interval})
		var invalid *InvalidSettingsError
		if !errors.As(err, &invalid) {
			t.Fatalf("间隔 %d 应作为参数错误被拒绝，实际 %v", interval, err)
		}
	}
}

func TestSaveFailureIsNotReportedAsInvalidInput(t *testing.T) {
	// 用一个已存在的目录占住设置文件路径，使写入必定失败
	directory := filepath.Join(t.TempDir(), "schedule.json")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	runner, err := newRunner(directory, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()

	_, err = runner.Update(Settings{Enabled: true, IntervalMinutes: 60})
	if err == nil {
		t.Fatal("写入失败时应返回错误")
	}
	var invalid *InvalidSettingsError
	if errors.As(err, &invalid) {
		t.Fatalf("写入失败不应被归为参数错误: %v", err)
	}
	if status := runner.Status(); status.Enabled {
		t.Fatalf("写入失败后应回滚内存设置: %+v", status)
	}
}

func TestUpdatePersistsAndReloads(t *testing.T) {
	runner, path := newTestRunner(t, func(context.Context) error { return nil })
	status, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 90})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || status.IntervalMinutes != 90 || status.NextRunAt == nil {
		t.Fatalf("更新后状态异常: %+v", status)
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}

	restored, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if got := restored.Status(); !got.Enabled || got.IntervalMinutes != 90 {
		t.Fatalf("重启后设置未恢复: %+v", got)
	}
}

func TestZeroTimesAreOmittedFromJSON(t *testing.T) {
	runner, _ := newTestRunner(t, func(context.Context) error { return nil })
	encoded, err := json.Marshal(runner.Status())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"lastRunAt", "nextRunAt", "0001-01-01"} {
		if strings.Contains(string(encoded), field) {
			t.Fatalf("未执行过时不应序列化 %q: %s", field, encoded)
		}
	}
}

// 倒计时必须按"上次执行 + 间隔"接续，否则间隔长于重启周期时永远不会触发。
func TestRestartContinuesCountdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedule.json")
	runner, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	// 用固定的七天而不是 MaxIntervalMinutes：本测试只要求"间隔长于重启周期"，
	// 不该随上限常量的调整而漂移
	if _, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 7 * 24 * 60}); err != nil {
		t.Fatal(err)
	}
	// 模拟六天前执行过一轮并落盘
	runner.mu.Lock()
	runner.lastRunAt = time.Now().Add(-6 * 24 * time.Hour)
	saveErr := runner.save()
	runner.mu.Unlock()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}

	restored, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()

	status := restored.Status()
	if status.LastRunAt == nil {
		t.Fatal("上次执行时间应当被持久化")
	}
	remaining := time.Until(*status.NextRunAt)
	if remaining > 25*time.Hour {
		t.Fatalf("重启不应把倒计时拉满，剩余 %v", remaining)
	}
	if remaining <= 0 {
		t.Fatalf("尚未到期时不应立即执行，剩余 %v", remaining)
	}
}

// 错过的轮次要补做，但至少留出 startupGrace 的缓冲。
func TestMissedRunIsCaughtUpAfterGrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedule.json")
	runner, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 60}); err != nil {
		t.Fatal(err)
	}
	runner.mu.Lock()
	runner.lastRunAt = time.Now().Add(-30 * 24 * time.Hour) // 面板停机一个月
	saveErr := runner.save()
	runner.mu.Unlock()
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}

	restored, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()

	remaining := time.Until(*restored.Status().NextRunAt)
	if remaining <= 0 || remaining > startupGrace+time.Second {
		t.Fatalf("错过的轮次应在 startupGrace 后尽快补做，剩余 %v", remaining)
	}
}

// 被跳过或失败的一轮应尽快重试，而不是推迟一个完整间隔。
func TestFailedRunRetriesSooner(t *testing.T) {
	runner, _ := newTestRunner(t, func(context.Context) error {
		return errors.New("另一个控制操作正在执行，本轮已跳过")
	})
	runner.mu.Lock()
	runner.settings = Settings{Enabled: true, IntervalMinutes: MaxIntervalMinutes}
	runner.mu.Unlock()
	fireSoon(t, runner)
	runner.mu.Lock()
	runner.settings.IntervalMinutes = MaxIntervalMinutes // fireSoon 会改成最小间隔，这里改回长间隔
	runner.mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if status := runner.Status(); status.LastError != "" {
			remaining := time.Until(*status.NextRunAt)
			if remaining > retryInterval+time.Minute {
				t.Fatalf("失败后应尽快重试，实际还要等 %v", remaining)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("定时任务未被执行")
}

// 无变化的保存不应重置倒计时。
func TestNoOpUpdateKeepsCountdown(t *testing.T) {
	runner, _ := newTestRunner(t, func(context.Context) error { return nil })
	if _, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 60}); err != nil {
		t.Fatal(err)
	}
	runner.mu.Lock()
	runner.nextRunAt = time.Now().Add(2 * time.Minute)
	runner.mu.Unlock()

	status, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if remaining := time.Until(*status.NextRunAt); remaining > 3*time.Minute {
		t.Fatalf("同值保存不应重置倒计时，剩余 %v", remaining)
	}
}

func TestCorruptSettingsFallBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedule.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	runner, err := newRunner(path, func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	if status := runner.Status(); status.Enabled || status.IntervalMinutes != defaultInterval {
		t.Fatalf("损坏设置应回退默认值: %+v", status)
	}
}

func TestLoopRunsTaskAndRecordsFailure(t *testing.T) {
	var calls atomic.Int64
	taskErr := errors.New("另一个控制操作正在执行，本轮已跳过")
	runner, _ := newTestRunner(t, func(context.Context) error {
		if calls.Add(1) == 1 {
			return taskErr
		}
		return nil
	})
	fireSoon(t, runner)

	deadline := time.Now().Add(5 * time.Second)
	var status Status
	for time.Now().Before(deadline) {
		status = runner.Status()
		if status.LastError == taskErr.Error() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if calls.Load() < 1 {
		t.Fatal("定时任务未被执行")
	}
	if status.LastError != taskErr.Error() || status.LastRunAt == nil {
		t.Fatalf("失败未被记录: %+v", status)
	}
	if status.NextRunAt == nil {
		t.Fatal("失败后应继续排期下一轮")
	}
}

func TestCloseCancelsInFlightTask(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	runner, _ := newTestRunner(t, func(ctx context.Context) error {
		close(started)
		<-ctx.Done() // 模拟一轮迟迟不返回的 dae reload
		close(finished)
		return ctx.Err()
	})
	fireSoon(t, runner)

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("定时任务未启动")
	}

	closed := make(chan struct{})
	go func() {
		_ = runner.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close 未能及时中断在途任务")
	}
	select {
	case <-finished:
	default:
		t.Fatal("在途任务的 context 应当被取消")
	}
}

func TestDisableStopsScheduling(t *testing.T) {
	var calls atomic.Int64
	runner, _ := newTestRunner(t, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	if _, err := runner.Update(Settings{Enabled: true, IntervalMinutes: 60}); err != nil {
		t.Fatal(err)
	}
	status, err := runner.Update(Settings{Enabled: false, IntervalMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if status.NextRunAt != nil {
		t.Fatalf("停用后不应有下一轮: %+v", status)
	}
	if calls.Load() != 0 {
		t.Fatalf("一小时间隔内不应有执行: %d", calls.Load())
	}
}
