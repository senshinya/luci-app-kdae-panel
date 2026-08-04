package daeconfig

import "strings"

// maskContent 返回与 content 等长的副本：注释整体、字符串字面量的内容都替换成空格，
// 换行保留。这样后续扫描可以直接按字节偏移定位原文，又不会被注释或节点链接里的
// `group {`、`policy:` 之类的假结构骗到。
func maskContent(content string) string {
	output := []byte(content)
	blank := func(from, to int) {
		for index := from; index < to && index < len(output); index++ {
			if output[index] != '\n' {
				output[index] = ' '
			}
		}
	}
	for index := 0; index < len(content); {
		switch {
		case content[index] == '#':
			end := strings.IndexByte(content[index:], '\n')
			if end < 0 {
				blank(index, len(content))
				index = len(content)
			} else {
				blank(index, index+end)
				index += end
			}
		case content[index] == '/' && index+1 < len(content) && content[index+1] == '*':
			closing := strings.Index(content[index+2:], "*/")
			end := len(content)
			if closing >= 0 {
				end = index + 2 + closing + 2
			}
			blank(index, end)
			index = end
		case content[index] == '\'' || content[index] == '"':
			end := skipString(content, index)
			// 只抹掉引号之间的内容，保留两端引号，扫描时它们会被当作普通符号忽略。
			blank(index+1, end-1)
			index = end
		default:
			index++
		}
	}
	return string(output)
}
