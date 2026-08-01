# Correctness and Recovery Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复四个已确认的正确性问题，并为每条失败路径补上可重试恢复测试。

**Architecture:** 保持现有 host、geodata、app 与 OpenWrt LuCI 边界。服务状态通过错误传播表达 unknown；Geo 在现有原子替换事务上增加优先级选择、提交后断言和 recovery 预检；setup-url 保持逐行文本协议。

**Tech Stack:** Go 1.25、OpenWrt rc.common/procd、LuCI JavaScript、Vue/Vitest、Playwright。

---

### Task 1: 固定 OpenWrt 数据目录

**Files:**
- Modify: `openwrt/kdae-panel/files/kdae-panel.init`
- Modify: `openwrt/kdae-panel/files/kdae-panel.config`
- Modify: `openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js`

- [x] **Step 1: 建立失败检查**

```sh
rg -n "data_dir" openwrt/kdae-panel/files openwrt/luci-app-kdae-panel
```

预期：init、默认 UCI 和 LuCI 表单均命中。

- [x] **Step 2: 固定 init 路径**

定义 `PANEL_DATA_DIR=/etc/kdae-panel`，删除 `config_get data_dir`。拒绝直接符号链接，检查 `mkdir`、`chmod` 的返回值，所有数据文件参数只引用常量。

- [x] **Step 3: 删除配置入口**

删除默认 UCI 的 `option data_dir` 和 LuCI `form.Value('data_dir', ...)`；遗留 UCI 值不再被读取。

- [x] **Step 4: 验证**

```sh
sh -n openwrt/kdae-panel/files/kdae-panel.init
! rg -n "config_get data_dir|option data_dir|'data_dir', _" openwrt
```

预期：shell 语法通过，旧配置入口无命中。

### Task 2: 让 procd 状态查询正确传播错误

**Files:**
- Modify: `internal/host/procd.go`
- Modify: `internal/host/procd_test.go`

- [x] **Step 1: 写失败测试**

增加 ubus 执行失败、非法 JSON、`enabled` 非预期退出码测试；退出码 1 仍为 disabled；同一 manager 第一次失败、第二次成功时不得锁存错误。

- [x] **Step 2: 运行测试确认失败**

```sh
go test -count=1 ./internal/host -run 'TestProcdStatus(Rejects|Recovers|Stopped)' -v
```

- [x] **Step 3: 实现三值查询**

将 `instance` 改为 `(ubusInstance, bool, error)`，将 `unitFileState` 改为 `(string, error)`。命令错误使用 `command.Describe`；只有成功的空服务表表示 inactive，只有退出码 1 表示 disabled；`os.Stat` 只把 `ENOENT` 当 not-found。

- [x] **Step 4: 运行调用方测试**

```sh
go test -count=1 ./internal/host ./internal/daeinstall ./internal/geodata ./internal/app
```

### Task 3: Geo 统一目标与事务恢复

**Files:**
- Modify: `internal/geodata/locate.go`
- Modify: `internal/geodata/update.go`
- Modify: `internal/geodata/geodata_test.go`

- [x] **Step 1: 写目标选择失败测试**

增加 split-directory 两种方向测试；无论 `Names` 顺序如何，都选择有效文件所在目录中搜索优先级最高者。

- [x] **Step 2: 写输入与恢复失败测试**

增加缺少文件、未知文件、既存 recovery、异常 recovery/final 类型，以及“第一次恢复失败、解除阻塞后第二次成功”的测试。

- [x] **Step 3: 运行测试确认失败**

```sh
go test -count=1 ./internal/geodata -run 'Test(TargetDir|UpdateRejects|UpdateRecovers|UpdateVerifies)' -v
```

- [x] **Step 4: 实现目标选择和输入校验**

让 `targetDir` 接收 `searchPath` 并按目录优先级选择；Apply 按 `Names` 固定顺序 stage，要求两个已知文件存在且非空，拒绝未知名称。

- [x] **Step 5: 实现 recovery 预检和覆盖保护**

新事务前扫描固定 recovery 路径。普通 backup 原子恢复到缺失或普通 final；非普通对象报错且不修改。commit 在 recovery 已存在时中止。

- [x] **Step 6: 实现提交后断言**

commit 后、reload 前重新 locate，断言两个有效路径都位于 TargetDir；失败使用事务回滚路径。

- [x] **Step 7: 运行集成测试**

```sh
go test -count=1 ./internal/geodata ./internal/daeinstall ./internal/app
```

### Task 4: 修复 setup URL 列表和 IPv6 wildcard

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `openwrt/luci-app-kdae-panel/htdocs/luci-static/resources/view/kdae-panel/panel.js`

- [x] **Step 1: 写后端失败测试**

增加 `[::]` fallback、ULA、CGNAT、全局单播、link-local 排除、去重与稳定排序测试。

- [x] **Step 2: 运行后端测试确认失败**

```sh
go test -count=1 ./internal/app -run 'TestBootstrapSetupURL' -v
```

- [x] **Step 3: 实现后端地址规则**

统一 wildcard 判断；`::` fallback 到 `::1`；枚举非 loopback、非 unspecified、非 multicast、非 link-local 的单播地址；使用 `net.JoinHostPort`、去重和排序。

- [x] **Step 4: 修复 LuCI 消费者**

读取结果改为 `{ present, links, invalidCount }`。按行解析并只接受 http/https，按当前 hostname 优先排序，每条 URL 单独渲染；非空文件没有合法 URL 时显示错误。

- [x] **Step 5: 验证 app 与前端**

```sh
go test -count=1 ./internal/app
npm run typecheck --prefix web
npm test --prefix web
npm run build --prefix web
```

### Task 5: 全量验证

**Files:**
- Verify all modified files

- [x] **Step 1: 格式与静态检查**

```sh
gofmt -w internal/host/procd.go internal/host/procd_test.go internal/geodata/locate.go internal/geodata/update.go internal/geodata/geodata_test.go internal/app/app.go internal/app/app_test.go
go vet ./...
```

- [x] **Step 2: Go 与 race**

```sh
go test -count=1 ./...
go test -count=1 -race ./...
```

- [x] **Step 3: 前端与 E2E**

```sh
npm run typecheck --prefix web
npm test --prefix web
npm run build --prefix web
node --test openwrt/luci-app-kdae-panel/tests/*.mjs
npm test --prefix e2e
```

- [x] **Step 4: 工作区确认**

```sh
git diff --check
git status --short
```

预期：没有格式错误；只有本计划内文件发生变化。
