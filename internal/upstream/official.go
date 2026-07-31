package upstream

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OfficialProvider 读取 daeuniverse/dae 的 GitHub Release。
// 每个资产都配一个 .dgst 文件,里面按 "<哈希>  <文件名>  <算法>" 分行列出多种摘要。
type OfficialProvider struct {
	client *httpClient
	owner  string
	repo   string
	now    func() time.Time

	cacheMu  sync.RWMutex
	releases map[string]cachedRelease
}

type cachedRelease struct {
	release   githubRelease
	expiresAt time.Time
}

func NewOfficialProvider(client *httpClient, owner, repo string) *OfficialProvider {
	return &OfficialProvider{
		client: client, owner: owner, repo: repo, now: time.Now,
		releases: make(map[string]cachedRelease),
	}
}

func (p *OfficialProvider) Source() Source {
	return SourceOfficial
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// githubAsset 只取用得上的字段。刻意不取 browser_download_url：
// 下载地址由面板自己拼，不采信响应里的 URL。
type githubAsset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (p *OfficialProvider) List(ctx context.Context, limit int) ([]Version, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", p.owner, p.repo, limit)
	var releases []githubRelease
	if err := p.client.getJSON(ctx, endpoint, &releases); err != nil {
		return nil, err
	}

	versions := make([]Version, 0, len(releases))
	for _, release := range releases {
		if release.Draft || release.TagName == "" {
			continue
		}
		p.rememberRelease(release)
		description := release.Name
		if description == release.TagName {
			description = ""
		}
		versions = append(versions, Version{
			Source:      SourceOfficial,
			Ref:         release.TagName,
			Label:       release.TagName,
			Description: description,
			PublishedAt: release.PublishedAt,
			Prerelease:  release.Prerelease,
			Installable: true,
		})
	}
	return versions, nil
}

func (p *OfficialProvider) Resolve(ctx context.Context, ref string, platform Platform) (Asset, error) {
	if !validTag.MatchString(ref) {
		return Asset{}, fmt.Errorf("版本号 %q 无效", ref)
	}
	release, ok := p.cachedRelease(ref)
	if !ok {
		endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s",
			p.owner, p.repo, url.PathEscape(ref))
		if err := p.client.getJSON(ctx, endpoint, &release); err != nil {
			return Asset{}, err
		}
		p.rememberRelease(release)
	}

	// 按候选顺序找本机能用的资产,首选没有就退到更保守的变体。
	var rejected []string
	for _, candidate := range platform.Candidates() {
		wanted := AssetName(candidate)
		asset, found := findAsset(release, wanted)
		if !found {
			continue
		}
		digest, err := p.fetchDigest(ctx, release, ref, wanted)
		if err != nil {
			// 校验和拿不到不该让整个解析当场中止：更保守的架构变体可能带着
			// .dgst，装它总好过因为首选变体漏发校验和而完全装不上。
			rejected = append(rejected, err.Error())
			continue
		}
		return Asset{
			URL:      p.downloadURL(ref, wanted),
			Filename: wanted,
			SHA256:   digest,
			Size:     asset.Size,
		}, nil
	}
	if len(rejected) > 0 {
		return Asset{}, fmt.Errorf("版本 %s 有适配本机架构的资产，但都无法安装：%s",
			ref, strings.Join(rejected, "；"))
	}
	return Asset{}, fmt.Errorf("版本 %s 没有提供适配本机架构（%s）的资产", ref, platform.Name)
}

// rememberRelease 让用户从版本列表点击安装时复用同一份 Release 元数据，
// 无需立刻再消耗一次 GitHub API。校验和仍从发布资产独立取得。
func (p *OfficialProvider) rememberRelease(release githubRelease) {
	if release.TagName == "" {
		return
	}
	p.cacheMu.Lock()
	p.releases[release.TagName] = cachedRelease{
		release: release, expiresAt: p.now().Add(jsonCacheTTL),
	}
	p.cacheMu.Unlock()
}

func (p *OfficialProvider) cachedRelease(ref string) (githubRelease, bool) {
	p.cacheMu.RLock()
	cached, ok := p.releases[ref]
	p.cacheMu.RUnlock()
	if !ok || !p.now().Before(cached.expiresAt) {
		return githubRelease{}, false
	}
	return cached.release, true
}

// downloadURL 自行拼出发布资产的地址，而不是使用响应里的 browser_download_url。
//
// 用处不在于抵御 TLS 被攻破——那种情况下什么都保不住——而在于地址完全由已固定的
// 仓库名、已过正则的 tag 和面板自己构造的资产名拼成。这样一份被篡改的接口响应
// 最多让校验和对不上（安装失败），而没法把下载悄悄指到同一个域下的别的仓库。
func (p *OfficialProvider) downloadURL(ref, assetName string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		p.owner, p.repo, url.PathEscape(ref), url.PathEscape(assetName))
}

func findAsset(release githubRelease, name string) (githubAsset, bool) {
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func (p *OfficialProvider) fetchDigest(ctx context.Context, release githubRelease, ref, assetName string) (string, error) {
	wanted := assetName + ".dgst"
	if _, found := findAsset(release, wanted); !found {
		return "", fmt.Errorf("资产 %s 缺少校验和文件", assetName)
	}
	content, err := p.client.getText(ctx, p.downloadURL(ref, wanted))
	if err != nil {
		return "", err
	}
	return parseDigest(content, assetName)
}

// parseDigest 从 .dgst 内容里取 sha256。
// 按"第三列等于 sha256"定位,而不是依赖行号——上游若增删摘要算法,行号会失效。
func parseDigest(content, assetName string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[2] != "sha256" {
			continue
		}
		if fields[1] != assetName {
			continue
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" {
			return "", fmt.Errorf("校验和文件里的 sha256 值格式无效")
		}
		return digest, nil
	}
	return "", fmt.Errorf("校验和文件里没有 %s 的 sha256 条目", assetName)
}
