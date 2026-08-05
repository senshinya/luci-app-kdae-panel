package daeconfig

var dnsInternalRequestMatchers = [...]string{"sub", "node", "subnode"}

// DNSRequestMatchers 只识别 dns.routing.request 内实际调用的内部选择器。
// tokenize 会跳过注释和字符串，因此节点链接或说明文字不会造成误判。
func DNSRequestMatchers(content string) []string {
	tokens := tokenize(content)
	scopes := make([]string, 0, 4)
	used := make(map[string]bool, len(dnsInternalRequestMatchers))

	for index, token := range tokens {
		switch token {
		case "{":
			name := ""
			if index > 0 && identifierToken(tokens[index-1]) {
				name = tokens[index-1]
			}
			scopes = append(scopes, name)
		case "}":
			if len(scopes) > 0 {
				scopes = scopes[:len(scopes)-1]
			}
		default:
			if !inDNSRequestScope(scopes) || index+1 >= len(tokens) || tokens[index+1] != "(" {
				continue
			}
			for _, matcher := range dnsInternalRequestMatchers {
				if token == matcher {
					used[matcher] = true
					break
				}
			}
		}
	}

	result := make([]string, 0, len(used))
	for _, matcher := range dnsInternalRequestMatchers {
		if used[matcher] {
			result = append(result, matcher)
		}
	}
	return result
}

func inDNSRequestScope(scopes []string) bool {
	return len(scopes) == 3 &&
		scopes[0] == "dns" &&
		scopes[1] == "routing" &&
		scopes[2] == "request"
}

func identifierToken(token string) bool {
	if token == "" || !isIdentifierStart(token[0]) {
		return false
	}
	for index := 1; index < len(token); index++ {
		if !isIdentifierPart(token[index]) {
			return false
		}
	}
	return true
}
