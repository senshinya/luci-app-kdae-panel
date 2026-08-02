// Package logfmt 解析 dae 输出的 logfmt 日志行。
package logfmt

import (
	"strconv"
	"strings"
)

// Parse 返回一行日志中的 key=value 字段。值可使用 Go 风格双引号转义；
// 同名键保留第一次出现的值，避免日志内容覆盖 dae 先写入的固定字段。
func Parse(line string) (map[string]string, bool) {
	fields := make(map[string]string)
	for offset := 0; ; {
		offset = skipSpace(line, offset)
		if offset == len(line) {
			break
		}

		nameStart := offset
		for offset < len(line) && line[offset] != '=' && !isSpace(line[offset]) {
			offset++
		}
		if offset == nameStart || offset == len(line) || line[offset] != '=' {
			break
		}
		name := line[nameStart:offset]
		offset++

		value, next, ok := parseValue(line, offset)
		if !ok {
			break
		}
		if _, exists := fields[name]; !exists {
			fields[name] = value
		}
		offset = next
	}
	if len(fields) == 0 {
		return nil, false
	}
	return fields, true
}

func parseValue(line string, offset int) (string, int, bool) {
	if offset >= len(line) || line[offset] != '"' {
		end := offset
		for end < len(line) && !isSpace(line[end]) {
			end++
		}
		return line[offset:end], end, true
	}

	for end := offset + 1; end < len(line); end++ {
		switch line[end] {
		case '\\':
			end++
		case '"':
			value, err := strconv.Unquote(line[offset : end+1])
			if err != nil || end+1 < len(line) && !isSpace(line[end+1]) {
				return "", offset, false
			}
			return value, end + 1, true
		}
	}
	return "", offset, false
}

func skipSpace(value string, offset int) int {
	for offset < len(value) && isSpace(value[offset]) {
		offset++
	}
	return offset
}

func isSpace(value byte) bool {
	return strings.ContainsRune(" \t\r\n", rune(value))
}
