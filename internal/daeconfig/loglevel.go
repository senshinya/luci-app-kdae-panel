// Package daeconfig 提供不依赖 dae 内部包的轻量配置读取能力。
package daeconfig

import "strings"

// LogLevel 返回 global.log_level 的有效值。字段未声明时返回 dae 的默认值 info；
// 声明存在但无法识别时返回空字符串，调用方据此避免展示错误状态。
func LogLevel(content string) string {
	tokens := tokenize(content)
	depth, globalDepth := 0, -1
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if depth == 0 && token == "global" && index+1 < len(tokens) && tokens[index+1] == "{" {
			globalDepth = 1
		}
		if globalDepth == depth && token == "log_level" && index+2 < len(tokens) && tokens[index+1] == ":" {
			if level := tokens[index+2]; validLogLevel(level) {
				return level
			}
			return ""
		}
		switch token {
		case "{":
			depth++
		case "}":
			if depth == globalDepth {
				globalDepth = -1
			}
			if depth > 0 {
				depth--
			}
		}
	}
	return "info"
}

func validLogLevel(value string) bool {
	switch value {
	case "error", "warn", "info", "debug", "trace":
		return true
	default:
		return false
	}
}

// tokenize 只保留定位 global 声明需要的标识符和结构符号；注释与字符串内容
// 必须跳过，否则节点链接或注释里的 `global { log_level: ... }` 会污染结果。
func tokenize(content string) []string {
	tokens := make([]string, 0, strings.Count(content, "\n")*2)
	for index := 0; index < len(content); {
		switch {
		case isSpace(content[index]):
			index++
		case content[index] == '#':
			index = skipUntil(content, index+1, '\n')
		case content[index] == '/' && index+1 < len(content) && content[index+1] == '*':
			if end := strings.Index(content[index+2:], "*/"); end >= 0 {
				index += end + 4
			} else {
				return tokens
			}
		case content[index] == '\'' || content[index] == '"':
			index = skipString(content, index)
		case isIdentifierStart(content[index]):
			end := index + 1
			for end < len(content) && isIdentifierPart(content[end]) {
				end++
			}
			tokens = append(tokens, content[index:end])
			index = end
		case strings.ContainsRune("{}:", rune(content[index])):
			tokens = append(tokens, content[index:index+1])
			index++
		default:
			index++
		}
	}
	return tokens
}

func skipUntil(content string, index int, delimiter byte) int {
	for index < len(content) && content[index] != delimiter {
		index++
	}
	return index
}

func skipString(content string, index int) int {
	quote := content[index]
	for index++; index < len(content); index++ {
		if content[index] == '\\' && index+1 < len(content) {
			index++
			continue
		}
		if content[index] == quote {
			return index + 1
		}
	}
	return len(content)
}

func isSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isIdentifierPart(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}
