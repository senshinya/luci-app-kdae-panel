# Correctness and Recovery Fixes Design

## Goal

修复 OpenWrt 数据目录权限、procd 状态误判、Geo 双文件目标选择，以及 LuCI 多条初始化链接四类正确性问题，并保证临时故障解除后操作可以重新成功。

## Invariants

1. OpenWrt 数据目录固定为 `/etc/kdae-panel`，所有数据文件参数只引用这个常量，且拒绝将专用目录作为符号链接使用。UCI 里的 `data_dir` 只被读来做一致性检查，绝不用于拼路径。
2. 只有成功且可解析的 procd 查询可以证明服务 inactive/disabled。查询错误不得伪装成服务状态，也不得被缓存。
3. `geoip.dat` 与 `geosite.dat` 必须来自同一次下载、写入同一目标目录，并在 reload 前确认二者都是搜索路径上的有效副本。
4. setup-url 文件是“每个非空行一个 URL”的列表。LuCI 必须逐条校验、排序和渲染，不能静默吞掉非空但无有效链接的文件。
5. 一次性初始化链接只列局域网地址：IPv4 收 RFC1918 与 CGNAT，IPv6 收 ULA 与全局单播。链接的 fragment 里带着 bootstrap token，公网 IPv4 一律排除。

## Recovery Rules

- UCI 中遗留的 `data_dir` 与固定路径不一致时拒绝启动并写 syslog，指明数据仍在原处以及如何清理该项；这条刻意是启动阻塞，静默忽略会让用户面对一个空面板而看不到原因。
- procd 每次状态读取都重新执行命令；外部故障恢复后下一次调用自然恢复。
- Geo 新事务开始前放回遗留的 `*.kdae-panel-previous`。普通文件可原子恢复；目录、符号链接等异常对象不被覆盖，并返回准确路径。
- Geo commit 不得覆盖既存 recovery 文件。验证或 reload 失败时回滚；回滚失败保留 recovery 文件供下一次调用恢复。
- Geo reload 成功后删除 recovery 文件；删不掉不影响本次更新结果，作为告警报出。刻意不引入 committed marker：它要区分的两种残留状态都会被同一次更新覆盖掉，代价配不上收益。
- setup URL 直到管理员创建成功才删除；格式错误不消耗 bootstrap token，页面显示可操作错误。

## Out of Scope

- 不迁移或兼容旧的 OpenWrt 自定义数据目录。
- 不重构全部后台任务生命周期。
- 不为 Geo 引入完整 WAL；本次使用现有 recovery 文件完成可重试恢复。

## Verification

- Go 单元测试、race、vet。
- Vue typecheck、Vitest、生产构建。
- Playwright E2E。
- OpenWrt init shell 语法检查及 LuCI 解析逻辑静态验证。
