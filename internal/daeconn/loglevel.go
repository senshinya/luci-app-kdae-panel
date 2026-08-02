package daeconn

import "strings"

// LogLevelFromConfig 从 .dae 配置原文里提取 global 的 log_level。
// 找不到返回空串，由调用方回退到 dae 的默认值。只做逐行前缀匹配：
// 配置的完整语法归 dae 管，这里只需要一个"日志级别够不够低"的提示，
// 解析失败的正确行为是不提示，而不是猜。
func LogLevelFromConfig(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if comment := strings.IndexByte(line, '#'); comment >= 0 {
			line = strings.TrimSpace(line[:comment])
		}
		key, value, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(key) != "log_level" {
			continue
		}
		return strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
	}
	return ""
}
