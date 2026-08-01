package configstore

import "strings"

const (
	maxExactDiffCells = 4_000_000
	maxDiffLines      = 4_000
	maxDiffInputLines = 50_000
	diffContextLines  = 3
)

type lineEdit struct {
	kind string
	text string
}

// compareConfigLines 生成“当前配置 -> 存档配置”的逐行差异。
// 常见配置使用精确 LCS；超大配置只比较公共前后缀，避免构造不受控的矩阵。
func compareConfigLines(current, backup []byte) ([]DiffLine, bool) {
	oldCount := strings.Count(string(current), "\n") + 1
	newCount := strings.Count(string(backup), "\n") + 1
	if oldCount > maxDiffInputLines || newCount > maxDiffInputLines {
		return []DiffLine{{
			Kind: "skip", Text: "配置行数过多，未展开逐行差异", SkipCount: max(oldCount, newCount),
		}}, true
	}
	oldLines := splitConfigLines(string(current))
	newLines := splitConfigLines(string(backup))
	exact := len(oldLines) == 0 || len(newLines) == 0 || len(oldLines) <= maxExactDiffCells/len(newLines)
	var edits []lineEdit
	if exact {
		edits = exactLineDiff(oldLines, newLines)
	} else {
		edits = boundedLineDiff(oldLines, newLines)
	}
	lines := numberDiffLines(edits)
	lines = collapseEqualLines(lines)
	lines, outputTruncated := truncateDiffLines(lines)
	return lines, !exact || outputTruncated
}

func splitConfigLines(content string) []string {
	if content == "" {
		return nil
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{""}
	}
	return strings.Split(content, "\n")
}

func exactLineDiff(oldLines, newLines []string) []lineEdit {
	columns := len(newLines) + 1
	lcs := make([]uint32, (len(oldLines)+1)*columns)
	for old := len(oldLines) - 1; old >= 0; old-- {
		for new := len(newLines) - 1; new >= 0; new-- {
			index := old*columns + new
			if oldLines[old] == newLines[new] {
				lcs[index] = lcs[(old+1)*columns+new+1] + 1
			} else {
				lcs[index] = max(lcs[(old+1)*columns+new], lcs[old*columns+new+1])
			}
		}
	}

	edits := make([]lineEdit, 0, len(oldLines)+len(newLines))
	old, new := 0, 0
	for old < len(oldLines) && new < len(newLines) {
		switch {
		case oldLines[old] == newLines[new]:
			edits = append(edits, lineEdit{kind: "context", text: oldLines[old]})
			old++
			new++
		case lcs[(old+1)*columns+new] >= lcs[old*columns+new+1]:
			edits = append(edits, lineEdit{kind: "remove", text: oldLines[old]})
			old++
		default:
			edits = append(edits, lineEdit{kind: "add", text: newLines[new]})
			new++
		}
	}
	for ; old < len(oldLines); old++ {
		edits = append(edits, lineEdit{kind: "remove", text: oldLines[old]})
	}
	for ; new < len(newLines); new++ {
		edits = append(edits, lineEdit{kind: "add", text: newLines[new]})
	}
	return edits
}

func boundedLineDiff(oldLines, newLines []string) []lineEdit {
	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix &&
		oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}
	edits := make([]lineEdit, 0, prefix+suffix+len(oldLines)+len(newLines))
	for _, line := range oldLines[:prefix] {
		edits = append(edits, lineEdit{kind: "context", text: line})
	}
	for _, line := range oldLines[prefix : len(oldLines)-suffix] {
		edits = append(edits, lineEdit{kind: "remove", text: line})
	}
	for _, line := range newLines[prefix : len(newLines)-suffix] {
		edits = append(edits, lineEdit{kind: "add", text: line})
	}
	for _, line := range oldLines[len(oldLines)-suffix:] {
		edits = append(edits, lineEdit{kind: "context", text: line})
	}
	return edits
}

func numberDiffLines(edits []lineEdit) []DiffLine {
	lines := make([]DiffLine, 0, len(edits))
	oldLine, newLine := 1, 1
	for _, edit := range edits {
		line := DiffLine{Kind: edit.kind, Text: edit.text}
		switch edit.kind {
		case "context":
			line.OldLine, line.NewLine = oldLine, newLine
			oldLine++
			newLine++
		case "remove":
			line.OldLine = oldLine
			oldLine++
		case "add":
			line.NewLine = newLine
			newLine++
		}
		lines = append(lines, line)
	}
	return lines
}

func collapseEqualLines(lines []DiffLine) []DiffLine {
	if len(lines) == 0 {
		return nil
	}
	result := make([]DiffLine, 0, len(lines))
	for start := 0; start < len(lines); {
		if lines[start].Kind != "context" {
			result = append(result, lines[start])
			start++
			continue
		}
		end := start
		for end < len(lines) && lines[end].Kind == "context" {
			end++
		}
		count := end - start
		keepHead := diffContextLines
		keepTail := diffContextLines
		if start == 0 {
			keepHead = 0
		}
		if end == len(lines) {
			keepTail = 0
		}
		if count <= keepHead+keepTail {
			result = append(result, lines[start:end]...)
		} else {
			result = append(result, lines[start:start+keepHead]...)
			result = append(result, DiffLine{Kind: "skip", Text: "未变内容", SkipCount: count - keepHead - keepTail})
			result = append(result, lines[end-keepTail:end]...)
		}
		start = end
	}
	return result
}

func truncateDiffLines(lines []DiffLine) ([]DiffLine, bool) {
	if len(lines) <= maxDiffLines {
		return lines, false
	}
	head := maxDiffLines / 2
	tail := maxDiffLines - head
	result := append([]DiffLine(nil), lines[:head]...)
	result = append(result, DiffLine{Kind: "skip", Text: "差异过多，省略中间部分", SkipCount: len(lines) - maxDiffLines})
	result = append(result, lines[len(lines)-tail:]...)
	return result, true
}
