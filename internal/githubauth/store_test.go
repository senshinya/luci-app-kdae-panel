package githubauth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const testToken = "github_pat_0123456789abcdefghijklmnop"

func TestStorePersistsWithoutRevealingToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials", "github-token")
	store, err := Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if store.Status().Configured {
		t.Fatal("新存储不应已配置")
	}
	if err := store.SetToken(testToken); err != nil {
		t.Fatal(err)
	}
	if status := store.Status(); !status.Configured || status.Source != "panel" {
		t.Fatalf("状态 = %+v", status)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("文件权限 = %o，期望 600", info.Mode().Perm())
	}

	reopened, err := Open(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.GitHubToken() != testToken {
		t.Fatal("重启后没有读回 Token")
	}
	if err := reopened.ClearToken(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("清除后文件仍存在: %v", err)
	}
}

func TestEnvironmentTokenCannotBeChangedFromPanel(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "github-token"), testToken)
	if err != nil {
		t.Fatal(err)
	}
	if status := store.Status(); status.Source != "environment" {
		t.Fatalf("状态 = %+v", status)
	}
	if err := store.SetToken(testToken + "x"); !errors.Is(err, ErrEnvironmentManaged) {
		t.Fatalf("设置错误 = %v", err)
	}
	if err := store.ClearToken(); !errors.Is(err, ErrEnvironmentManaged) {
		t.Fatalf("清除错误 = %v", err)
	}
	if store.GitHubToken() != testToken {
		t.Fatal("环境 Token 被修改")
	}
}

func TestStoreRejectsSymlinkAndMalformedToken(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte(testToken), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "github-token")
	if err := os.Symlink(target, link); err == nil {
		if _, err := Open(link, ""); err == nil {
			t.Fatal("不应读取符号链接")
		}
	}

	store, err := Open(filepath.Join(directory, "new-token"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"short", testToken + "\nsecond"} {
		if err := store.SetToken(token); err == nil {
			t.Fatalf("Token %q 应被拒绝", token)
		}
	}
}
