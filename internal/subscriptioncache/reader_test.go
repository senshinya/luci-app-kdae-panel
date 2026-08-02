package subscriptioncache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReaderListsSanitizedSubscriptionNodes(t *testing.T) {
	directory := t.TempDir()
	persist := filepath.Join(directory, "persist.d")
	if err := os.Mkdir(persist, 0o700); err != nil {
		t.Fatal(err)
	}
	vmessJSON := `{"ps":"新加坡 01","add":"sg.example.com","port":"443"}`
	vmess := "vmess://" + base64.RawStdEncoding.EncodeToString([]byte(vmessJSON))
	links := strings.Join([]string{
		"vless://secret@hk.example.com:443#香港%2001",
		"trojan://another-secret@backup.example.com:443#香港%2001",
		vmess,
		"ss://secret@example.com:443",
	}, "\n")
	cache := base64.StdEncoding.EncodeToString([]byte(links))
	if err := os.WriteFile(filepath.Join(persist, "main.sub"), []byte(cache), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(persist, "../ignored.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}

	reader, err := New(filepath.Join(directory, "config.dae"))
	if err != nil {
		t.Fatal(err)
	}
	sources, err := reader.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].Tag != "main" || len(sources[0].Nodes) != 2 || sources[0].Skipped != 1 {
		t.Fatalf("订阅缓存结果异常: %+v", sources)
	}
	if sources[0].Nodes[0].Name != "香港 01" || sources[0].Nodes[0].Matches != 2 {
		t.Fatalf("同名节点未聚合: %+v", sources[0].Nodes[0])
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") {
		t.Fatalf("API 数据泄露了节点认证信息: %s", encoded)
	}
}

func TestReaderParsesSIP008AndReportsInvalidFiles(t *testing.T) {
	directory := t.TempDir()
	persist := filepath.Join(directory, "persist.d")
	if err := os.Mkdir(persist, 0o700); err != nil {
		t.Fatal(err)
	}
	sip := `{"version":1,"servers":[{"remarks":"东京","server":"jp.example.com","server_port":443}]}`
	if err := os.WriteFile(filepath.Join(persist, "sip.sub"), []byte(sip), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(persist, "broken.sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(persist, "../bad tag.sub"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}

	reader, _ := New(filepath.Join(directory, "config.dae"))
	sources, err := reader.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 || sources[0].Tag != "broken" || sources[0].Problem == "" {
		t.Fatalf("异常缓存没有被安全报告: %+v", sources)
	}
	if len(sources[1].Nodes) != 1 || sources[1].Nodes[0].Name != "东京" {
		t.Fatalf("SIP008 解析异常: %+v", sources[1])
	}
}

func TestReaderReturnsEmptyWhenPersistDirectoryDoesNotExist(t *testing.T) {
	reader, err := New(filepath.Join(t.TempDir(), "config.dae"))
	if err != nil {
		t.Fatal(err)
	}
	sources, err := reader.List(context.Background())
	if err != nil || sources == nil || len(sources) != 0 {
		t.Fatalf("空缓存结果异常: sources=%v err=%v", sources, err)
	}
}
