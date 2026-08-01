package host

import "testing"

// 日志页按 level 字段精确匹配筛选，所以同一条 dae 日志在两个后端下必须得到
// 同一个 level/priority。这个用例把两条解析路径摆在一起比：systemd 侧走
// journald 的 MESSAGE，procd 侧走 logread 的整行，最终结论必须一致。
func TestBothBackendsAgreeOnDaeLogLevels(t *testing.T) {
	cases := []struct {
		daeLevel     string
		wantPriority int
		wantLevel    string
	}{
		{"panic", 2, "critical"},
		{"fatal", 2, "critical"},
		{"error", 3, "error"},
		{"warn", 4, "warning"},
		{"info", 6, "info"},
		{"debug", 7, "debug"},
		{"trace", 7, "trace"},
	}
	for _, testCase := range cases {
		message := `level=` + testCase.daeLevel + ` msg="节点不可达"`

		// systemd：journald 把 dae 的 stdout 一律记成 info（PRIORITY=6），
		// 正文里的 level 才是真的。
		priority, level := logLevel(message, 6)
		if priority != testCase.wantPriority || level != testCase.wantLevel {
			t.Errorf("systemd 侧 %s = %s/%d，期望 %s/%d",
				testCase.daeLevel, level, priority, testCase.wantLevel, testCase.wantPriority)
		}

		// procd：logread 前缀里的 daemon.info 同样不可信，正文优先。
		line := "Fri Jul 31 01:02:03 2026 daemon.info dae[4321]: " + message
		entry, ok := parseLogreadLine(line, "dae")
		if !ok {
			t.Fatalf("procd 侧未能解析 %q", line)
		}
		if entry.Priority != testCase.wantPriority || entry.Level != testCase.wantLevel {
			t.Errorf("procd 侧 %s = %s/%d，期望 %s/%d",
				testCase.daeLevel, entry.Level, entry.Priority, testCase.wantLevel, testCase.wantPriority)
		}
	}
}

// 认不出的级别名不能被硬塞成某一档：systemd 退回 journald 的 PRIORITY，
// procd 退回 logread 前缀里的 facility.level。
func TestUnknownLevelFallsBackPerBackend(t *testing.T) {
	message := `level=verbose msg="不认识的级别"`

	priority, level := logLevel(message, 4)
	if priority != 4 || level != "warning" {
		t.Errorf("systemd 侧 = %s/%d，期望退回 journald 的 warning/4", level, priority)
	}

	entry, ok := parseLogreadLine("Fri Jul 31 01:02:03 2026 daemon.err dae[4321]: "+message, "dae")
	if !ok {
		t.Fatal("procd 侧未能解析")
	}
	if entry.Priority != 3 || entry.Level != "error" {
		t.Errorf("procd 侧 = %s/%d，期望退回前缀的 error/3", entry.Level, entry.Priority)
	}
	// 正文认不出级别时整行原样保留，不能只剩 msg 的内容。
	if entry.Message != message {
		t.Errorf("procd 侧消息 = %q，期望保留整行 %q", entry.Message, message)
	}
}
