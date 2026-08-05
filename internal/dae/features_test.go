package dae

import (
	"reflect"
	"testing"
)

func TestDNSRequestMatchers(t *testing.T) {
	outline := Outline{Structure: []OutlineElement{{
		Mapping: "dns",
		Structure: []OutlineElement{{
			Mapping: "routing",
			Structure: []OutlineElement{{
				Mapping:     "request",
				Description: "Available: qname, qtype, sub, node and subnode.",
			}},
		}},
	}}}
	supported, known := DNSRequestMatchers(outline)
	if !known || !reflect.DeepEqual(supported, []string{"sub", "node", "subnode"}) {
		t.Fatalf("DNSRequestMatchers() = %v, %v", supported, known)
	}
}

func TestDNSRequestMatchersDoesNotInferFromSimilarWords(t *testing.T) {
	outline := Outline{Structure: []OutlineElement{{
		Mapping: "dns",
		Structure: []OutlineElement{{
			Mapping:     "request",
			Description: "Only qname and subnode_extension are available.",
		}},
	}}}
	supported, known := DNSRequestMatchers(outline)
	if !known || len(supported) != 0 {
		t.Fatalf("不应从相似单词推断能力: %v, %v", supported, known)
	}
}

func TestDNSRequestMatchersUnknownWithoutDescription(t *testing.T) {
	outline := Outline{Structure: []OutlineElement{{
		Mapping: "dns",
		Structure: []OutlineElement{{
			Mapping: "request",
		}},
	}}}
	if supported, known := DNSRequestMatchers(outline); known || len(supported) != 0 {
		t.Fatalf("缺少描述时应为未知: %v, %v", supported, known)
	}
}
