package daeconfig

import (
	"errors"
	"strings"
	"testing"
)

const sampleConfig = `global {
    log_level: info
}

node {
    local_a: 'vless://uuid@a.example.com:443#A'
    local_b: 'vless://uuid@b.example.com:443#B'
}

group {
    # policy: fixed(99) 这行在注释里，不能被选中
    proxy {
        filter: name(local_a, local_b)
        policy: fixed(1)
    }
    backup {
        policy: min_moving_avg
    }
}

routing {
    fallback: proxy
}
`

func TestGroupPolicySpanReturnsValueRange(t *testing.T) {
	start, end, err := GroupPolicySpan(sampleConfig, "proxy")
	if err != nil {
		t.Fatalf("定位 proxy 的 policy 失败: %v", err)
	}
	if got := sampleConfig[start:end]; got != "fixed(1)" {
		t.Fatalf("policy 值区间不对: %q", got)
	}
}

func TestGroupPolicySpanIgnoresComment(t *testing.T) {
	start, end, err := GroupPolicySpan(sampleConfig, "backup")
	if err != nil {
		t.Fatalf("定位 backup 的 policy 失败: %v", err)
	}
	if got := sampleConfig[start:end]; got != "min_moving_avg" {
		t.Fatalf("policy 值区间不对: %q", got)
	}
}

func TestGroupPolicySpanRejectsUnknownGroup(t *testing.T) {
	if _, _, err := GroupPolicySpan(sampleConfig, "missing"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("未知分组应判为不可定位，实际: %v", err)
	}
}

func TestGroupPolicySpanRejectsMissingPolicy(t *testing.T) {
	content := "group {\n    proxy {\n        filter: name(a)\n    }\n}\n"
	if _, _, err := GroupPolicySpan(content, "proxy"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("缺少 policy 应判为不可定位，实际: %v", err)
	}
}

func TestGroupPolicySpanRejectsDuplicatePolicy(t *testing.T) {
	content := "group {\n    proxy {\n        policy: fixed(0)\n        policy: random\n    }\n}\n"
	if _, _, err := GroupPolicySpan(content, "proxy"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("重复 policy 应判为不可定位，实际: %v", err)
	}
}

func TestGroupPolicySpanRejectsMultilinePolicy(t *testing.T) {
	content := "group {\n    proxy {\n        policy:\n            fixed(0)\n    }\n}\n"
	if _, _, err := GroupPolicySpan(content, "proxy"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("跨行 policy 应判为不可定位，实际: %v", err)
	}
}

func TestGroupPolicySpanIgnoresStructureInsideStrings(t *testing.T) {
	content := "node {\n    weird: 'vless://u@h:443#group { fake { policy: random } }'\n}\n" +
		"group {\n    proxy {\n        policy: fixed(2)\n    }\n}\n"
	start, end, err := GroupPolicySpan(content, "proxy")
	if err != nil {
		t.Fatalf("定位失败: %v", err)
	}
	if got := content[start:end]; got != "fixed(2)" {
		t.Fatalf("policy 值区间不对: %q", got)
	}
	if _, _, err := GroupPolicySpan(content, "fake"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("字符串里的假分组不该被找到，实际: %v", err)
	}
}

func TestGroupPolicySpanStopsAtTrailingComment(t *testing.T) {
	content := "group {\n    proxy {\n        policy: fixed(10)  # 固定第 10 个\n    }\n}\n"
	start, end, err := GroupPolicySpan(content, "proxy")
	if err != nil {
		t.Fatalf("定位失败: %v", err)
	}
	if got := content[start:end]; got != "fixed(10)" {
		t.Fatalf("policy 值区间不对: %q", got)
	}
}

func TestSetGroupPolicyReplacesOnlyThatValue(t *testing.T) {
	next, err := SetGroupPolicy(sampleConfig, "proxy", "fixed(0)")
	if err != nil {
		t.Fatalf("改写失败: %v", err)
	}
	if !strings.Contains(next, "policy: fixed(0)") {
		t.Fatalf("新 policy 未写入:\n%s", next)
	}
	if strings.Contains(next, "policy: fixed(1)") {
		t.Fatalf("旧 policy 仍在:\n%s", next)
	}
	if !strings.Contains(next, "policy: min_moving_avg") {
		t.Fatalf("另一个分组的 policy 被改动了:\n%s", next)
	}
	if !strings.Contains(next, "# policy: fixed(99) 这行在注释里，不能被选中") {
		t.Fatalf("注释被破坏了:\n%s", next)
	}
	if len(next)-len(sampleConfig) != len("fixed(0)")-len("fixed(1)") {
		t.Fatalf("改写影响了 policy 值以外的字节")
	}
}

func TestSetGroupPolicyRejectsUnlocatable(t *testing.T) {
	if _, err := SetGroupPolicy(sampleConfig, "missing", "fixed(0)"); !errors.Is(err, ErrGroupPolicyUnlocatable) {
		t.Fatalf("未知分组应报不可定位，实际: %v", err)
	}
}

func TestMaskContentPreservesLength(t *testing.T) {
	masked := maskContent(sampleConfig)
	if len(masked) != len(sampleConfig) {
		t.Fatalf("掩码改变了长度: %d vs %d", len(masked), len(sampleConfig))
	}
	if strings.Contains(masked, "fixed(99)") {
		t.Fatalf("注释内容未被抹掉")
	}
	if strings.Count(masked, "\n") != strings.Count(sampleConfig, "\n") {
		t.Fatalf("掩码改变了行数")
	}
}
