// Package subscriptioncache 只读解析 dae 已经落盘的订阅缓存。
package subscriptioncache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxSources   = 128
	maxFileBytes = 8 << 20
)

var validTag = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

type Node struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol,omitempty"`
	Host     string `json:"host,omitempty"`
	Matches  int    `json:"matches"`
}

type Source struct {
	Tag      string    `json:"tag"`
	Nodes    []Node    `json:"nodes"`
	CachedAt time.Time `json:"cachedAt"`
	Skipped  int       `json:"skipped,omitempty"`
	Problem  string    `json:"problem,omitempty"`
}

type Reader struct {
	directory string
}

func New(configPath string) (*Reader, error) {
	if strings.TrimSpace(configPath) == "" {
		return nil, errors.New("dae 配置路径不能为空")
	}
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("解析 dae 配置路径: %w", err)
	}
	return &Reader{directory: filepath.Join(filepath.Dir(absConfig), "persist.d")}, nil
}

func (r *Reader) List(ctx context.Context) ([]Source, error) {
	entries, err := os.ReadDir(r.directory)
	if os.IsNotExist(err) {
		return []Source{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取订阅缓存目录: %w", err)
	}

	files := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sub") && validTag.MatchString(strings.TrimSuffix(entry.Name(), ".sub")) {
			files = append(files, entry)
		}
	}
	if len(files) > maxSources {
		return nil, fmt.Errorf("订阅缓存数量超过 %d 个上限", maxSources)
	}

	sources := make([]Source, 0, len(files))
	for _, entry := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tag := strings.TrimSuffix(entry.Name(), ".sub")
		source := Source{Tag: tag, Nodes: []Node{}}
		path := filepath.Join(r.directory, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			source.Problem = "读取缓存状态失败: " + err.Error()
			sources = append(sources, source)
			continue
		}
		source.CachedAt = info.ModTime()
		if !info.Mode().IsRegular() {
			source.Problem = "缓存不是普通文件"
			sources = append(sources, source)
			continue
		}
		if info.Size() > maxFileBytes {
			source.Problem = fmt.Sprintf("缓存超过 %d MiB 上限", maxFileBytes>>20)
			sources = append(sources, source)
			continue
		}

		content, err := readLimited(path, info)
		if err != nil {
			source.Problem = err.Error()
			sources = append(sources, source)
			continue
		}
		source.Nodes, source.Skipped, err = parseSubscription(content)
		if err != nil {
			source.Problem = err.Error()
		}
		sources = append(sources, source)
	}
	sort.Slice(sources, func(left, right int) bool { return sources[left].Tag < sources[right].Tag })
	return sources, nil
}

func readLimited(path string, expected os.FileInfo) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开订阅缓存: %w", err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("读取订阅缓存状态: %w", err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(expected, opened) {
		return nil, errors.New("订阅缓存在读取前被替换")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取订阅缓存: %w", err)
	}
	if len(content) > maxFileBytes {
		return nil, fmt.Errorf("订阅缓存超过 %d MiB 上限", maxFileBytes>>20)
	}
	return content, nil
}
