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

var callInitList = rpc.declare({
	object: 'luci',
	method: 'getInitList',
	params: [ 'name' ],
	expect: { '': {} }
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

// serviceEnabled 判断服务有没有开机自启（/etc/rc.d 下有没有对应的启动链接）。
function serviceEnabled(reply, name) {
	return ((reply || {})[name] || {}).enabled === true;
}

// 徽标是整行唯一的饱和色块，所以它必须和按钮的强调色分得开。主题里
// .label.notice 用的正是 --primary-color-high——和 .cbi-button-action 同一个色相，
// 并排放就分不清哪个是状态、哪个是能点的。运行态改用 success（绿）、停止用
// warning（琥珀），把主色整块让给按钮。
// margin 也要清掉：.label 自带 .4em 左边距，会让整列对不齐。
var BADGE_STYLE = 'display:inline-block;padding:3px 8px;border-radius:3px;font-size:12px;margin:0';

function badge() {
	return E('span', { 'style': BADGE_STYLE }, '');
}

// setBadge 就地改写徽标而不是换一个新元素：状态刷新要作用在已经挂进 DOM 的
// 节点上，重建元素还得再回头找一次父节点。
function setBadge(node, ok, okText, badText) {
	node.className = ok ? 'label success' : 'label warning';
	node.textContent = ok ? okText : badText;
}

// procd 的 restart 是异步的：rc.common 把它展开成 stop + start，stop 只是一次
// ubus service delete，发完 SIGTERM 就返回，新实例要等旧进程真正退干净才被拉起
// （dae 退出时还要卸载 eBPF 挂载点）。命令一返回就去查状态，查到的往往是中间态，
// 徽标会先闪一下「已停止」。等一秒再刷新，读到的才是结果而不是过程。
var REFRESH_DELAY = 1000;

function afterDelay(fn) {
	return new Promise(function (resolve) {
		window.setTimeout(resolve, REFRESH_DELAY);
	}).then(fn);
}

// withBusy 接管按钮的忙碌态，而不是用 ui.createHandlerFn。
//
// 后者在 finally 里无条件写 t.disabled = false，而本页的按钮是跟着服务状态走的
// （服务跑着时「启动」置灰）。两者叠在一起的结果是：点「停止」成功之后，刷新把
// 「停止」置了灰，紧接着 finally 又把它点亮——一个对已停止服务可点的停止键。
// 这里动作结束时只把 disabled 交还给 paint 去判断，不一律放开。
function withBusy(node, setBusy, fn) {
	return function () {
		setBusy(true);
		node.classList.add('spinning');
		if (node.blur)
			node.blur();
		return Promise.resolve(fn()).catch(function (err) {
			// rpc 层面直接失败时（会话过期、ubus 不通）原来什么提示都没有，
			// 按钮转一圈回到原样，看着像点了个空气。
			ui.addNotification(null, E('p', _('操作失败：%s').format(
				(err && err.message) ? err.message : String(err))), 'error');
		}).finally(function () {
			node.classList.remove('spinning');
			setBusy(false);
		});
	};
}

// setupURL 读面板启动时写下的一次性初始化链接。
// 文件不在是常态（管理员已创建，面板把它删了），不该当成错误。
function setupURL() {
	return fs.read('/var/run/kdae-panel/setup-url')
		.then(function (content) { return (content || '').trim(); })
		.catch(function () { return ''; });
}

// 开机自启的开关不跟启停按钮排在一起，而是放进「开机自启」那一列，紧挨着它自己
// 改的那个状态。最初试过在同一列里用一条竖线把两组按钮隔开，实测两头不讨好：
// 1280px 下 1px 淡灰线在描边按钮旁边根本看不见，375px 下按钮一换行，它又被甩到
// 第一行末尾，卡在两个启停按钮中间。表格的列边界本来就是现成的分隔，不用另画。
//
// 单元格里也不套 flex 容器：横向对齐由主题决定——argon 给 .td/.th 设了
// text-align:center，官方 bootstrap 是 left——而 flex 容器会把 text-align 整个
// 无视掉，于是带按钮的那两列的内容会相对表头偏到一边。直接放 inline-block 元素、
// 用外边距拉开间距，对齐就自动跟着单元格走，挤不下时照样换行。
var CELL_ITEM_STYLE = 'display:inline-block;vertical-align:middle;margin:2px 3px';

// 两个主题的 .cbi-section 都不带横向内边距：argon 明写 padding:0，靠 h3 自己的
// padding(1.25rem) 撑出卡片留白，官方 bootstrap 则两边都没有。所以直接塞进去的
// 自由块（按钮行、提示条、地址行）会紧贴卡片边缘，得自己带上和 argon 标题一致的
// 横向留白。表格不用，它本来就该横贯整张卡片。
var BLOCK_INSET = '1.25rem';

return view.extend({
	load: function () {
		return Promise.all([
			uci.load('kdae-panel'),
			callServiceList('kdae-panel'),
			callServiceList('dae'),
			setupURL(),
			// 开机自启状态取不到时退回空对象而不是让整页加载失败：它只决定一个
			// 徽标怎么显示，不该因为它把服务状态和配置表单一起拖下水。
			L.resolveDefault(callInitList('kdae-panel'), {}),
			L.resolveDefault(callInitList('dae'), {})
		]);
	},

	render: function (data) {
		var link = data[3];
		var port = uci.get('kdae-panel', 'main', 'listen_port') || '2026';
		var panelURL = location.protocol + '//' + location.hostname + ':' + port + '/';
		var services = [
			{ label: _('kdae 面板'), name: 'kdae-panel', running: serviceRunning(data[1] || {}, 'kdae-panel'),
			  enabled: serviceEnabled(data[4], 'kdae-panel') },
			{ label: _('dae'), name: 'dae', running: serviceRunning(data[2] || {}, 'dae'),
			  enabled: serviceEnabled(data[5], 'dae') }
		];

		// 状态块不做成表单项：CBI 的 option 只为"编辑一个 UCI 值"而生，
		// 硬塞按钮和链接要整个覆写它的 render，一旦 LuCI 改动内部约定就会碎。
		// 直接建 DOM 再和表单拼起来，行为确定得多。
		var statusView = (function () {
			// dae 的开机自启默认是关的：软件包只 enable 面板自己，而面板装完 dae
			// 也刻意不自动启动它（透明代理配置不当会切断你当前的连接）。这句必须
			// 说出来——否则用户会在下一次重启路由器时才发现。
			// 它只在 dae 确实没自启时出现：自启之后同一句话就是纯噪声，长期挂在
			// 那儿只会让人以后连真正的告警也不看。
			// 用 warning 而不是 notice：主题里 .alert-message.notice 是一层近乎无色的
			// 灰渐变，实测看着像个禁用的输入框，完全不像在提醒什么。琥珀色也和
			// 「已停止」徽标同色，整页的色彩语言统一为"琥珀＝需要你处理"。
			var daeHint = E('div', { 'class': 'alert-message warning',
				'style': 'margin:14px ' + BLOCK_INSET + ' 0' },
				_('dae 尚未设为开机自启，路由器重启后不会自动接管流量。确认配置无误后，点上面的「设为自启」。'));

			function buildRow(service) {
				var running = service.running;
				var enabled = service.enabled;
				var busy = false;

				// 行内样式写在 style 上而不是 class 上：paint() 会整体改写 className，
				// 写进 class 的话每刷新一次就掉一次。
				var runBadge = badge();
				var bootText = E('span', { 'style': CELL_ITEM_STYLE }, '');
				var startButton = E('button', { 'class': 'cbi-button', 'style': CELL_ITEM_STYLE }, _('启动'));
				var restartButton = E('button', { 'class': 'cbi-button', 'style': CELL_ITEM_STYLE }, _('重启'));
				var stopButton = E('button', { 'class': 'cbi-button', 'style': CELL_ITEM_STYLE }, _('停止'));
				var bootButton = E('button', { 'class': 'cbi-button', 'style': CELL_ITEM_STYLE }, '');

				// 不适用的按钮置灰而不是藏起来：列宽保持稳定，而且旁边的徽标已经
				// 把原因写在脸上了。强调色只给"此刻该点的那个"——改版前整行最扎眼
				// 的按钮恰好是空操作（服务跑着，「启动」却满色可点）。
				function setButton(node, usable, activeClass) {
					// className 整体改写会把 withBusy 加的 spinning 一起抹掉，
					// 而刷新恰好发生在动作还没结束的时候。
					var spinning = node.classList.contains('spinning');
					node.disabled = busy || !usable;
					node.className = (usable ? activeClass : 'cbi-button') + (spinning ? ' spinning' : '');
				}

				function paint() {
					setBadge(runBadge, running, _('运行中'), _('已停止'));
					bootText.textContent = enabled ? _('已启用') : _('未启用');
					bootButton.textContent = enabled ? _('取消自启') : _('设为自启');
					bootButton.disabled = busy;
					setButton(startButton, !running, 'cbi-button cbi-button-action');
					setButton(restartButton, running, 'cbi-button cbi-button-action');
					setButton(stopButton, running, 'cbi-button cbi-button-reset');
					if (service.name === 'dae')
						daeHint.style.display = enabled ? 'none' : '';
				}

				function setBusy(value) {
					busy = value;
					paint();
				}

				function refresh() {
					return Promise.all([
						L.resolveDefault(callServiceList(service.name), {}),
						L.resolveDefault(callInitList(service.name), {})
					]).then(function (reply) {
						running = serviceRunning(reply[0] || {}, service.name);
						enabled = serviceEnabled(reply[1], service.name);
						paint();
					});
				}

				// rpcd 把 init 脚本的退出码原样带回 result。忽略它就会出现
				// 「已启动 dae」的绿色提示和「已停止」的徽标同时挂在屏幕上——
				// dae 尚未安装时点「启动」正是这种情况，脚本会带着说明退出 1。
				function bind(node, actionName, okText, failText) {
					node.addEventListener('click', withBusy(node, setBusy, function () {
						return callInitAction(service.name, actionName).then(function (ok) {
							ui.addNotification(null, E('p', ok ? okText : failText), ok ? 'info' : 'error');
							return afterDelay(refresh);
						});
					}));
				}

				bind(startButton, 'start', _('已启动 %s').format(service.name),
					_('启动 %s 失败，请查看系统日志').format(service.name));
				bind(restartButton, 'restart', _('已重启 %s').format(service.name),
					_('重启 %s 失败，请查看系统日志').format(service.name));
				bind(stopButton, 'stop', _('已停止 %s').format(service.name),
					_('停止 %s 失败，请查看系统日志').format(service.name));

				bootButton.addEventListener('click', withBusy(bootButton, setBusy, function () {
					var wanted = enabled ? 'disable' : 'enable';
					return callInitAction(service.name, wanted).then(function (ok) {
						ui.addNotification(null, E('p', ok
							? (wanted === 'enable'
								? _('%s 已设为开机自启').format(service.name)
								: _('%s 已取消开机自启').format(service.name))
							: _('修改 %s 的开机自启失败').format(service.name)), ok ? 'info' : 'error');
						// 自启只改 /etc/rc.d 里的软链，没有中间态，不必等。
						return refresh();
					});
				}));

				paint();

				return E('div', { 'class': 'tr' }, [
					E('div', { 'class': 'td', 'data-title': _('服务') }, E('strong', {}, service.label)),
					E('div', { 'class': 'td', 'data-title': _('运行状态') }, runBadge),
					E('div', { 'class': 'td', 'data-title': _('开机自启') }, [ bootText, bootButton ]),
					E('div', { 'class': 'td', 'data-title': _('操作') }, [ startButton, restartButton, stopButton ])
				]);
			}

			// 用 LuCI 自己的表格标记（.table/.tr/.th/.td）而不是一条 flex：
			// 对齐由表格保证，主题换了也不用管，「系统 → 启动项」用的就是这套。
			var table = E('div', { 'class': 'table' }, [
				E('div', { 'class': 'tr table-titles' }, [
					E('div', { 'class': 'th', 'style': 'width:20%' }, _('服务')),
					E('div', { 'class': 'th', 'style': 'width:15%' }, _('运行状态')),
					E('div', { 'class': 'th', 'style': 'width:28%' }, _('开机自启')),
					E('div', { 'class': 'th' }, _('操作'))
				])
			].concat(services.map(buildRow)));

			// 按钮和地址说的是同一件事——去面板——所以并成一行，不再一个在表格上头
			// 一个在下头各说一遍。按钮是动作，地址是参照：能选中复制，反代或改过
			// listen_port 时也能一眼核对面板到底在哪儿。
			// 面板拒绝被 iframe 嵌入（CSP frame-ancestors 'none'），只能新标签打开；
			// 这句话不占版面，挂在按钮的 title 上。
			var openRow = E('div', { 'style': 'display:flex;align-items:center;gap:12px;' +
				'flex-wrap:wrap;margin:0 ' + BLOCK_INSET + ' 14px' }, [
				E('button', {
					'class': 'cbi-button cbi-button-action important',
					'title': _('在新标签页打开面板'),
					'click': function () { window.open(panelURL, '_blank', 'noopener'); }
				}, _('打开面板')),
				E('a', { 'href': panelURL, 'target': '_blank', 'rel': 'noopener' }, panelURL)
			]);

			var body = [
				// h3 直接当 .cbi-section 的头一个孩子，不套任何容器：argon 把 h3 本身
				// 当卡片标题栏（自带背景与 1.25rem 内边距），包一层再和按钮排成 flex，
				// 标题会撑满整行、把按钮挤到下一行的最左边——那正是第一版在真机上的
				// 样子。让主题按它自己的方式渲染标题，最稳。
				E('h3', {}, _('服务状态')),
				// 「打开面板」是这个页面存在的理由，紧跟标题下面当主操作，
				// 不和一堆启停按钮挤在同一条流水线上。
				openRow,
				// 窄屏下表格会被挤扁，给它一层横向滚动兜底而不是让按钮溢出到卡片外。
				E('div', { 'style': 'overflow-x:auto' }, table),
				daeHint
			];

			if (link) {
				body.push(E('div', {
					'class': 'alert-message warning',
					'style': 'margin:14px ' + BLOCK_INSET + ' 0'
				}, [
					E('p', {}, E('strong', {}, _('尚未创建管理员'))),
					E('p', {}, _('打开下面的一次性链接完成初始化，创建成功后链接立即失效：')),
					E('p', {}, E('a', { 'href': link, 'target': '_blank', 'rel': 'noopener' }, link))
				]));
			}

			// padding-bottom 是补 argon 的：它把 .cbi-section 的 padding 设成 0，卡片
			// 内的留白全靠各个子元素自己带，而最后一个子元素（表格或提示条）都不带
			// 下边距，于是内容直接压在卡片下边框上。行内样式优先级高于主题那条
			// padding:0，只覆盖 bottom 这一边。
			// margin-bottom 则是状态块和下面「设置」卡片之间的间距。
			return E('div', { 'class': 'cbi-section',
				'style': 'padding-bottom:1rem;margin-bottom:24px' }, body);
		})();

		// Map 不带标题：它会把标题渲染在自己那整块表单的最上方，而状态块是拼在
		// 表单前面的，结果就是页面标题跑到了内容中间。标题和说明改由下面自己出，
		// 用的是和 LuCI 完全相同的标记（h2[name=content] + div.cbi-map-descr），
		// 主题样式一致。
		var m = new form.Map('kdae-panel');

		var s = m.section(form.NamedSection, 'main', 'kdae-panel', _('设置'));

		var o = s.option(form.Flag, 'enabled', _('开机自启'),
			_('关闭后面板不会随系统启动，已经在跑的实例不受影响。'));
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'listen_addr', _('监听地址'),
			_('0.0.0.0 表示接受本机与局域网连接。'));
		o.datatype = 'ipaddr';
		o.default = '0.0.0.0';

		o = s.option(form.Value, 'listen_port', _('监听端口'),
			_('默认 2026，避开 daed 等同类软件包占用的 2023。'));
		o.datatype = 'port';
		o.default = '2026';

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
			return E([], [
				E('h2', { 'name': 'content' }, _('kdae 面板')),
				E('div', { 'class': 'cbi-map-descr' },
					_('面板负责 dae 的配置编排、版本管理与日志；dae 的可执行文件由面板下载安装，不经 opkg。')),
				statusView,
				formView
			]);
		});
	}
});
