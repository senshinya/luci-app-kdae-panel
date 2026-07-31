package upstream

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// dae 查找的两个 geo 数据文件名。
const (
	GeoIPName   = "geoip.dat"
	GeoSiteName = "geosite.dat"
)

// maxGeoBytes 限制单个 geo 文件的下载量。
// 实测 geoip.dat 在 17~23MB、geosite.dat 在 2~11MB（随来源而异），
// 且都在逐年增长，64MB 留足余量的同时挡住无限响应体。
const maxGeoBytes = 64 << 20

// GeoSource 是 geo 数据的来源。
//
// 两个来源的规则集不是同一套，切换会改变 geosite: 规则匹配的域名集合，
// 而 dae 照样 validate 通过、照样运行——因此必须由用户显式选择，
// 界面也要把差异讲清楚，不能替他做主。
type GeoSource string

const (
	// GeoSourceLoyalsoldier 是中文社区常用的重编排规则集，每天发布。
	GeoSourceLoyalsoldier GeoSource = "loyalsoldier"
	// GeoSourceV2fly 与 dae 发布包自带的是同一套数据——dae 的 CI 正是从这里
	// 取 geo 打进发布包的（geosite.dat 就是 dlc.dat 改名）。
	GeoSourceV2fly GeoSource = "v2fly"
)

// ParseGeoSource 把外部输入映射成已知来源。
func ParseGeoSource(value string) (GeoSource, error) {
	switch GeoSource(value) {
	case GeoSourceLoyalsoldier:
		return GeoSourceLoyalsoldier, nil
	case GeoSourceV2fly:
		return GeoSourceV2fly, nil
	}
	if source := GeoSource(value); source != "" {
		if _, ok := customGeoSourceID(source); ok {
			return source, nil
		}
	}
	return "", fmt.Errorf("未知的 geo 数据来源 %q", value)
}

// geoOrigin 是某个来源里一个 geo 文件的出处。
//
// 分文件记录而不是"一个来源一个仓库"，是因为 v2fly 把两个文件放在两个仓库里，
// 各自独立发布、tag 也不同。
type geoOrigin struct {
	owner string
	repo  string
	// asset 是上游的资产名，可能与 dae 使用的文件名不同。
	asset string
}

func (o geoOrigin) repository() string { return o.owner + "/" + o.repo }

// GeoSourceInfo 描述一个来源，供界面呈现与选择。
type GeoSourceInfo struct {
	Source GeoSource `json:"source"`
	Label  string    `json:"label"`
	// Repositories 如实列出信任根，可能不止一个。
	Repositories []string `json:"repositories"`
	Note         string   `json:"note"`
	Custom       bool     `json:"custom,omitempty"`
}

// GeoFile 是一个 geo 数据文件的元数据。
type GeoFile struct {
	// Name 是 dae 使用的文件名（geoip.dat / geosite.dat）。
	Name string `json:"name"`
	// Repository 是它的出处，同一来源下两个文件可能来自不同仓库。
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	// Asset 是它在上游发布里的资产名，可能与 Name 不同（v2fly 的 dlc.dat）。
	Asset       string    `json:"asset"`
	Size        int64     `json:"size"`
	PublishedAt time.Time `json:"publishedAt"`
	// 自定义来源的直链只在 Latest → Fetch 的内存事务中传递，不进入 API 或账本。
	downloadURL string
	digestURL   string
}

// GeoRelease 描述一次可安装的 geo 数据。
//
// geo 没有"回到某个历史版本"的用例，用户要的永远是最新的一份，
// 因此只解析 latest，不提供版本列表。
type GeoRelease struct {
	Source GeoSource `json:"source"`
	// Tag 是展示用的版本标识。多仓库来源没有单一 tag，此时按文件名拼出来。
	Tag         string    `json:"tag"`
	PublishedAt time.Time `json:"publishedAt"`
	Files       map[string]GeoFile
}

// GeoData 是取回并逐一校验过 sha256 的 geo 文件内容。
type GeoData struct {
	Release GeoRelease
	Files   map[string][]byte
}

// GeoProvider 从 GitHub Release 取 geo 数据。
//
// 与 dae 发布包不同，这些 .dat 是裸资产而不是 zip，因此不走 FetchBundle 那条
// 解包路径；校验和是每个资产各自的 .sha256sum，格式是 coreutils 的
// "<哈希>  <文件名>"，与 dae 的 .dgst（三列，含算法名）也不一样。
type GeoProvider struct {
	client  *httpClient
	source  GeoSource
	label   string
	note    string
	origins map[string]geoOrigin
}

// Info 返回来源的展示信息，其中 Repositories 如实列出全部信任根。
func (p *GeoProvider) Info() GeoSourceInfo {
	repositories := make([]string, 0, len(p.origins))
	for _, origin := range p.origins {
		if !slices.Contains(repositories, origin.repository()) {
			repositories = append(repositories, origin.repository())
		}
	}
	slices.Sort(repositories)
	return GeoSourceInfo{
		Source:       p.source,
		Label:        p.label,
		Repositories: repositories,
		Note:         p.note,
	}
}

// NewGeoRegistry 构造指向上游默认仓库的 geo provider 集合。
func NewGeoRegistry() *GeoRegistry {
	return NewGeoRegistryWithGitHubToken(emptyGitHubTokenSource{})
}

func NewGeoRegistryWithGitHubToken(source GitHubTokenSource) *GeoRegistry {
	client := newHTTPClientWithTokenSource(source)
	registry := newGeoRegistry(
		&GeoProvider{
			client: client,
			source: GeoSourceLoyalsoldier,
			label:  "Loyalsoldier 规则集",
			note: "中文社区常用的重编排规则集，每天发布。" +
				"分类比 v2fly 更细（如 geosite:gfw、geosite:greatfire），" +
				"但同名分类所含域名与 v2fly 不同。",
			origins: map[string]geoOrigin{
				GeoIPName:   {"Loyalsoldier", "v2ray-rules-dat", GeoIPName},
				GeoSiteName: {"Loyalsoldier", "v2ray-rules-dat", GeoSiteName},
			},
		},
		&GeoProvider{
			client: client,
			source: GeoSourceV2fly,
			label:  "v2fly 官方（与 dae 发布包同源）",
			note: "dae 的 CI 正是从这两个仓库取 geo 打进发布包的，" +
				"因此它与首次安装写入的数据属于同一规则体系；若当前仍使用随包数据，" +
				"更新到它不会换来源。geosite.dat 在上游名为 dlc.dat。",
			origins: map[string]geoOrigin{
				GeoIPName:   {"v2fly", "geoip", GeoIPName},
				GeoSiteName: {"v2fly", "domain-list-community", "dlc.dat"},
			},
		},
	)
	registry.customClient = newCustomHTTPClient()
	registry.custom, _ = openCustomGeoStore("")
	return registry
}

// OpenGeoRegistryWithGitHubToken 在内置来源之外加载可由面板维护的自定义来源。
func OpenGeoRegistryWithGitHubToken(source GitHubTokenSource, customSourcePath string) (*GeoRegistry, error) {
	registry := NewGeoRegistryWithGitHubToken(source)
	store, err := openCustomGeoStore(customSourcePath)
	if err != nil {
		return nil, err
	}
	registry.custom = store
	return registry, nil
}

// GeoRegistry 按来源持有 provider。
type GeoRegistry struct {
	providers map[GeoSource]geoProvider
	order     []GeoSource
	custom    *customGeoStore
	// 自定义来源永远使用不带 GitHub Token 的独立客户端。
	customClient *httpClient
}

type geoProvider interface {
	Info() GeoSourceInfo
	Latest(context.Context) (GeoRelease, error)
	Fetch(context.Context, GeoRelease) (GeoData, error)
}

func newGeoRegistry(providers ...*GeoProvider) *GeoRegistry {
	registry := &GeoRegistry{providers: make(map[GeoSource]geoProvider, len(providers))}
	for _, provider := range providers {
		registry.providers[provider.source] = provider
		registry.order = append(registry.order, provider.source)
	}
	return registry
}

// Sources 按呈现顺序返回全部来源。
func (r *GeoRegistry) Sources() []GeoSourceInfo {
	custom := r.CustomSources()
	infos := make([]GeoSourceInfo, 0, len(r.order)+len(custom))
	for _, source := range r.order {
		infos = append(infos, r.providers[source].Info())
	}
	for _, source := range custom {
		infos = append(infos, (&customGeoProvider{client: r.customClient, source: source}).Info())
	}
	return infos
}

func (r *GeoRegistry) provider(source GeoSource) (geoProvider, error) {
	provider, ok := r.providers[source]
	if ok {
		return provider, nil
	}
	if r.custom != nil {
		if custom, found := r.custom.get(source); found {
			return &customGeoProvider{client: r.customClient, source: custom}, nil
		}
	}
	return nil, fmt.Errorf("未知的 geo 数据来源 %q", source)
}

func (r *GeoRegistry) Latest(ctx context.Context, source GeoSource) (GeoRelease, error) {
	provider, err := r.provider(source)
	if err != nil {
		return GeoRelease{}, err
	}
	return provider.Latest(ctx)
}

func (r *GeoRegistry) Fetch(ctx context.Context, release GeoRelease) (GeoData, error) {
	if _, custom := customGeoSourceID(release.Source); custom {
		return (&customGeoProvider{client: r.customClient}).Fetch(ctx, release)
	}
	provider, err := r.provider(release.Source)
	if err != nil {
		return GeoData{}, err
	}
	return provider.Fetch(ctx, release)
}

func (r *GeoRegistry) CustomSources() []CustomGeoSource {
	if r.custom == nil {
		return nil
	}
	return r.custom.list()
}

func (r *GeoRegistry) CreateCustomSource(source CustomGeoSource) (CustomGeoSource, error) {
	if r.custom == nil {
		return CustomGeoSource{}, errors.New("自定义 geo 来源存储未初始化")
	}
	return r.custom.create(source)
}

func (r *GeoRegistry) UpdateCustomSource(id string, source CustomGeoSource) (CustomGeoSource, error) {
	if r.custom == nil {
		return CustomGeoSource{}, errors.New("自定义 geo 来源存储未初始化")
	}
	return r.custom.update(id, source)
}

func (r *GeoRegistry) DeleteCustomSource(id string) error {
	if r.custom == nil {
		return errors.New("自定义 geo 来源存储未初始化")
	}
	return r.custom.delete(id)
}

// Latest 解析每个文件各自最新的发布，确认资产与校验和文件都在。
func (p *GeoProvider) Latest(ctx context.Context) (GeoRelease, error) {
	// 同一仓库只查一次：Loyalsoldier 的两个文件在同一次发布里。
	cache := map[string]githubRelease{}
	result := GeoRelease{Source: p.source, Files: make(map[string]GeoFile, len(p.origins))}

	for name, origin := range p.origins {
		release, ok := cache[origin.repository()]
		if !ok {
			fetched, err := p.latestRelease(ctx, origin)
			if err != nil {
				return GeoRelease{}, err
			}
			cache[origin.repository()], release = fetched, fetched
		}

		asset, found := findAsset(release, origin.asset)
		if !found {
			return GeoRelease{}, fmt.Errorf("%s 的发布 %s 里没有 %s",
				origin.repository(), release.TagName, origin.asset)
		}
		// 校验和缺失一律拒绝，与安装二进制同一条纪律，没有跳过开关。
		if _, found := findAsset(release, origin.asset+".sha256sum"); !found {
			return GeoRelease{}, fmt.Errorf("%s 的 %s 没有校验和文件，拒绝下载",
				origin.repository(), origin.asset)
		}
		if asset.Size > maxGeoBytes {
			return GeoRelease{}, fmt.Errorf("%s 声明大小 %d 字节，超过 %d 限制",
				origin.asset, asset.Size, maxGeoBytes)
		}
		result.Files[name] = GeoFile{
			Name:        name,
			Repository:  origin.repository(),
			Tag:         release.TagName,
			Asset:       origin.asset,
			Size:        asset.Size,
			PublishedAt: release.PublishedAt,
		}
	}
	result.Tag, result.PublishedAt = summarise(result.Files)
	return result, nil
}

func (p *GeoProvider) latestRelease(ctx context.Context, origin geoOrigin) (githubRelease, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", origin.owner, origin.repo)
	var release githubRelease
	if err := p.client.getJSON(ctx, endpoint, &release); err != nil {
		return githubRelease{}, err
	}
	if !validTag.MatchString(release.TagName) {
		return githubRelease{}, fmt.Errorf("%s 的发布 tag %q 无效", origin.repository(), release.TagName)
	}
	return release, nil
}

// summarise 把每个文件的 tag 归纳成一个展示用的版本标识。
// 两个文件同属一次发布时就用那个 tag；来自不同仓库时如实分别列出，
// 不编造一个看起来统一、实则不存在的版本号。
func summarise(files map[string]GeoFile) (string, time.Time) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)

	var latest time.Time
	tags := make([]string, 0, len(names))
	distinct := map[string]bool{}
	for _, name := range names {
		file := files[name]
		if file.PublishedAt.After(latest) {
			latest = file.PublishedAt
		}
		distinct[file.Tag] = true
		tags = append(tags, strings.TrimSuffix(name, ".dat")+" "+file.Tag)
	}
	if len(distinct) == 1 {
		for tag := range distinct {
			return tag, latest
		}
	}
	return strings.Join(tags, " / "), latest
}

// Fetch 下载并逐个校验 geo 文件。任一文件校验不过就整体失败，不产出任何内容。
func (p *GeoProvider) Fetch(ctx context.Context, release GeoRelease) (GeoData, error) {
	data := GeoData{Release: release, Files: make(map[string][]byte, len(release.Files))}
	for name, file := range release.Files {
		origin, ok := p.origins[name]
		if !ok {
			return GeoData{}, fmt.Errorf("来源 %s 不提供 %s", p.source, name)
		}
		digest, err := p.fetchDigest(ctx, origin, file.Tag)
		if err != nil {
			return GeoData{}, err
		}
		limit := int64(maxGeoBytes)
		if file.Size > 0 && file.Size < limit {
			// 已知大小时收紧上限，留一点余量容忍元数据差异。
			limit = file.Size + (1 << 20)
		}
		content, err := p.client.download(ctx, downloadURL(origin, file.Tag, origin.asset), limit)
		if err != nil {
			return GeoData{}, err
		}
		if err := verifyDigest(content, digest); err != nil {
			return GeoData{}, fmt.Errorf("%s: %w", origin.asset, err)
		}
		data.Files[name] = content
	}
	return data, nil
}

func (p *GeoProvider) fetchDigest(ctx context.Context, origin geoOrigin, tag string) (string, error) {
	content, err := p.client.getText(ctx, downloadURL(origin, tag, origin.asset+".sha256sum"))
	if err != nil {
		return "", err
	}
	return parseSHA256Sum(content, origin.asset)
}

// downloadURL 自行拼出资产地址，与官方发布走同一条纪律：
// 不采信接口响应里的 browser_download_url，被篡改的响应最多让校验和对不上。
func downloadURL(origin geoOrigin, tag, assetName string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		origin.owner, origin.repo, url.PathEscape(tag), url.PathEscape(assetName))
}

// parseSHA256Sum 解析 coreutils sha256sum 的输出格式："<哈希>  <文件名>"。
//
// 刻意不复用 parseDigest：dae 的 .dgst 是三列（哈希、文件名、算法名），
// 用它解析这里的两列内容会一律报"没有对应条目"，错误信息与真正的原因南辕北辙。
//
// 文件名一栏容忍不匹配：这个文件本来就是按资产名单独取回的，
// 一份文件对一个校验和，不存在选错行的可能。
func parseSHA256Sum(content, assetName string) (string, error) {
	for line := range strings.Lines(content) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		digest := strings.ToLower(fields[0])
		if len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" {
			continue
		}
		return digest, nil
	}
	return "", fmt.Errorf("%s 的校验和文件里没有有效的 sha256 值", assetName)
}
