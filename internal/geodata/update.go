package geodata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/tuoro/kdae-panel/internal/atomicfile"
	"github.com/tuoro/kdae-panel/internal/daediag"
	"github.com/tuoro/kdae-panel/internal/upstream"
)

// Download 取回并校验指定来源最新的 geo 数据。
// 这一步耗时最长且不触碰任何共享状态，因此调用方可以不持有控制锁先做完。
func (m *Manager) Download(ctx context.Context, source upstream.GeoSource) (upstream.GeoData, error) {
	release, err := m.fetcher.Latest(ctx, source)
	if err != nil {
		return upstream.GeoData{}, err
	}
	data, err := m.fetcher.Fetch(ctx, release)
	if err != nil {
		return upstream.GeoData{}, err
	}
	m.logger.Info("已取得并校验 geo 数据",
		"source", source, "tag", release.Tag, "files", len(data.Files))
	return data, nil
}

// Apply 把已下载的 geo 数据装上去，并让 dae 重新读取。
// 调用方应在持有全局控制锁时调用它。
//
// 事务顺序：暂存新文件 → 把旧文件改名留作回滚点 → 原子替换 → 运行中的
// dae 用服务后端记录的 PID reload → 成功则删掉回滚点，失败则把旧文件放回去
// 并再 reload 一次。dae 未运行时不需要 reload，下次启动会直接读取新文件。
//
// 之所以必须能回滚：dae validate 察觉不到 geo 的问题，一份语义不兼容或损坏的
// geo 会让 reload 失败，而 dae 不运行时流量就不再被透明代理接管——这属于
// 静默的 fail-open，用户不会立刻察觉。
func (m *Manager) Apply(ctx context.Context, data upstream.GeoData) (Status, error) {
	service := m.inspectService(ctx)
	status := m.status(service)
	// 状态查询失败与"存在待处理的回滚点"都在这里被拦下：前者说明目标目录只能靠猜，
	// 后者说明磁盘上还留着仅存的一份旧数据，覆盖它等于把它删掉。
	if !status.Updatable {
		return Status{}, errors.New(status.Problem)
	}
	if len(data.Files) == 0 {
		return Status{}, errors.New("没有可写入的 geo 数据")
	}
	if err := validateGeoFiles(data.Files); err != nil {
		return Status{}, err
	}

	if err := cleanupTemporaryResiduals(status.Residuals); err != nil {
		return Status{}, fmt.Errorf("清理上次异常退出遗留的 Geo 暂存文件: %w", err)
	}
	targets := make(map[string]string, len(status.Files))
	for _, file := range status.Files {
		targets[file.Name] = file.TargetPath
	}
	transaction := &geoTransaction{}
	defer transaction.cleanup()

	for _, name := range Names {
		target := targets[name]
		if target == "" {
			return Status{}, fmt.Errorf("无法确定 %s 的写入位置", name)
		}
		if err := transaction.stage(name, target, data.Files[name]); err != nil {
			return Status{}, err
		}
	}
	if err := transaction.commit(); err != nil {
		return Status{}, err
	}
	if err := verifyEffectiveFiles(status.SearchPath, targets); err != nil {
		return Status{}, transaction.abort(fmt.Errorf("提交后的 geo 文件未按预期生效: %w", err))
	}

	reloaded, err := m.reload(ctx, service)
	if err != nil {
		err = daediag.ExplainGeoError(err)
		return Status{}, m.rollbackAppliedGeo(ctx, transaction,
			fmt.Errorf("新 geo 数据导致 dae 重载失败: %w", err))
	}
	cleanupErr := transaction.done()

	state := &State{
		Source:       data.Release.Source,
		Repositories: repositoriesOf(data.Release),
		Tag:          data.Release.Tag,
		UpdatedAt:    nowUTC(),
	}
	stateErr := m.writeState(state)
	if stateErr != nil {
		m.logger.Warn("记录 geo 更新状态失败", "error", stateErr)
	}
	if cleanupErr != nil {
		m.logger.Warn("清理旧 Geo 回滚点失败", "error", cleanupErr)
	}
	m.logger.Info("已更新 geo 数据",
		"source", state.Source, "tag", state.Tag, "targets", targets,
		"reloaded", reloaded)

	updated := m.Status(ctx)
	if cleanupErr != nil {
		m.logger.Warn("geo 数据已生效，但旧回滚点未能删除", "error", cleanupErr)
		updated.Warnings = append(updated.Warnings, fmt.Sprintf(
			"geo 数据已更新并生效，但旧回滚点未能删除，磁盘上暂时多一份旧副本；"+
				"它会挡住下一次更新，请在 Geo 页面确认当前文件正常后清理残留（%v）", cleanupErr))
	}
	if stateErr != nil {
		updated.Warnings = append(updated.Warnings,
			fmt.Sprintf("geo 数据已更新并生效，但更新记录写入失败（%v）", stateErr))
	}
	if cleanupErr != nil {
		updated.Warnings = append(updated.Warnings,
			fmt.Sprintf("Geo 数据已更新并生效，但旧回滚点清理失败（%v）；可在异常文件区域确认后清理", cleanupErr))
	}
	return updated, nil
}

// rollbackAppliedGeo 处理“新文件已经被 dae 读取，但事务不能结算”的共同分支。
// 不只放回磁盘文件，还必须再 reload 一次，否则磁盘恢复了，运行中的 dae 仍可能
// 使用刚才那份未被接受的数据。
func (m *Manager) rollbackAppliedGeo(ctx context.Context, transaction *geoTransaction, cause error) error {
	if restoreErr := transaction.rollback(); restoreErr != nil {
		return fmt.Errorf("%w；且旧数据未能还原：%v", cause, restoreErr)
	}
	rollbackService := m.inspectService(ctx)
	rollbackReloaded, reloadErr := m.reload(ctx, rollbackService)
	if reloadErr != nil {
		reloadErr = daediag.ExplainGeoError(reloadErr)
		return fmt.Errorf("%w；旧数据已还原但重载仍未成功：%v", cause, reloadErr)
	}
	if !rollbackReloaded {
		return fmt.Errorf("%w；旧数据已还原；dae 当前未运行，启动后会读取原数据", cause)
	}
	return fmt.Errorf("%w；已还原为原数据", cause)
}

func validateGeoFiles(files map[string][]byte) error {
	for name := range files {
		if !slices.Contains(Names, name) {
			return fmt.Errorf("未知的 geo 文件 %s，拒绝写入", name)
		}
	}
	for _, name := range Names {
		content, ok := files[name]
		if !ok {
			return fmt.Errorf("缺少 %s，geoip.dat 与 geosite.dat 必须成对更新", name)
		}
		if len(content) == 0 {
			return fmt.Errorf("%s 内容为空，拒绝写入", name)
		}
	}
	return nil
}

// verifyEffectiveFiles 确认刚写下去的那两份就是 dae 会读到的那两份。
//
// 每个文件都就地更新，正常情况下这必然成立；不成立说明搜索路径上冒出了一份优先级
// 更高的副本，此时接口报成功而 dae 读的是别的数据，只能整个回滚。
func verifyEffectiveFiles(searchPath []string, targets map[string]string) error {
	for _, file := range locate(searchPath, Names) {
		if !file.Present {
			return fmt.Errorf("%s 不存在", file.Name)
		}
		if filepath.Clean(file.Path) != filepath.Clean(targets[file.Name]) {
			return fmt.Errorf("%s 实际从 %s 生效，而不是目标位置 %s",
				file.Name, file.Path, targets[file.Name])
		}
	}
	return nil
}

// reload 只对运行中的服务传 PID（systemd 用 MainPID，procd 用实例 PID）。
// 服务未运行时文件落盘即完成；服务状态未知时保留无参数调用，以兼容没有
// ServiceController 的定制构建。
func (m *Manager) reload(ctx context.Context, service serviceSnapshot) (bool, error) {
	switch service.state {
	case ServiceStateActive:
		return true, m.reloader.ReloadPID(ctx, service.status.MainPID)
	case ServiceStateInactive:
		return false, nil
	default:
		return true, m.reloader.Reload(ctx)
	}
}

// repositoriesOf 汇总本次数据实际来自哪些仓库。
// 同一来源可能横跨多个仓库（v2fly 的 geoip 与 domain-list-community），
// 账本如实记下全部信任根，日后来源改名或换仓库时旧记录仍然读得懂。
func repositoriesOf(release upstream.GeoRelease) []string {
	repositories := make([]string, 0, len(release.Files))
	for _, file := range release.Files {
		if !slices.Contains(repositories, file.Repository) {
			repositories = append(repositories, file.Repository)
		}
	}
	slices.Sort(repositories)
	return repositories
}

// geoTransaction 管理一次多文件替换：要么两个文件都换成新的，要么都退回旧的。
//
// 分开换是不行的：geoip 和 geosite 来自同一次发布，只换掉其中一个会让 dae
// 拿着两个不同版本的规则集跑，而这种不一致既不会报错也无从察觉。
type geoTransaction struct {
	staged   []stagedFile
	cleanups []func()
}

type stagedFile struct {
	name  string
	final string
	// temp 是待启用的新文件；backup 是被顶掉的旧文件改名后的位置，旧文件不存在时为空。
	temp   string
	backup string
	// replaced 为真表示新文件已经就位。
	replaced bool
}

func (t *geoTransaction) stage(name, final string, content []byte) error {
	path, cleanup, err := atomicfile.StagePattern(filepath.Dir(final), geoTempPattern, content, geoMode)
	if err != nil {
		return fmt.Errorf("暂存 %s: %w", name, err)
	}
	t.cleanups = append(t.cleanups, cleanup)
	t.staged = append(t.staged, stagedFile{name: name, final: final, temp: path})
	return nil
}

// commit 依次把每个文件换成新的；中途失败就把已经动过的都退回去。
func (t *geoTransaction) commit() error {
	for index := range t.staged {
		file := &t.staged[index]
		final := file.final
		backup := final + rollbackSuffix
		// 上一次留下的回滚点是仅存的一份旧数据。Status 已经把"存在回滚点"报成不可
		// 更新，这里再拦一道：真撞上就说明拦截没生效，覆盖它等于把旧数据删掉。
		if _, err := os.Lstat(backup); err == nil {
			return t.abort(fmt.Errorf("%s 的回滚点 %s 已存在，拒绝覆盖", file.name, backup))
		} else if !os.IsNotExist(err) {
			return t.abort(fmt.Errorf("检查 %s 回滚点: %w", file.name, err))
		}

		// 旧文件先改名留作回滚点，而不是读进内存：geo 有几十兆，
		// 一次更新已经要在内存里放下两份新数据，再加两份旧的没有必要。
		if info, err := os.Lstat(final); err == nil {
			if !info.Mode().IsRegular() {
				return t.abort(fmt.Errorf("%s 当前路径 %s 不是普通文件，拒绝替换", file.name, final))
			}
			if err := os.Rename(final, backup); err != nil {
				return t.abort(fmt.Errorf("备份 %s: %w", file.name, err))
			}
			file.backup = backup
		} else if !os.IsNotExist(err) {
			return t.abort(err)
		}

		if err := atomicfile.Replace(file.temp, final); err != nil {
			return t.abort(fmt.Errorf("写入 %s: %w", file.name, err))
		}
		file.replaced = true
	}
	return nil
}

// abort 在 commit 中途失败时退回原样，并把"没能退回去"这件事带进错误里。
// 还原失败意味着某个文件此刻不在原位、旧数据只剩那个回滚点，
// 这是必须让用户看见的信息，不能像原先那样吞掉。
func (t *geoTransaction) abort(cause error) error {
	if err := t.rollback(); err != nil {
		return fmt.Errorf("%w；且旧数据未能还原（*%s 回滚点已保留）：%v", cause, rollbackSuffix, err)
	}
	return cause
}

// rollback 把动过的文件退回原样。
//
// 判据是 backup 而不是 replaced：commit 先把旧文件改名、再换上新文件，
// 两步之间旧文件已经不在原位而 replaced 还是假。若按 replaced 跳过，
// 这个中间态就永远不会被还原，原位置成了空缺。
func (t *geoTransaction) rollback() error {
	var failures []error
	for index := range t.staged {
		file := &t.staged[index]
		final := file.final

		if file.backup == "" {
			// 本来就没有这个文件，退回原样就是删掉新写的那份。
			if file.replaced {
				if err := os.Remove(final); err != nil && !os.IsNotExist(err) {
					failures = append(failures, fmt.Errorf("删除原本不存在的 %s: %w", file.name, err))
					continue
				}
				file.replaced = false
			}
			continue
		}
		if err := atomicfile.Replace(file.backup, final); err != nil {
			// 还原失败就保留回滚点：它是仅存的一份旧数据，绝不能顺手删掉。
			failures = append(failures, fmt.Errorf("还原 %s: %w", file.name, err))
			continue
		}
		file.backup, file.replaced = "", false
	}
	return errors.Join(failures...)
}

// done 在 reload 成功后丢弃回滚点；删不掉时保留路径并交给状态页处理。
//
// 刻意不长期保留：geo 随时可以从上游重新下载并自校验，留一份几十兆的旧副本
// 只是白占磁盘。二进制不同——kdae 的 CI 产物 90 天就过期，旧版本可能永久取不回来，
// 所以那边才需要长期回滚点。
//
// 删不掉不影响这次更新的结果：新数据已经落盘并被 dae 接受。返回错误只为把
// "磁盘上白留了一份几十兆的旧副本"如实报给用户——原先这里是 `_ = os.Remove`，
// 失败连一行日志都没有。
func (t *geoTransaction) done() error {
	var failures []error
	for index := range t.staged {
		file := &t.staged[index]
		if file.backup == "" {
			continue
		}
		if err := os.Remove(file.backup); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Errorf("删除 %s 回滚点 %s: %w", file.name, file.backup, err))
			continue
		}
		file.backup = ""
	}
	return errors.Join(failures...)
}

// cleanup 删掉还没启用的暂存文件。已经启用的不动——那是 commit 的成果。
//
// 刻意不碰回滚点：走到这里还剩下的回滚点意味着还原失败，那是仅存的一份旧数据。
// 回滚点由成功路径上的 done() 负责清理，失败路径宁可留着让人工处理。
func (t *geoTransaction) cleanup() {
	for _, cleanup := range t.cleanups {
		cleanup()
	}
}

func (m *Manager) readState() (*State, error) {
	content, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *Manager) writeState(state *State) error {
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return atomicfile.Write(m.statePath, content, 0o600)
}
