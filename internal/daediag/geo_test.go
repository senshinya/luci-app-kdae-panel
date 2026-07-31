package daediag

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tuoro/kdae-panel/internal/host"
)

func TestExplainGeoFailureNamesMissingClassification(t *testing.T) {
	started := time.Now()
	err := ExplainGeoFailure(errors.New("执行 systemd restart 失败"), []host.LogEntry{{
		Timestamp: started,
		Message:   "country code twitter not found in /etc/dae/geoip.dat",
	}}, started)
	message := err.Error()
	for _, want := range []string{"geoip:twitter", "Geo 数据", "dae validate"} {
		if !strings.Contains(message, want) {
			t.Fatalf("提示 %q 应包含 %q", message, want)
		}
	}
}

func TestExplainGeoFailureIgnoresStaleAndUnrelatedLogs(t *testing.T) {
	cause := errors.New("启动失败")
	started := time.Now()
	entries := []host.LogEntry{
		{Timestamp: started.Add(-time.Minute), Message: "country code old not found in /etc/dae/geoip.dat"},
		{Timestamp: started, Message: "permission denied"},
	}
	if got := ExplainGeoFailure(cause, entries, started); got.Error() != cause.Error() {
		t.Fatalf("不相关日志不应改写错误：%v", got)
	}
}

func TestExplainGeoErrorReadsReloadOutput(t *testing.T) {
	err := ExplainGeoError(errors.New(
		"dae 重载失败: code geolocation-!cn not found in /etc/dae/geosite.dat"))
	if message := err.Error(); !strings.Contains(message, "geosite:geolocation-!cn") ||
		!strings.Contains(message, "Geo 数据") {
		t.Fatalf("reload 错误应直接指出缺失分类：%v", err)
	}
}
