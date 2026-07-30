'use strict';
'require view';
'require form';
'require uci';
'require fs';
'require rpc';
'require ui';

var callServiceList = rpc.declare({
	object: 'service',
	method: 'list',
	params: [ 'name' ],
	expect: { '': {} }
});

var callInitAction = rpc.declare({
	object: 'luci',
	method: 'setInitAction',
	params: [ 'name', 'action' ],
	expect: { result: false }
});

// serviceRunning 从 procd 的实例表判断服务是否在跑。
// 查不到实例就是没跑——procd 只在实例被定义后才列出它。
function serviceRunning(reply, name) {
	var service = reply[name] || {};
	var instances = service.instances || {};
	for (var key in instances)
		if (instances[key].running)
			return true;
	return false;
}

function statusBadge(running) {
	return E('span', {
		'class': running ? 'label notice' : 'label warning',
		'style': 'padding:2px 8px;border-radius:3px'
	}, running ? _('运行中') : _('已停止'));
}

// setupURL 读面板启动时写下的一次性初始化链接。
// 文件不在是常态（管理员已创建，面板把它删了），不该当成错误。
function setupURL() {
	return fs.read('/var/run/kdae-panel/setup-url')
		.then(function (content) { return (content || '').trim(); })
		.catch(function () { return ''; });
}

return view.extend({
	load: function () {
		return Promise.all([
			uci.load('kdae-panel'),
			callServiceList('kdae-panel'),
			callServiceList('dae'),
			setupURL()
		]);
	},

	render: function (data) {
		var panelRunning = serviceRunning(data[1] || {}, 'kdae-panel');
		var daeRunning = serviceRunning(data[2] || {}, 'dae');
		var link = data[3];
		var port = uci.get('kdae-panel', 'main', 'listen_port') || '2023';

		// 状态块不做成表单项：CBI 的 option 只为"编辑一个 UCI 值"而生，
		// 硬塞按钮和链接要整个覆写它的 render，一旦 LuCI 改动内部约定就会碎。
		// 直接建 DOM 再和表单拼起来，行为确定得多。
		var statusView = (function () {
			var rows = [
				[ _('kdae 面板'), panelRunning, 'kdae-panel' ],
				[ _('dae'), daeRunning, 'dae' ]
			].map(function (row) {
				return E('div', { 'style': 'display:flex;align-items:center;gap:12px;margin-bottom:8px' }, [
					E('strong', { 'style': 'min-width:8em' }, row[0]),
					statusBadge(row[1]),
					E('button', {
						'class': 'cbi-button cbi-button-apply',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'restart').then(function () {
								ui.addNotification(null, E('p', _('已请求重启 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('重启')),
					E('button', {
						'class': 'cbi-button cbi-button-reset',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'stop').then(function () {
								ui.addNotification(null, E('p', _('已停止 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('停止')),
					E('button', {
						'class': 'cbi-button cbi-button-action',
						'click': ui.createHandlerFn(null, function (name) {
							return callInitAction(name, 'start').then(function () {
								ui.addNotification(null, E('p', _('已启动 %s').format(name)), 'info');
							});
						}, row[2])
					}, _('启动'))
				]);
			});

			rows.push(E('div', { 'style': 'margin-top:16px' }, [
				E('button', {
					'class': 'cbi-button cbi-button-action important',
					'click': function () {
						window.open(location.protocol + '//' + location.hostname + ':' + port + '/', '_blank');
					}
				}, _('打开面板'))
			]));

			// 面板拒绝被 iframe 嵌入（CSP frame-ancestors 'none'），因此新标签打开。
			rows.push(E('p', { 'style': 'margin-top:8px' },
				E('em', {}, _('面板在新标签页打开，地址为 %s。').format(
					location.protocol + '//' + location.hostname + ':' + port + '/'))));

			if (link) {
				rows.push(E('div', {
					'class': 'alert-message warning',
					'style': 'margin-top:16px'
				}, [
					E('p', {}, E('strong', {}, _('尚未创建管理员'))),
					E('p', {}, _('打开下面的一次性链接完成初始化，创建成功后链接立即失效：')),
					E('p', {}, E('a', { 'href': link, 'target': '_blank' }, link))
				]));
			}

			return E('div', { 'class': 'cbi-section' }, [
				E('h3', {}, _('服务状态')),
				E('div', {}, rows)
			]);
		})();

		var m = new form.Map('kdae-panel', _('kdae 面板'),
			_('面板负责 dae 的配置编排、版本管理与日志；dae 的可执行文件由面板下载安装，不经 opkg。'));

		var s = m.section(form.NamedSection, 'main', 'kdae-panel', _('设置'));

		var o = s.option(form.Flag, 'enabled', _('开机自启'),
			_('关闭后面板不会随系统启动，已经在跑的实例不受影响。'));
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'listen_addr', _('监听地址'),
			_('0.0.0.0 表示接受本机与局域网连接。'));
		o.datatype = 'ipaddr';
		o.default = '0.0.0.0';

		o = s.option(form.Value, 'listen_port', _('监听端口'));
		o.datatype = 'port';
		o.default = '2023';

		o = s.option(form.Value, 'data_dir', _('数据目录'),
			_('数据库、配置备份、状态文件与 dae 本地版本库的位置。' +
			  '不要改到 /var 或 /tmp 下——那里是内存文件系统，重启即空。'));
		o.default = '/etc/kdae-panel';

		o = s.option(form.Value, 'dae_binary', _('dae 可执行文件'),
			_('面板与 dae 的启动脚本读的是同一个值，不要单独修改启动脚本。'));
		o.default = '/usr/bin/dae';

		o = s.option(form.Value, 'dae_config', _('dae 配置文件'),
			_('geo 数据也放在这个文件所在的目录。'));
		o.default = '/etc/dae/config.dae';

		o = s.option(form.Flag, 'enable_dae_install', _('由面板管理 dae 版本'),
			_('允许面板下载、安装、切换与回滚 dae。关闭后版本管理页不可用。'));
		o.default = '1';

		o = s.option(form.Flag, 'enable_geo_update', _('由面板管理 geo 数据'),
			_('允许一键更新 geoip.dat / geosite.dat，更新只触发 dae reload 不重启。'));
		o.default = '1';

		o = s.option(form.Value, 'trusted_proxies', _('可信代理'),
			_('可以转发客户端地址和协议的代理 CIDR，逗号分隔。'));
		o.default = '127.0.0.0/8,::1/128';

		o = s.option(form.Value, 'session_ttl', _('会话有效期'),
			_('形如 12h、30m。'));
		o.default = '12h';

		o = s.option(form.Flag, 'secure_cookie', _('Cookie 仅 HTTPS'),
			_('通过 HTTPS 反向代理访问时打开。'));
		o.default = '0';

		// 配置全部通过命令行参数传给面板，改完必须重启才生效。
		// 重启不在这里做：init 脚本的 service_triggers 里注册了
		// procd_add_reload_trigger "kdae-panel"，LuCI 的「保存并应用」提交
		// UCI 后 procd 会自己触发 reload，而 reload_service 就是 restart。
		return m.render().then(function (formView) {
			return E([], [ statusView, formView ]);
		});
	}
});
