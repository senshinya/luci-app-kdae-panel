package upstream

import "context"

// PackageRepoOwner / PackageRepoName 是 OpenWrt 软件包（kdae-panel 与
// luci-app-kdae-panel 两个 ipk）的发布仓库坐标。
//
// 单独成文件而不是并进 panel.go：那份文件属于上游，本分支每次合并都要动它。
// 这里的两个常量是本分支独有的事实，放在上游不会碰的文件里，合并冲突为零。
//
// 为什么不能复用 PanelRepoOwner/PanelRepoName：那是上游 tuoro/kdae-panel 的
// 坐标，它发布的是 systemd 部署用的裸二进制，版本线也和本软件包不是一回事。
// 拿它的 tag 去和 ipk 的版本比，得到的"有新版本"既装不上也不该装。
const (
	PackageRepoOwner = "senshinya"
	PackageRepoName  = "luci-app-kdae-panel"
)

// ReleasesURL 是某个仓库最新发布的网页地址，供界面上的"查看发布说明"使用。
//
// 由后端给出而不是让前端写死：前端写死意味着两套部署各存一份链接，
// 而"检查的是哪个仓库"这件事只有后端知道，早晚会出现提示指向 A、链接指向 B。
func ReleasesURL(owner, repo string) string {
	return "https://github.com/" + owner + "/" + repo + "/releases/latest"
}

// NewPackageReleaseChecker 返回查询 ipk 发布仓库最新正式 tag 的检查器。
//
// 走带 token 的客户端：版本检查每 6 小时一次看似稀疏，但 GitHub 对匿名请求
// 按出口 IP 限流，同一个宽带出口下的多台路由器会互相挤掉配额。
func NewPackageReleaseChecker(source GitHubTokenSource) func(context.Context) (string, error) {
	client := newHTTPClientWithTokenSource(source)
	return func(ctx context.Context) (string, error) {
		return latestPanelRelease(ctx, client, PackageRepoOwner, PackageRepoName)
	}
}
