package dae

import "strings"

var dnsInternalRequestMatchers = [...]string{"sub", "node", "subnode"}

// DNSRequestMatchers 返回目标二进制明确声明支持的内部 DNS 请求选择器。
// known 为假表示 outline 没有提供 request 描述，此时不能把“未知”当成“支持”。
func DNSRequestMatchers(outline Outline) (supported []string, known bool) {
	dns := findOutlineElement(outline.Structure, "dns")
	if dns == nil {
		return nil, false
	}
	request := findOutlineElement(dns.Structure, "request")
	if request == nil || strings.TrimSpace(request.Description) == "" {
		return nil, false
	}
	for _, matcher := range dnsInternalRequestMatchers {
		if containsIdentifier(request.Description, matcher) {
			supported = append(supported, matcher)
		}
	}
	return supported, true
}

func findOutlineElement(elements []OutlineElement, mapping string) *OutlineElement {
	for index := range elements {
		element := &elements[index]
		if strings.EqualFold(element.Mapping, mapping) {
			return element
		}
		if nested := findOutlineElement(element.Structure, mapping); nested != nil {
			return nested
		}
	}
	return nil
}

func containsIdentifier(text, identifier string) bool {
	for offset := 0; offset < len(text); {
		found := strings.Index(text[offset:], identifier)
		if found < 0 {
			return false
		}
		start := offset + found
		end := start + len(identifier)
		if (start == 0 || !isIdentifierByte(text[start-1])) &&
			(end == len(text) || !isIdentifierByte(text[end])) {
			return true
		}
		offset = end
	}
	return false
}

func isIdentifierByte(value byte) bool {
	return value == '_' || value >= '0' && value <= '9' ||
		value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
