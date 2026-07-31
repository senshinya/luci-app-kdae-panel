// Package daediag 把 dae 运行期日志里的常见根因转换成面板可操作的提示。
package daediag

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/tuoro/kdae-panel/internal/host"
)

// ExplainGeoFailure 在近期日志里发现 Geo 分类缺失时补充明确说明。
// dae validate 不加载 Geo 数据，版本切换的预检因此无法提前发现这类错误。
func ExplainGeoFailure(cause error, entries []host.LogEntry, since time.Time) error {
	if cause == nil {
		cause = errors.New("dae 启动失败")
	}
	if classification, line := missingGeoLine(cause.Error()); line != "" {
		return explainMissingGeo(cause, classification, line)
	}
	classification, line := missingGeoClassification(entries, since)
	if line == "" {
		return cause
	}
	return explainMissingGeo(cause, classification, line)
}

// ExplainGeoError 检查 dae 命令本身返回的错误。reload 会把运行期的 Geo 分类
// 错误直接写到输出里，不必再绕到 journald 才能给出可操作提示。
func ExplainGeoError(cause error) error {
	if cause == nil {
		cause = errors.New("dae 操作失败")
	}
	classification, line := missingGeoLine(cause.Error())
	if line == "" {
		return cause
	}
	return explainMissingGeo(cause, classification, line)
}

func explainMissingGeo(cause error, classification, line string) error {
	return fmt.Errorf("%w；配置引用了当前 Geo 数据库不存在的分类 %s（dae 日志：%s）。"+
		"请在“Geo 数据”页更新或切换来源，或修改路由规则；dae validate 不会检查分类是否实际存在",
		cause, classification, line)
}

func missingGeoClassification(entries []host.LogEntry, since time.Time) (string, string) {
	for index := len(entries) - 1; index >= 0; index-- {
		entry := entries[index]
		if !since.IsZero() && !entry.Timestamp.IsZero() && entry.Timestamp.Before(since.Add(-2*time.Second)) {
			continue
		}
		if classification, line := missingGeoLine(entry.Message); line != "" {
			return classification, line
		}
	}
	return "", ""
}

func missingGeoLine(message string) (string, string) {
	line := strings.TrimSpace(message)
	lower := strings.ToLower(line)
	marker := strings.Index(lower, " not found in ")
	if marker < 0 {
		return "", ""
	}
	kind := ""
	switch {
	case strings.Contains(lower[marker:], "geoip.dat"):
		kind = "geoip"
	case strings.Contains(lower[marker:], "geosite.dat") || strings.Contains(lower[marker:], "dlc.dat"):
		kind = "geosite"
	default:
		return "", ""
	}
	before := strings.Fields(line[:marker])
	code := "未知"
	if len(before) > 0 {
		code = strings.TrimFunc(before[len(before)-1], func(character rune) bool {
			return unicode.IsPunct(character) || unicode.IsSpace(character)
		})
		if code == "" {
			code = "未知"
		}
	}
	runes := []rune(line)
	if len(runes) > 400 {
		line = string(runes[:400]) + "…"
	}
	return kind + ":" + code, line
}
