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
// dae 使用 systemd MainPID reload → 成功则删掉回滚点，失败则把旧文件放回去
// 并再 reload 一次。dae 未运行时不需要 reload，下次启动会直接读取新文件。
//
// 之所以必须能回滚：dae validate 察觉不到 geo 的问题，一份语义不兼容或损坏的
// geo 会让 reload 失败，而 dae 不运行时流量就不再被透明代理接管——这属于
// 静默的 fail-open，用户不会立刻察觉。
func (m *Manager) Apply(ctx context.Context, data upstream.GeoData) (Status, error) {
	status := m.Status(ctx)
	if !status.Updatable {
		return Status{}, errors.New(status.Problem)
	}
	if len(data.Files) == 0 {
		return Status{}, errors.New("没有可写入的 geo 数据")
	}
	service := m.inspectService(ctx)

	transaction := &geoTransaction{directory: status.TargetDir}
	defer transaction.cleanup()

	for name, content := range data.Files {
		if len(content) == 0 {
			return Status{}, fmt.Errorf("%s 内容为空，拒绝写入", name)
		}
		if err := transaction.stage(name, content); err != nil {
			return Status{}, err
		}
	}
	if err := transaction.commit(); err != nil {
		return Status{}, err
	}

	reloaded, err := m.reload(ctx, service)
	if err != nil {
		err = daediag.ExplainGeoError(err)
		// 换上去 dae 不认，退回原样并让它重新读回旧数据。
		restoreErr := transaction.rollback()
		if restoreErr != nil {
			return Status{}, fmt.Errorf(
				"新 geo 数据导致 dae 重载失败（%w），且旧数据未能还原：%v", err, restoreErr)
		}
		rollbackService := m.inspectService(ctx)
		rollbackReloaded, reloadErr := m.reload(ctx, rollbackService)
		if reloadErr != nil {
			reloadErr = daediag.ExplainGeoError(reloadErr)
			return Status{}, fmt.Errorf(
				"新 geo 数据导致 dae 重载失败（%w），旧数据已还原但重载仍未成功：%v", err, reloadErr)
		}
		if !rollbackReloaded {
			return Status{}, fmt.Errorf(
				"新 geo 数据导致 dae 重载失败（%w），旧数据已还原；dae 当前未运行，启动后会读取原数据", err)
		}
		return Status{}, fmt.Errorf("新 geo 数据导致 dae 重载失败，已还原为原数据：%w", err)
	}
	transaction.done()

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
	if stateErr != nil {
		updated.Warnings = append(updated.Warnings,
			fmt.Sprintf("geo 数据已更新并生效，但更新记录写入失败（%v）", stateErr))
	}
	return updated, nil
}

// reload 只对运行中的 systemd 服务传 MainPID。服务未运行时文件落盘即完成；
// 服务状态未知时保留无参数调用，以兼容没有 ServiceController 的定制构建。
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

		// 旧文件先改名留作回滚点，而不是读进内存：geo 有几十兆，
		// 一次更新已经要在内存里放下两份新数据，再加两份旧的没有必要。
		if _, err := os.Stat(final); err == nil {
			backup := final + ".kdae-panel-previous"
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
		return fmt.Errorf("%w；且旧数据未能还原（回滚点保留在 %s 下的 *.kdae-panel-previous）：%v",
			cause, t.directory, err)
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
					failures = append(failures, err)
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
func (t *geoTransaction) done() {
	for index := range t.staged {
		if backup := t.staged[index].backup; backup != "" {
			_ = os.Remove(backup)
			t.staged[index].backup = ""
		}
	}
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
