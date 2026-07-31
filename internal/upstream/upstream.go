// Package upstream 从上游发现可安装的 dae 版本并取回对应的发布资产。
//
// 支持两个来源,它们的"版本"含义不同,界面需要如实区分:
//   - official: daeuniverse/dae 的 GitHub Release,版本即 tag,可回到任意历史版本,
//     校验和来自随包发布的 .dgst 文件。
//   - kdae: olicesx/dae 的 kdae 分支没有 Release,只有 CI 构建产物。版本即一次
//     成功的构建,校验和来自 GitHub Actions API 的 digest 字段。产物 90 天过期,
//     过期后无法安装。
//
// 两个来源的清单接口都无需任何凭据。kdae 的产物下载需要认证,因此改由
// nightly.link 重定向到 GitHub 自己的签名地址;由于校验和是经另一条 TLS 连接
// 从 GitHub API 独立取得的,该重定向服务无法篡改内容,只影响可用性。
package upstream

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

type Source string

const (
	SourceOfficial Source = "official"
	SourceKdae     Source = "kdae"
)

// Version 是一个可供选择安装的版本。
type Version struct {
	Source      Source    `json:"source"`
	Ref         string    `json:"ref"`
	Label       string    `json:"label"`
	Description string    `json:"description,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	Installable bool      `json:"installable"`
	// Note 说明不可安装的原因,如产物已过期。
	Note string `json:"note,omitempty"`
	// ExpiresAt 仅 kdae 有意义:CI 产物的过期时间。
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// Asset 是某个版本针对本机架构的可下载文件。
type Asset struct {
	URL      string
	Filename string
	SHA256   string
	Size     int64
	// Nested 为真时,下载得到的 zip 里还套着一层发布 zip(Actions 产物即如此)。
	Nested bool
}

type Provider interface {
	Source() Source
	List(ctx context.Context, limit int) ([]Version, error)
	Resolve(ctx context.Context, ref string, platform Platform) (Asset, error)
}

var (
	// tag 允许字母数字与 .-_,排除斜杠与点号开头,避免拼进 URL 或路径时越界。
	validTag   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	validRunID = regexp.MustCompile(`^[0-9]{1,19}$`)
)

// ParseSource 把外部输入映射成已知来源。
func ParseSource(value string) (Source, error) {
	switch Source(value) {
	case SourceOfficial:
		return SourceOfficial, nil
	case SourceKdae:
		return SourceKdae, nil
	default:
		return "", fmt.Errorf("未知的上游来源 %q", value)
	}
}

// Registry 按来源持有 provider，并与它们共用同一个 HTTP 客户端，
// 使连接池与代理认知在列版本、解析资产、下载发布包之间保持一致。
type Registry struct {
	providers map[Source]Provider
	client    *httpClient
}

func NewRegistry(providers ...Provider) *Registry {
	return newRegistry(newHTTPClient(), providers...)
}

func newRegistry(client *httpClient, providers ...Provider) *Registry {
	registry := &Registry{
		providers: make(map[Source]Provider, len(providers)),
		client:    client,
	}
	for _, provider := range providers {
		registry.providers[provider.Source()] = provider
	}
	return registry
}

// NewDefaultRegistry 构造指向上游默认仓库的 provider 集合。
func NewDefaultRegistry() *Registry {
	return NewDefaultRegistryWithGitHubToken(emptyGitHubTokenSource{})
}

// NewDefaultRegistryWithGitHubToken 构造可即时读取面板凭据的默认 provider 集合。
func NewDefaultRegistryWithGitHubToken(source GitHubTokenSource) *Registry {
	client := newHTTPClientWithTokenSource(source)
	return newRegistry(client,
		NewOfficialProvider(client, "daeuniverse", "dae"),
		NewKdaeProvider(client, "olicesx", "dae", "kdae", "build.yml"),
	)
}

func (r *Registry) Provider(source Source) (Provider, error) {
	provider, ok := r.providers[source]
	if !ok {
		return nil, fmt.Errorf("未知的上游来源 %q", source)
	}
	return provider, nil
}

func (r *Registry) List(ctx context.Context, source Source, limit int) ([]Version, error) {
	provider, err := r.Provider(source)
	if err != nil {
		return nil, err
	}
	return provider.List(ctx, limit)
}

func (r *Registry) Resolve(ctx context.Context, source Source, ref string, platform Platform) (Asset, error) {
	provider, err := r.Provider(source)
	if err != nil {
		return Asset{}, err
	}
	return provider.Resolve(ctx, ref, platform)
}
