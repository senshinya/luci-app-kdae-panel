package daeconfig

import (
	"reflect"
	"testing"
)

func TestDNSRequestMatchers(t *testing.T) {
	content := `global {
  marker: node(not_a_match)
}
# dns { routing { request { sub(comment) -> resolver } } }
dns {
  upstream { resolver: 'udp://node(example):53' }
  routing {
    request {
      qname(regex: 'subnode\(ignored\)') -> resolver
      sub(my_sub) -> resolver
      node(name_keyword: hk) -> resolver
      subnode(subtag: my_sub) -> resolver
    }
    response {
      node(not_a_request_match) -> accept
    }
  }
}`
	want := []string{"sub", "node", "subnode"}
	if got := DNSRequestMatchers(content); !reflect.DeepEqual(got, want) {
		t.Fatalf("DNSRequestMatchers() = %v, want %v", got, want)
	}
}

func TestDNSRequestMatchersIgnoresOtherScopesAndComments(t *testing.T) {
	content := `/* dns { routing { request { node(block_comment) -> x } } } */
routing { node(other_section) -> proxy }
dns {
  routing {
    response { sub(other_section) -> accept }
  }
}`
	if got := DNSRequestMatchers(content); len(got) != 0 {
		t.Fatalf("不应识别其他范围或注释: %v", got)
	}
}
