// Package logfmt 解析 dae 输出的 logfmt 行（`level=info msg="…" key=value`）。
//
// 独立成包是因为两处要用同一份实现：host 从中取 level 与 msg 供日志页筛选，
// daeconn 取全部键还原连接活动。两份各自实现过一次，而其中的引号处理错位
// 会被日志内容直接利用——嗅探到的域名、订阅里的节点名都可以含引号。
package logfmt

import (
	"strconv"
	"strings"
)

// Parse 提取一行里的所有 key=value。返回 false 表示这行不是 logfmt。
//
// 值可以带双引号；logrus 对含空格或引号的值用 %q 渲染，值内的引号会转义成
// \"。找闭合引号必须跳过被转义的那些，否则一个来自域名或节点名的引号就能
// 提前收尾，让后面的内容被当成独立字段解析——伪造 outbound、覆盖 msg，
// 或者干脆让整行解析失败从而在界面上隐身。
//
// 同名键取首次出现的值。dae 的固定字段（level、msg）永远排在数据字段之前，
// 因此即便将来出现新的越界形态，伪造片段也盖不掉真实的级别与正文。
func Parse(line string) (map[string]string, bool) {
	fields := make(map[string]string)
	rest := strings.TrimSpace(line)
	for rest != "" {
		name, remainder, found := strings.Cut(rest, "=")
		if !found {
			break
		}
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, " \t") {
			break
		}
		var value string
		var ok bool
		if strings.HasPrefix(remainder, `"`) {
			value, rest, ok = cutQuoted(remainder)
			if !ok {
				break
			}
		} else {
			value, rest, _ = strings.Cut(remainder, " ")
			rest = strings.TrimSpace(rest)
		}
		if _, exists := fields[name]; !exists {
			fields[name] = value
		}
	}
	if len(fields) == 0 {
		return nil, false
	}
	return fields, true
}

// cutQuoted 从 input 的开头切下一个带引号的值，返回解引号后的内容与剩余部分。
// input 必须以 " 开头。引号内的 \" 与 \\ 按 Go 字面量规则处理。
func cutQuoted(input string) (value, rest string, ok bool) {
	for index := 1; index < len(input); index++ {
		switch input[index] {
		case '\\':
			index++ // 跳过被转义的那个字节，它不能作为闭合引号
		case '"':
			unquoted, err := strconv.Unquote(input[:index+1])
			if err != nil {
				return "", "", false
			}
			return unquoted, strings.TrimSpace(input[index+1:]), true
		}
	}
	return "", "", false
}
