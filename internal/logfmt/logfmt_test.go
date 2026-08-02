package logfmt

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	want := map[string]string{
		"level":    "info",
		"msg":      `192.0.2.1:1234 <-> example.com:443`,
		"network":  "tcp4",
		"outbound": "proxy",
	}
	got, ok := Parse(`level=info msg="192.0.2.1:1234 <-> example.com:443" network=tcp4 outbound=proxy`)
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse() = %#v, %v，期望 %#v, true", got, ok, want)
	}
}

func TestParseQuotedEscapesAndDuplicateKeys(t *testing.T) {
	fields, ok := Parse(`level=info msg="node \"quoted\" outbound=block" outbound=proxy path="C:\\dae" outbound=direct`)
	if !ok {
		t.Fatal("带转义的合法日志应能解析")
	}
	if fields["msg"] != `node "quoted" outbound=block` {
		t.Fatalf("msg = %q", fields["msg"])
	}
	if fields["outbound"] != "proxy" {
		t.Fatalf("重复字段应保留首次值，实际为 %q", fields["outbound"])
	}
	if fields["path"] != `C:\dae` {
		t.Fatalf("path = %q", fields["path"])
	}
}

func TestParseRejectsMalformedPrefixAndStopsAtMalformedSuffix(t *testing.T) {
	if fields, ok := Parse(`not-logfmt level=info`); ok || fields != nil {
		t.Fatalf("非 logfmt 前缀不应被接受: %#v, %v", fields, ok)
	}
	fields, ok := Parse(`level=info msg="unterminated outbound=block`)
	if !ok || !reflect.DeepEqual(fields, map[string]string{"level": "info"}) {
		t.Fatalf("已完成字段应保留、残缺字段应丢弃: %#v, %v", fields, ok)
	}
}

func TestParseEmptyValueAndWhitespace(t *testing.T) {
	fields, ok := Parse("  empty=\tquoted=\"\"\r\nlevel=debug  ")
	want := map[string]string{"empty": "", "quoted": "", "level": "debug"}
	if !ok || !reflect.DeepEqual(fields, want) {
		t.Fatalf("Parse() = %#v, %v，期望 %#v, true", fields, ok, want)
	}
}
