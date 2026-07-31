package upstream

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
)

const (
	customGeoSourcePrefix = "custom:"
	maxCustomGeoSources   = 24
	maxCustomGeoURLBytes  = 4096
	customGeoStoreMode    = 0o600
	maxGeoDigestBytes     = 64 << 10
)

var ErrCustomGeoSourceNotFound = errors.New("自定义 geo 来源不存在")

// CustomGeoSource 是管理员保存的一组 geo 数据直链。
//
// 四个地址都必须是公网 HTTPS。校验文件独立填写，是为了让数据与摘要可以来自
// 不同 CDN；面板不会提供跳过 SHA-256 的开关。
type CustomGeoSource struct {
	ID               string    `json:"id"`
	Source           GeoSource `json:"source"`
	Label            string    `json:"label"`
	GeoIPURL         string    `json:"geoipUrl"`
	GeoIPSHA256URL   string    `json:"geoipSha256Url"`
	GeoSiteURL       string    `json:"geositeUrl"`
	GeoSiteSHA256URL string    `json:"geositeSha256Url"`
}

type customGeoSourceFile struct {
	Sources []CustomGeoSource `json:"sources"`
}

// customGeoStore 用单独的 0600 文件持久化来源。URL 的查询串可能带有管理员自行
// 配置的临时凭据，因此不能混进普通配置、日志或对外可读的文件。
type customGeoStore struct {
	mu      sync.RWMutex
	path    string
	sources []CustomGeoSource
}

func openCustomGeoStore(filePath string) (*customGeoStore, error) {
	store := &customGeoStore{path: filePath}
	if filePath == "" {
		return store, nil
	}
	info, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("检查自定义 geo 来源文件: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("自定义 geo 来源文件 %s 不是普通文件", filePath)
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("自定义 geo 来源文件超过 1 MiB 限制")
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取自定义 geo 来源: %w", err)
	}
	var persisted customGeoSourceFile
	if err := json.Unmarshal(content, &persisted); err != nil {
		return nil, fmt.Errorf("解析自定义 geo 来源: %w", err)
	}
	if len(persisted.Sources) > maxCustomGeoSources {
		return nil, fmt.Errorf("自定义 geo 来源超过 %d 个限制", maxCustomGeoSources)
	}
	seen := make(map[string]bool, len(persisted.Sources))
	for index := range persisted.Sources {
		source := &persisted.Sources[index]
		if err := normalizeCustomGeoSource(source, source.ID); err != nil {
			return nil, fmt.Errorf("第 %d 个自定义 geo 来源: %w", index+1, err)
		}
		if seen[source.ID] {
			return nil, fmt.Errorf("自定义 geo 来源 ID %q 重复", source.ID)
		}
		seen[source.ID] = true
	}
	store.sources = persisted.Sources
	return store, nil
}

func (s *customGeoStore) list() []CustomGeoSource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// API 契约是数组。返回 nil 会被编码成 null，前端在删除最后一个来源后读取
	// .length 时会中断渲染，连空态和关闭按钮都会一起消失。
	return append([]CustomGeoSource{}, s.sources...)
}

func (s *customGeoStore) get(source GeoSource) (CustomGeoSource, bool) {
	id, ok := customGeoSourceID(source)
	if !ok {
		return CustomGeoSource{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, candidate := range s.sources {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return CustomGeoSource{}, false
}

func (s *customGeoStore) create(input CustomGeoSource) (CustomGeoSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sources) >= maxCustomGeoSources {
		return CustomGeoSource{}, fmt.Errorf("最多保存 %d 个自定义 geo 来源", maxCustomGeoSources)
	}
	id, err := newCustomGeoSourceID()
	if err != nil {
		return CustomGeoSource{}, err
	}
	if err := normalizeCustomGeoSource(&input, id); err != nil {
		return CustomGeoSource{}, err
	}
	next := append(append([]CustomGeoSource(nil), s.sources...), input)
	if err := s.write(next); err != nil {
		return CustomGeoSource{}, err
	}
	s.sources = next
	return input, nil
}

func (s *customGeoStore) update(id string, input CustomGeoSource) (CustomGeoSource, error) {
	if err := validateCustomGeoSourceID(id); err != nil {
		return CustomGeoSource{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.sources, func(source CustomGeoSource) bool { return source.ID == id })
	if index < 0 {
		return CustomGeoSource{}, ErrCustomGeoSourceNotFound
	}
	if err := normalizeCustomGeoSource(&input, id); err != nil {
		return CustomGeoSource{}, err
	}
	next := append([]CustomGeoSource(nil), s.sources...)
	next[index] = input
	if err := s.write(next); err != nil {
		return CustomGeoSource{}, err
	}
	s.sources = next
	return input, nil
}

func (s *customGeoStore) delete(id string) error {
	if err := validateCustomGeoSourceID(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.sources, func(source CustomGeoSource) bool { return source.ID == id })
	if index < 0 {
		return ErrCustomGeoSourceNotFound
	}
	next := append([]CustomGeoSource(nil), s.sources[:index]...)
	next = append(next, s.sources[index+1:]...)
	if err := s.write(next); err != nil {
		return err
	}
	s.sources = next
	return nil
}

func (s *customGeoStore) write(sources []CustomGeoSource) error {
	if s.path == "" {
		return errors.New("自定义 geo 来源未配置持久化文件")
	}
	content, err := json.MarshalIndent(customGeoSourceFile{Sources: sources}, "", "  ")
	if err != nil {
		return fmt.Errorf("编码自定义 geo 来源: %w", err)
	}
	if err := atomicfile.Write(s.path, append(content, '\n'), customGeoStoreMode); err != nil {
		return fmt.Errorf("保存自定义 geo 来源: %w", err)
	}
	return nil
}

func normalizeCustomGeoSource(source *CustomGeoSource, id string) error {
	if err := validateCustomGeoSourceID(id); err != nil {
		return err
	}
	source.ID = id
	source.Source = GeoSource(customGeoSourcePrefix + id)
	source.Label = strings.TrimSpace(source.Label)
	if source.Label == "" || utf8.RuneCountInString(source.Label) > 80 {
		return errors.New("来源名称长度必须在 1 到 80 个字符之间")
	}
	for _, character := range source.Label {
		if unicode.IsControl(character) {
			return errors.New("来源名称不能包含控制字符")
		}
	}
	source.GeoIPURL = strings.TrimSpace(source.GeoIPURL)
	source.GeoIPSHA256URL = strings.TrimSpace(source.GeoIPSHA256URL)
	source.GeoSiteURL = strings.TrimSpace(source.GeoSiteURL)
	source.GeoSiteSHA256URL = strings.TrimSpace(source.GeoSiteSHA256URL)
	for label, target := range map[string]string{
		"geoip.dat 地址":     source.GeoIPURL,
		"geoip.dat 校验地址":   source.GeoIPSHA256URL,
		"geosite.dat 地址":   source.GeoSiteURL,
		"geosite.dat 校验地址": source.GeoSiteSHA256URL,
	} {
		if len(target) == 0 || len(target) > maxCustomGeoURLBytes {
			return fmt.Errorf("%s长度必须在 1 到 %d 字节之间", label, maxCustomGeoURLBytes)
		}
		if _, err := parsePublicHTTPSURL(target); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}
	return nil
}

func newCustomGeoSourceID() (string, error) {
	content := make([]byte, 8)
	if _, err := rand.Read(content); err != nil {
		return "", fmt.Errorf("生成自定义 geo 来源 ID: %w", err)
	}
	return hex.EncodeToString(content), nil
}

func validateCustomGeoSourceID(id string) error {
	if len(id) != 16 {
		return errors.New("自定义 geo 来源 ID 无效")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return errors.New("自定义 geo 来源 ID 无效")
	}
	return nil
}

func customGeoSourceID(source GeoSource) (string, bool) {
	value := string(source)
	if !strings.HasPrefix(value, customGeoSourcePrefix) {
		return "", false
	}
	id := strings.TrimPrefix(value, customGeoSourcePrefix)
	return id, validateCustomGeoSourceID(id) == nil
}

type customGeoProvider struct {
	client *httpClient
	source CustomGeoSource
}

func (p *customGeoProvider) Info() GeoSourceInfo {
	repositories := make([]string, 0, 4)
	for _, target := range []string{
		p.source.GeoIPURL, p.source.GeoIPSHA256URL,
		p.source.GeoSiteURL, p.source.GeoSiteSHA256URL,
	} {
		parsed, _ := url.Parse(target)
		if host := parsed.Hostname(); host != "" && !slices.Contains(repositories, host) {
			repositories = append(repositories, host)
		}
	}
	slices.Sort(repositories)
	return GeoSourceInfo{
		Source:       p.source.Source,
		Label:        p.source.Label,
		Repositories: repositories,
		Note:         "管理员配置的公网 HTTPS 直链；每次更新都会重新下载并校验两个文件的 SHA-256。",
		Custom:       true,
	}
}

func (p *customGeoProvider) Latest(context.Context) (GeoRelease, error) {
	now := time.Now().UTC()
	return GeoRelease{
		Source:      p.source.Source,
		Tag:         "实时链接",
		PublishedAt: now,
		Files: map[string]GeoFile{
			GeoIPName: customGeoFile(GeoIPName, p.source.GeoIPURL, p.source.GeoIPSHA256URL, now),
			GeoSiteName: customGeoFile(GeoSiteName, p.source.GeoSiteURL,
				p.source.GeoSiteSHA256URL, now),
		},
	}, nil
}

func customGeoFile(name, downloadURL, digestURL string, publishedAt time.Time) GeoFile {
	parsed, _ := url.Parse(downloadURL)
	asset := path.Base(parsed.Path)
	if asset == "." || asset == "/" || asset == "" {
		asset = name
	}
	return GeoFile{
		Name:        name,
		Repository:  parsed.Hostname(),
		Asset:       asset,
		PublishedAt: publishedAt,
		downloadURL: downloadURL,
		digestURL:   digestURL,
	}
}

// Fetch 对自定义来源执行与内置来源相同的完整性约束。URL 来自 GeoRelease 的
// 不导出字段，因此来源在 Latest 与 Fetch 之间被编辑也不会混用两组链接。
func (p *customGeoProvider) Fetch(ctx context.Context, release GeoRelease) (GeoData, error) {
	data := GeoData{Release: release, Files: make(map[string][]byte, len(release.Files))}
	digests := make(map[string]string, len(release.Files))
	for _, name := range []string{GeoIPName, GeoSiteName} {
		file, ok := release.Files[name]
		if !ok || file.downloadURL == "" || file.digestURL == "" {
			return GeoData{}, fmt.Errorf("自定义来源 %s 缺少 %s 的地址", release.Source, name)
		}
		digestContent, err := p.client.download(ctx, file.digestURL, maxGeoDigestBytes)
		if err != nil {
			return GeoData{}, fmt.Errorf("下载 %s 校验文件: %w", name, err)
		}
		digest, err := parseSHA256Sum(string(digestContent), file.Asset)
		if err != nil {
			return GeoData{}, err
		}
		content, err := p.client.download(ctx, file.downloadURL, maxGeoBytes)
		if err != nil {
			return GeoData{}, fmt.Errorf("下载 %s: %w", name, err)
		}
		if err := verifyDigest(content, digest); err != nil {
			return GeoData{}, fmt.Errorf("%s: %w", name, err)
		}
		file.Size = int64(len(content))
		file.Tag = digest[:12]
		data.Release.Files[name] = file
		data.Files[name] = content
		digests[name] = digest
	}
	data.Release.Tag = "geoip " + digests[GeoIPName][:12] + " / geosite " + digests[GeoSiteName][:12]
	data.Release.PublishedAt = time.Now().UTC()
	return data, nil
}
