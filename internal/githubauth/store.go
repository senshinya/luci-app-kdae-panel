// Package githubauth 持久化面板访问 GitHub API 使用的只读凭据。
package githubauth

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
)

const tokenMode = 0o600

var (
	ErrEnvironmentManaged = errors.New("GitHub Token 由 KDAE_PANEL_GITHUB_TOKEN 环境变量管理，不能在面板内修改")
	ErrInvalidToken       = errors.New("GitHub Token 格式无效")
)

// Status 只描述凭据是否存在及其来源，绝不把 Token 本身返回给前端。
type Status struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"`
}

// Store 优先使用部署环境提供的 Token；没有环境值时才读取面板保存的独立文件。
type Store struct {
	mu          sync.RWMutex
	path        string
	environment string
	token       string
}

func Open(path, environment string) (*Store, error) {
	store := &Store{path: path, environment: strings.TrimSpace(environment)}
	if store.environment != "" {
		if err := validate(store.environment); err != nil {
			return nil, fmt.Errorf("KDAE_PANEL_GITHUB_TOKEN: %w", err)
		}
		store.token = store.environment
		return store, nil
	}

	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("检查 GitHub Token 文件: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("GitHub Token 文件 %s 不是普通文件", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 GitHub Token 文件: %w", err)
	}
	store.token = strings.TrimSpace(string(content))
	if store.token == "" {
		return store, nil
	}
	if err := validate(store.token); err != nil {
		return nil, fmt.Errorf("GitHub Token 文件: %w", err)
	}
	return store, nil
}

func (s *Store) GitHubToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

func (s *Store) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.environment != "" {
		return Status{Configured: true, Source: "environment"}
	}
	if s.token != "" {
		return Status{Configured: true, Source: "panel"}
	}
	return Status{}
}

func (s *Store) SetToken(token string) error {
	if s.environment != "" {
		return ErrEnvironmentManaged
	}
	token = strings.TrimSpace(token)
	if err := validate(token); err != nil {
		return err
	}
	if err := atomicfile.Write(s.path, []byte(token+"\n"), tokenMode); err != nil {
		return fmt.Errorf("保存 GitHub Token: %w", err)
	}
	s.mu.Lock()
	s.token = token
	s.mu.Unlock()
	return nil
}

func (s *Store) ClearToken() error {
	if s.environment != "" {
		return ErrEnvironmentManaged
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除 GitHub Token 文件: %w", err)
	}
	s.mu.Lock()
	s.token = ""
	s.mu.Unlock()
	return nil
}

func validate(token string) error {
	if len(token) < 20 || len(token) > 512 {
		return fmt.Errorf("%w：长度必须在 20 到 512 个字符之间", ErrInvalidToken)
	}
	for _, character := range token {
		if character > unicode.MaxASCII || unicode.IsSpace(character) || unicode.IsControl(character) {
			return fmt.Errorf("%w：只能包含不带空白的 ASCII 字符", ErrInvalidToken)
		}
	}
	return nil
}
