package daeconfig

import (
	"errors"
	"fmt"
	"strings"
)

// ErrGroupPolicyUnlocatable 表示面板无法在不破坏其余内容的前提下定点改写该分组的
// policy：分组不存在、没有 policy、有多处 policy，或者值跨行。调用方应提示改用
// 代理编排页的原文编辑，而不是猜一个位置写下去。
var ErrGroupPolicyUnlocatable = errors.New("无法定位该分组的 policy 声明")

// GroupPolicySpan 返回 group 节中指定分组 policy 值在 content 里的字节区间。
func GroupPolicySpan(content, group string) (int, int, error) {
	masked := maskContent(content)
	groupBody, ok := sectionBody(masked, 0, len(masked), "group")
	if !ok {
		return 0, 0, fmt.Errorf("%w: 配置中没有 group 节", ErrGroupPolicyUnlocatable)
	}
	body, ok := sectionBody(masked, groupBody.start, groupBody.end, group)
	if !ok {
		return 0, 0, fmt.Errorf("%w: 分组 %s 不存在", ErrGroupPolicyUnlocatable, group)
	}

	found := false
	start, end := 0, 0
	for index := body.start; index < body.end; {
		if !isIdentifierStart(masked[index]) {
			index++
			continue
		}
		wordEnd := index + 1
		for wordEnd < body.end && isIdentifierPart(masked[wordEnd]) {
			wordEnd++
		}
		if masked[index:wordEnd] != "policy" {
			index = wordEnd
			continue
		}
		colon := wordEnd
		for colon < body.end && (masked[colon] == ' ' || masked[colon] == '\t') {
			colon++
		}
		if colon >= body.end || masked[colon] != ':' {
			index = wordEnd
			continue
		}
		if found {
			return 0, 0, fmt.Errorf("%w: 分组 %s 有多处 policy 声明", ErrGroupPolicyUnlocatable, group)
		}
		valueStart := colon + 1
		for valueStart < body.end && (masked[valueStart] == ' ' || masked[valueStart] == '\t') {
			valueStart++
		}
		lineEnd := valueStart
		for lineEnd < body.end && masked[lineEnd] != '\n' {
			lineEnd++
		}
		// policy 的值本来就是单 token（如 fixed(0)、min_moving_avg）。右边界只取到第一个
		// 空白为止，其后到行尾必须全是空白（含已被掩码抹掉的行尾注释），否则说明这一行
		// 上还有别的声明（如 `policy: fixed(0) filter: name(a)` 写在同一行），贸然把
		// 剩下的内容也当成 policy 值删掉会连带删除别的声明，属于“定点改写”做不到的情况。
		valueEnd := valueStart
		for valueEnd < lineEnd && masked[valueEnd] != ' ' && masked[valueEnd] != '\t' && masked[valueEnd] != '\r' {
			valueEnd++
		}
		if valueEnd == valueStart {
			return 0, 0, fmt.Errorf("%w: 分组 %s 的 policy 值跨行", ErrGroupPolicyUnlocatable, group)
		}
		trailing := valueEnd
		for trailing < lineEnd && (masked[trailing] == ' ' || masked[trailing] == '\t' || masked[trailing] == '\r') {
			trailing++
		}
		if trailing != lineEnd {
			return 0, 0, fmt.Errorf("%w: 分组 %s 的 policy 声明与其他内容共享一行，无法安全定位", ErrGroupPolicyUnlocatable, group)
		}
		found, start, end = true, valueStart, valueEnd
		index = lineEnd
	}
	if !found {
		return 0, 0, fmt.Errorf("%w: 分组 %s 没有 policy 声明", ErrGroupPolicyUnlocatable, group)
	}
	return start, end, nil
}

// SetGroupPolicy 只替换指定分组 policy 的值，其余字节（含注释与缩进）原样保留。
func SetGroupPolicy(content, group, policy string) (string, error) {
	start, end, err := GroupPolicySpan(content, group)
	if err != nil {
		return "", err
	}
	return content[:start] + policy + content[end:], nil
}

type span struct{ start, end int }

// sectionBody 在 masked[from:to) 这一层里找 `name { ... }`，返回花括号之间的区间。
// 只认同一层（depth 0）的声明，嵌套更深的同名节不会被误选。
func sectionBody(masked string, from, to int, name string) (span, bool) {
	depth := 0
	lastWord, lastWordEnd := "", -1
	for index := from; index < to; {
		char := masked[index]
		switch {
		case isIdentifierStart(char):
			end := index + 1
			for end < to && isIdentifierPart(masked[end]) {
				end++
			}
			lastWord, lastWordEnd = masked[index:end], end
			index = end
		case char == '{':
			if depth == 0 && lastWord == name && lastWordEnd >= 0 && strings.TrimSpace(masked[lastWordEnd:index]) == "" {
				closing, ok := matchBrace(masked, index, to)
				if !ok {
					return span{}, false
				}
				return span{start: index + 1, end: closing}, true
			}
			depth++
			lastWord, lastWordEnd = "", -1
			index++
		case char == '}':
			if depth > 0 {
				depth--
			}
			lastWord, lastWordEnd = "", -1
			index++
		default:
			if !isSpace(char) {
				lastWord, lastWordEnd = "", -1
			}
			index++
		}
	}
	return span{}, false
}

// matchBrace 返回与 masked[open] 配对的 '}' 下标。
func matchBrace(masked string, open, to int) (int, bool) {
	depth := 0
	for index := open; index < to; index++ {
		switch masked[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}
