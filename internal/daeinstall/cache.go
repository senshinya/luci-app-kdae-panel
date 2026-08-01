package daeinstall

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

const (
	cacheSchema           = 1
	cacheMagic            = "KDAEVC1\n"
	maxCacheMetadataBytes = 64 << 10
	maxCachedBinaryBytes  = 256 << 20
)

var (
	// ErrCachedVersionNotFound 供 API 把重复删除映射成 404。
	ErrCachedVersionNotFound = errors.New("本地版本不存在")
	errInvalidVersionCache   = errors.New("本地版本缓存无效")
)

// Version 在上游版本之上补充本机缓存状态；缓存不属于 upstream 的职责。
type Version struct {
	upstream.Version
	Cached      bool       `json:"cached,omitempty"`
	CachedOnly  bool       `json:"cachedOnly,omitempty"`
	CachedAt    *time.Time `json:"cachedAt,omitempty"`
	CachedBytes int64      `json:"cachedBytes,omitempty"`
}

type cacheMetadata struct {
	Schema        int             `json:"schema"`
	Source        upstream.Source `json:"source"`
	Ref           string          `json:"ref"`
	Label         string          `json:"label"`
	Platform      string          `json:"platform"`
	AssetPlatform string          `json:"assetPlatform,omitempty"`
	SHA256        string          `json:"sha256"`
	Size          int64           `json:"size"`
	CachedAt      time.Time       `json:"cachedAt"`
}

// versionCache 把每个版本保存成一个自描述文件，避免“二进制已写、索引未写”的
// 双文件中间态。列表只读文件头；真正切换时才读完整二进制并重新计算 sha256。
type versionCache struct {
	directory string
	mu        sync.RWMutex
}

func newVersionCache(statePath string) *versionCache {
	return &versionCache{directory: filepath.Join(filepath.Dir(statePath), "dae-versions")}
}

func cacheKey(source upstream.Source, ref, platform string) string {
	digest := sha256.Sum256([]byte(string(source) + "\x00" + ref + "\x00" + platform))
	return hex.EncodeToString(digest[:])
}

func (c *versionCache) path(source upstream.Source, ref, platform string) string {
	return filepath.Join(c.directory, cacheKey(source, ref, platform)+".cache")
}

func (c *versionCache) store(source upstream.Source, ref, label, platform, assetPlatform string, content []byte) error {
	if _, err := upstream.ParseSource(string(source)); err != nil {
		return err
	}
	if ref == "" || platform == "" {
		return errors.New("缓存版本缺少版本号或平台")
	}
	if err := assertELF(content); err != nil {
		return err
	}
	if len(content) > maxCachedBinaryBytes {
		return fmt.Errorf("dae 二进制超过 %d 字节缓存上限", maxCachedBinaryBytes)
	}
	if label == "" {
		label = ref
	}
	metadata := cacheMetadata{
		Schema:        cacheSchema,
		Source:        source,
		Ref:           ref,
		Label:         label,
		Platform:      platform,
		AssetPlatform: assetPlatform,
		SHA256:        digestBytes(content),
		Size:          int64(len(content)),
		CachedAt:      nowUTC(),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if len(encoded) > maxCacheMetadataBytes {
		return errors.New("缓存版本元数据过大")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.MkdirAll(c.directory, 0o700); err != nil {
		return fmt.Errorf("创建版本缓存目录: %w", err)
	}
	file, err := os.CreateTemp(c.directory, ".kdae-panel-version-*")
	if err != nil {
		return fmt.Errorf("创建版本缓存暂存文件: %w", err)
	}
	staged := file.Name()
	cleanup := func() { _ = os.Remove(staged) }
	defer cleanup()
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(encoded)))
	for _, part := range [][]byte{[]byte(cacheMagic), length[:], encoded, content} {
		if _, err := file.Write(part); err != nil {
			_ = file.Close()
			return fmt.Errorf("写入版本缓存: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("同步版本缓存: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭版本缓存: %w", err)
	}
	if err := atomicfile.Replace(staged, c.path(source, ref, platform)); err != nil {
		return fmt.Errorf("提交版本缓存: %w", err)
	}
	return nil
}

func (c *versionCache) load(source upstream.Source, ref, platform string) ([]byte, cacheMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	metadata, content, err := readCacheFile(c.path(source, ref, platform), true)
	return content, metadata, err
}

func (c *versionCache) list(source upstream.Source, platform string) ([]cacheMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entries, err := os.ReadDir(c.directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var versions []cacheMetadata
	var failures []error
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cache") {
			continue
		}
		metadata, _, err := readCacheFile(filepath.Join(c.directory, entry.Name()), false)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}
		if metadata.Source == source && metadata.Platform == platform {
			versions = append(versions, metadata)
		}
	}
	sort.Slice(versions, func(left, right int) bool {
		return versions[left].CachedAt.After(versions[right].CachedAt)
	})
	return versions, errors.Join(failures...)
}

func (c *versionCache) delete(source upstream.Source, ref, platform string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := c.path(source, ref, platform)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return ErrCachedVersionNotFound
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("缓存路径 %s 不是普通文件，拒绝删除", path)
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

func (c *versionCache) discardInvalid(source upstream.Source, ref, platform string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := c.path(source, ref, platform)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("缓存路径 %s 不是普通文件，拒绝清理", path)
	}
	return os.Remove(path)
}

func readCacheFile(path string, withBinary bool) (cacheMetadata, []byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return cacheMetadata{}, nil, ErrCachedVersionNotFound
	}
	if err != nil {
		return cacheMetadata{}, nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return cacheMetadata{}, nil, fmt.Errorf("%w：缓存不是普通文件", errInvalidVersionCache)
	}
	file, err := os.Open(path)
	if err != nil {
		return cacheMetadata{}, nil, err
	}
	defer file.Close()
	header := make([]byte, len(cacheMagic)+4)
	if _, err := io.ReadFull(file, header); err != nil {
		return cacheMetadata{}, nil, fmt.Errorf("%w：读取文件头: %v", errInvalidVersionCache, err)
	}
	if string(header[:len(cacheMagic)]) != cacheMagic {
		return cacheMetadata{}, nil, fmt.Errorf("%w：文件头不匹配", errInvalidVersionCache)
	}
	metadataSize := int64(binary.BigEndian.Uint32(header[len(cacheMagic):]))
	if metadataSize <= 0 || metadataSize > maxCacheMetadataBytes {
		return cacheMetadata{}, nil, fmt.Errorf("%w：元数据长度 %d 非法", errInvalidVersionCache, metadataSize)
	}
	encoded := make([]byte, int(metadataSize))
	if _, err := io.ReadFull(file, encoded); err != nil {
		return cacheMetadata{}, nil, fmt.Errorf("%w：读取元数据: %v", errInvalidVersionCache, err)
	}
	var metadata cacheMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		return cacheMetadata{}, nil, fmt.Errorf("%w：解析元数据: %v", errInvalidVersionCache, err)
	}
	remaining := info.Size() - int64(len(header)) - metadataSize
	if err := validateCacheMetadata(path, metadata, remaining); err != nil {
		return cacheMetadata{}, nil, err
	}
	if !withBinary {
		return metadata, nil, nil
	}
	content := make([]byte, int(metadata.Size))
	if _, err := io.ReadFull(file, content); err != nil {
		return cacheMetadata{}, nil, fmt.Errorf("%w：读取二进制: %v", errInvalidVersionCache, err)
	}
	if digestBytes(content) != metadata.SHA256 {
		return cacheMetadata{}, nil, fmt.Errorf("%w：sha256 不匹配", errInvalidVersionCache)
	}
	if err := assertELF(content); err != nil {
		return cacheMetadata{}, nil, fmt.Errorf("%w：%v", errInvalidVersionCache, err)
	}
	return metadata, content, nil
}

func validateCacheMetadata(path string, metadata cacheMetadata, remaining int64) error {
	if metadata.Schema != cacheSchema {
		return fmt.Errorf("%w：不支持的格式版本 %d", errInvalidVersionCache, metadata.Schema)
	}
	if _, err := upstream.ParseSource(string(metadata.Source)); err != nil {
		return fmt.Errorf("%w：%v", errInvalidVersionCache, err)
	}
	if metadata.Ref == "" || metadata.Label == "" || metadata.Platform == "" || metadata.CachedAt.IsZero() {
		return fmt.Errorf("%w：缺少必要元数据", errInvalidVersionCache)
	}
	if metadata.Size <= 0 || metadata.Size > maxCachedBinaryBytes || remaining != metadata.Size {
		return fmt.Errorf("%w：二进制长度 %d 非法", errInvalidVersionCache, metadata.Size)
	}
	digest, err := hex.DecodeString(metadata.SHA256)
	if err != nil || len(digest) != sha256.Size {
		return fmt.Errorf("%w：sha256 格式错误", errInvalidVersionCache)
	}
	expected := cacheKey(metadata.Source, metadata.Ref, metadata.Platform) + ".cache"
	if filepath.Base(path) != expected {
		return fmt.Errorf("%w：文件名与元数据不匹配", errInvalidVersionCache)
	}
	return nil
}
