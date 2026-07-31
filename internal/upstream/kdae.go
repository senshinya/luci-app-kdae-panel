package upstream

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// KdaeProvider 把 kdae 分支上每一次成功的 CI 构建当作一个可安装版本。
// 该分支没有 Release,产物只存在于 Actions,因此:
//   - 清单与校验和取自 GitHub API(匿名可用)
//   - 下载走 nightly.link 重定向,因为 Actions 产物的直链需要认证
//   - 产物 90 天过期,过期的构建标记为不可安装而不是假装能装
type KdaeProvider struct {
	client   *httpClient
	owner    string
	repo     string
	branch   string
	workflow string
	now      func() time.Time

	cacheMu      sync.RWMutex
	verifiedRuns map[string]time.Time
}

func NewKdaeProvider(client *httpClient, owner, repo, branch, workflow string) *KdaeProvider {
	return &KdaeProvider{
		client: client, owner: owner, repo: repo, branch: branch, workflow: workflow,
		now: time.Now, verifiedRuns: make(map[string]time.Time),
	}
}

func (p *KdaeProvider) Source() Source {
	return SourceKdae
}

type workflowRun struct {
	ID         int64  `json:"id"`
	HeadSHA    string `json:"head_sha"`
	HeadBranch string `json:"head_branch"`
	Event      string `json:"event"`
	// Path 形如 ".github/workflows/build.yml"。List 靠 URL 隐式限定了工作流，
	// 而 Resolve 按 run id 取产物没有这层限制，必须显式核对。
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
	Conclusion string    `json:"conclusion"`
	HeadCommit struct {
		Message string `json:"message"`
	} `json:"head_commit"`
	HeadRepository struct {
		FullName string `json:"full_name"`
	} `json:"head_repository"`
}

type workflowRuns struct {
	Runs []workflowRun `json:"workflow_runs"`
}

// trustworthy 重新校验每一条 run，不信任查询参数的过滤结果。
//
// 关键在于 ?branch= 过滤的是 head_branch，而对 pull_request 事件，head_branch
// 是 PR 源分支名。若上游哪天给 build.yml 加上 pull_request 触发，任何人 fork
// 之后把自己的分支命名为 kdae 再开 PR，其构建就会出现在这个列表里——而它的
// digest 是 GitHub 对攻击者产物如实计算的，哈希校验会完整通过。
// 因此必须确认产物确实构建自本仓库自己的分支。
func (p *KdaeProvider) trustworthy(run workflowRun) bool {
	if run.Conclusion != "success" || run.HeadBranch != p.branch {
		return false
	}
	// 字段缺失一律当作不可信，不因为解析不到就放行。
	if !strings.EqualFold(run.HeadRepository.FullName, p.owner+"/"+p.repo) {
		return false
	}
	if run.Path != ".github/workflows/"+p.workflow {
		return false
	}
	switch run.Event {
	case "push", "workflow_dispatch":
		return true
	default:
		return false
	}
}

type runArtifacts struct {
	Artifacts []struct {
		Name      string    `json:"name"`
		SizeBytes int64     `json:"size_in_bytes"`
		Digest    string    `json:"digest"`
		Expired   bool      `json:"expired"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"artifacts"`
}

func (p *KdaeProvider) List(ctx context.Context, limit int) ([]Version, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/%s/runs?branch=%s&status=success&per_page=%d",
		p.owner, p.repo, p.workflow, p.branch, limit)
	var runs workflowRuns
	if err := p.client.getJSON(ctx, endpoint, &runs); err != nil {
		return nil, err
	}

	now := time.Now()
	versions := make([]Version, 0, len(runs.Runs))
	for _, run := range runs.Runs {
		if !p.trustworthy(run) {
			continue
		}
		ref := strconv.FormatInt(run.ID, 10)
		p.rememberVerifiedRun(ref)
		// 产物保留 90 天,过期后无法下载。这里按创建时间推算,
		// 精确的过期时间在 Resolve 时以 API 返回的 expires_at 为准。
		expiresAt := run.CreatedAt.Add(90 * 24 * time.Hour)
		expired := now.After(expiresAt)
		version := Version{
			Source:      SourceKdae,
			Ref:         ref,
			Label:       shortSHA(run.HeadSHA),
			Description: firstLine(run.HeadCommit.Message),
			PublishedAt: run.CreatedAt,
			Installable: !expired,
			ExpiresAt:   &expiresAt,
		}
		if expired {
			version.Note = "构建产物已超过 90 天保留期，无法下载"
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (p *KdaeProvider) Resolve(ctx context.Context, ref string, platform Platform) (Asset, error) {
	if !validRunID.MatchString(ref) {
		return Asset{}, fmt.Errorf("构建编号 %q 无效", ref)
	}
	// 安装请求里的构建编号可以是任意值，不一定来自本面板列出的清单，
	// 因此这里必须重新核对来源，而不是依赖 List 已经过滤过。
	if err := p.verifyRun(ctx, ref); err != nil {
		return Asset{}, err
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%s/artifacts?per_page=100",
		p.owner, p.repo, ref)
	var artifacts runArtifacts
	if err := p.client.getJSON(ctx, endpoint, &artifacts); err != nil {
		return Asset{}, err
	}

	for _, candidate := range platform.Candidates() {
		wanted := AssetName(candidate)
		for _, artifact := range artifacts.Artifacts {
			if artifact.Name != wanted {
				continue
			}
			if artifact.Expired {
				return Asset{}, fmt.Errorf("构建 %s 的产物已过期，无法安装", ref)
			}
			digest, err := parseArtifactDigest(artifact.Digest)
			if err != nil {
				return Asset{}, fmt.Errorf("构建 %s 的产物 %s：%w", ref, wanted, err)
			}
			return Asset{
				// nightly.link 只发重定向，字节仍来自 GitHub；
				// 校验和已由上面的 API 独立取得，因此它无法篡改内容。
				URL: fmt.Sprintf("https://nightly.link/%s/%s/actions/runs/%s/%s",
					p.owner, p.repo, ref, wanted),
				Filename: wanted,
				SHA256:   digest,
				Size:     artifact.SizeBytes,
				Nested:   true, // Actions 产物是 zip 套 zip
			}, nil
		}
	}
	return Asset{}, fmt.Errorf("构建 %s 没有提供适配本机架构（%s）的产物", ref, platform.Name)
}

// verifyRun 单独取回该构建并核对它确实产自本仓库自己的分支。
func (p *KdaeProvider) verifyRun(ctx context.Context, ref string) error {
	if p.runRecentlyVerified(ref) {
		return nil
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs/%s", p.owner, p.repo, ref)
	var run workflowRun
	if err := p.client.getJSON(ctx, endpoint, &run); err != nil {
		return err
	}
	if !p.trustworthy(run) {
		return fmt.Errorf("构建 %s 不是 %s/%s 的 %s 分支产物，拒绝安装", ref, p.owner, p.repo, p.branch)
	}
	p.rememberVerifiedRun(ref)
	return nil
}

// List 已逐项完成了与 Resolve 相同的来源核对。短期记住结论可省去用户点击安装时
// 对同一个 run 的重复 API 请求；任意外部传入且没在清单出现过的编号仍会重新核验。
func (p *KdaeProvider) rememberVerifiedRun(ref string) {
	p.cacheMu.Lock()
	p.verifiedRuns[ref] = p.now().Add(jsonCacheTTL)
	p.cacheMu.Unlock()
}

func (p *KdaeProvider) runRecentlyVerified(ref string) bool {
	p.cacheMu.RLock()
	expiresAt, ok := p.verifiedRuns[ref]
	p.cacheMu.RUnlock()
	return ok && p.now().Before(expiresAt)
}

// parseArtifactDigest 解析 "sha256:<64 位十六进制>"。
// 老产物可能没有 digest 字段,此时拒绝安装而不是跳过校验。
func parseArtifactDigest(value string) (string, error) {
	algorithm, digest, found := strings.Cut(strings.TrimSpace(value), ":")
	if !found || algorithm != "sha256" {
		return "", fmt.Errorf("缺少可用的 sha256 校验和，拒绝安装")
	}
	digest = strings.ToLower(digest)
	if len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" {
		return "", fmt.Errorf("校验和格式无效")
	}
	return digest, nil
}

func shortSHA(value string) string {
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(value), "\n")
	if len(line) > 120 {
		return line[:120] + "…"
	}
	return line
}
