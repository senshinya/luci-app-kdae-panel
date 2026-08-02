package daeconfig

import "testing"

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "多行", content: "global {\n  log_level: warn\n}\n", want: "warn"},
		{name: "紧凑写法", content: "global { log_level: debug }", want: "debug"},
		{name: "默认值", content: "global { lan_interface: auto }", want: "info"},
		{
			name: "跳过注释与字符串",
			content: `# global { log_level: error }
node { demo: 'global { log_level: trace }' }
/* global { log_level: warn } */
global { log_level: info }`,
			want: "info",
		},
		{name: "未知值", content: "global { log_level: verbose }", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := LogLevel(test.content); got != test.want {
				t.Fatalf("LogLevel() = %q，期望 %q", got, test.want)
			}
		})
	}
}
