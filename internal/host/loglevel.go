package host

import "strings"

// canonicalLevel 把日志里出现的级别名归一成面板对外的那一套名字与 syslog 优先级。
//
// 两个后端必须共用这张表。日志页按 level 字段精确匹配做筛选（"错误"只收
// level == "error"），所以同一条 dae 日志在两种部署下必须落进同一个桶——systemd
// 报 "warning" 而 procd 照 syslog 前缀报 "warn"，用户就会发现同一台机器换个
// 部署方式后筛选结果对不上。名字以 systemd 侧原有的写法为准。
//
// 第三个返回值为 false 表示这个名字不认识，调用方应保留自己那条回退路径
// （journald 的 PRIORITY，或 logread 前缀里的 facility.level），而不是编一个级别。
func canonicalLevel(value string) (int, string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "emerg", "emergency":
		return 0, "emerg", true
	case "alert":
		return 1, "alert", true
	// logrus 的 panic/fatal 与 syslog 的 crit 都是"进程活不下去了"这一档。
	case "panic", "fatal", "crit", "critical":
		return 2, "critical", true
	case "err", "error":
		return 3, "error", true
	case "warn", "warning":
		return 4, "warning", true
	case "notice":
		return 5, "notice", true
	case "info":
		return 6, "info", true
	case "debug":
		return 7, "debug", true
	// syslog 没有比 debug 更细的档，trace 只能与 debug 同级；名字保留，
	// 日志页的"跟踪"筛选正是按名字匹配的。
	case "trace":
		return 7, "trace", true
	default:
		return -1, "", false
	}
}
