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

// geoRecoverySuffix 是被顶掉的旧文件改名后的后缀，也是崩溃后唯一的恢复线索。
const geoRecoverySuffix = ".kdae-panel-previous"

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
	if service.inspectErr != nil {
		return Status{}, errors.New(status.Problem)
	}
	if len(data.Files) == 0 {
		return Status{}, errors.New("没有可写入的 geo 数据")
	}
	if err := validateGeoFiles(data.Files); err != nil {
		return Status{}, err
	}
	if err := recoverGeoSearchPath(status.SearchPath, Names); err != nil {
		return Status{}, err
	}
	// recovery 可能让此前缺失的正式文件重新出现，目标目录必须基于恢复后的
	// 有效文件重算；沿用恢复前的 fallback 会把 recovery 永久遗落在低优先级目录。
	status = m.status(service)
	if !status.Updatable {
		return Status{}, errors.New(status.Problem)
	}
	transaction := &geoTransaction{directory: status.TargetDir}
	defer transaction.cleanup()

	for _, name := range Names {
		content := data.Files[name]
		if err := transaction.stage(name, content); err != nil {
			return Status{}, err
		}
	}
	if err := transaction.commit(); err != nil {
		return Status{}, err
	}
	if err := verifyEffectiveFiles(status.SearchPath, status.TargetDir); err != nil {
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
	m.logger.Info("已更新 geo 数据",
		"source", state.Source, "tag", state.Tag, "directory", status.TargetDir,
		"reloaded", reloaded)

	updated := m.Status(ctx)
	if cleanupErr != nil {
		m.logger.Warn("geo 数据已生效，但旧回滚点未能删除", "error", cleanupErr)
		updated.Warnings = append(updated.Warnings, fmt.Sprintf(
			"geo 数据已更新并生效，但旧回滚点未能删除，磁盘上暂时多一份旧副本；下次更新会先把它放回再覆盖为新数据（%v）", cleanupErr))
	}
	if stateErr != nil {
		updated.Warnings = append(updated.Warnings,
			fmt.Sprintf("geo 数据已更新并生效，但更新记录写入失败（%v）", stateErr))
	}
	return updated, nil
}

func recoverGeoSearchPath(searchPath, names []string) error {
	visited := make([]string, 0, len(searchPath))
	for _, directory := range searchPath {
		directory = filepath.Clean(directory)
		if slices.Contains(visited, directory) {
			continue
		}
		visited = append(visited, directory)
		// systemd 的 ProtectHome 会故意把 dae 可能读取的 root asset 目录
		// 隐藏给面板。这里无法创建过面板 recovery，跳过是契约的一部分；
		// 其他目录的权限错误仍必须上报，不能把真实故障一概吞掉。
		if directory == filepath.Clean(SandboxHiddenDir) && SandboxHidesHome() {
			continue
		}
		if err := recoverGeoBackups(directory, names); err != nil {
			return err
		}
	}
	return nil
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

// recoverGeoBackups 在新事务开始前放回上一次被中断留下的回滚点。
//
// 只有一种状态需要恢复：进程死在"旧文件已改名、新文件还没换入"之间，此时正式
// 文件根本不存在，dae 一重启就会因为读不到 geo 而起不来。reload 之后残留的回滚点
// 长得一模一样，也会被这里放回去——那只是让 dae 短暂退回上一版规则集，而紧接着
// 的这次更新马上就会覆盖它。为了区分这两者去维护一套持久化的提交标记，代价远大
// 于它买到的东西：geo 随时可以从上游重新下载并自校验。
//
// 先把全部路径检查完再动手，避免一个异常目录让恢复做到一半才停；真正替换时若仍
// 遇到 I/O 故障，下一次调用会重新扫描剩余 recovery 文件并继续。
func recoverGeoBackups(directory string, names []string) error {
	type recovery struct{ name, final, backup string }
	actions := make([]recovery, 0, len(names))
	for _, name := range names {
		final := filepath.Join(directory, name)
		backup := final + geoRecoverySuffix
		backupInfo, err := os.Lstat(backup)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("检查 %s recovery 文件: %w", name, err)
			}
			continue
		}
		if !backupInfo.Mode().IsRegular() {
			return fmt.Errorf("%s recovery 路径 %s 不是普通文件，拒绝覆盖", name, backup)
		}
		finalInfo, err := os.Lstat(final)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("检查 %s 当前文件: %w", name, err)
		}
		if err == nil && !finalInfo.Mode().IsRegular() {
			return fmt.Errorf("%s 当前路径 %s 不是普通文件；修正后可重新更新，recovery 文件保留在 %s",
				name, final, backup)
		}
		actions = append(actions, recovery{name: name, final: final, backup: backup})
	}
	for _, action := range actions {
		if err := atomicfile.Replace(action.backup, action.final); err != nil {
			return fmt.Errorf("恢复上次中断留下的 %s: %w", action.name, err)
		}
	}
	return nil
}

func verifyEffectiveFiles(searchPath []string, target string) error {
	for _, file := range locate(searchPath, Names) {
		if !file.Present {
			return fmt.Errorf("%s 不存在", file.Name)
		}
		if filepath.Clean(filepath.Dir(file.Path)) != filepath.Clean(target) {
			return fmt.Errorf("%s 实际从 %s 生效，而不是目标目录 %s", file.Name, file.Path, target)
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
	directory string
	staged    []stagedFile
	cleanups  []func()
}

type stagedFile struct {
	name string
	// temp 是待启用的新文件；backup 是被顶掉的旧文件改名后的位置，旧文件不存在时为空。
	temp   string
	backup string
	// replaced 为真表示新文件已经就位。
	replaced bool
}

func (t *geoTransaction) stage(name string, content []byte) error {
	path, cleanup, err := atomicfile.Stage(t.directory, content, geoMode)
	if err != nil {
		return fmt.Errorf("暂存 %s: %w", name, err)
	}
	t.cleanups = append(t.cleanups, cleanup)
	t.staged = append(t.staged, stagedFile{name: name, temp: path})
	return nil
}

// commit 依次把每个文件换成新的；中途失败就把已经动过的都退回去。
func (t *geoTransaction) commit() error {
	for index := range t.staged {
		file := &t.staged[index]
		final := filepath.Join(t.directory, file.name)
		backup := final + geoRecoverySuffix
		// 上一次留下的回滚点是仅存的一份旧数据。Apply 开头已经沿搜索路径恢复过
		// 一轮，这里再拦一道：真撞上就说明恢复没生效，覆盖它等于把旧数据删掉。
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
		return fmt.Errorf("%w；且旧数据未能还原（回滚点保留在 %s 下的 *%s）：%v",
			cause, t.directory, geoRecoverySuffix, err)
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
		final := filepath.Join(t.directory, file.name)

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

// done 在 reload 成功后丢弃回滚点。
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
		backup := t.staged[index].backup
		if backup == "" {
			continue
		}
		if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Errorf("删除 %s 的回滚点: %w", t.staged[index].name, err))
			continue
		}
		t.staged[index].backup = ""
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
