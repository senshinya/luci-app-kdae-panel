import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const moduleURL = new URL('../htdocs/luci-static/resources/kdae_setup_urls.js', import.meta.url);
const panelURL = new URL('../htdocs/luci-static/resources/view/kdae-panel/panel.js', import.meta.url);

async function loadHelper() {
	const source = await readFile(moduleURL, 'utf8');
	const factory = new Function('baseclass', 'URL', source);
	return factory({ extend: value => value }, URL);
}

test('parses every setup URL line without merging fragments', async () => {
	const helper = await loadHelper();
	const first = 'http://10.0.0.2:2026/setup#bootstrap=first';
	const second = 'http://10.0.0.3:2026/setup#bootstrap=second';
	const result = helper.parse(`${first}\r\n${second}\n`, '10.0.0.2');

	assert.equal(result.present, true);
	assert.equal(result.invalidCount, 0);
	assert.deepEqual(result.links, [first, second]);
});

test('keeps valid links, reports invalid lines, and prioritizes the current host', async () => {
	const helper = await loadHelper();
	const current = 'http://[fd00::2]:2026/setup#bootstrap=current';
	const other = 'http://192.168.1.1:2026/setup#bootstrap=other';
	const result = helper.parse(`${other}\nnot a URL\nftp://example.test/setup\n${current}\n`, '[fd00::2]');

	assert.equal(result.present, true);
	assert.equal(result.invalidCount, 2);
	assert.deepEqual(result.links, [current, other]);
});

test('distinguishes an empty file from a non-empty file without valid links', async () => {
	const helper = await loadHelper();
	assert.deepEqual(helper.parse('  \n', 'router.lan'), {
		present: false,
		links: [],
		invalidCount: 0,
	});
	assert.deepEqual(helper.parse('broken\n', 'router.lan'), {
		present: true,
		links: [],
		invalidCount: 1,
	});
});

test('the LuCI page load path returns the parsed URL list', async () => {
	const helper = await loadHelper();
	const source = await readFile(panelURL, 'utf8');
	const first = 'http://10.0.0.2:2026/setup#bootstrap=first';
	const second = 'http://10.0.0.3:2026/setup#bootstrap=second';
	const previousL = globalThis.L;
	const previousLocation = globalThis.location;
	globalThis.L = { resolveDefault: promise => promise };
	globalThis.location = { hostname: '10.0.0.2', protocol: 'http:' };
	try {
		const factory = new Function('view', 'form', 'uci', 'fs', 'rpc', 'ui', 'kdae_setup_urls', source);
		const page = factory(
			{ extend: value => value },
			{},
			{ load: async () => {}, get: () => null },
			{ read: async () => `${first}\n${second}\n` },
			{ declare: () => async () => ({}) },
			{},
			helper,
		);
		const data = await page.load();
		assert.deepEqual(data[3].links, [first, second]);
	} finally {
		globalThis.L = previousL;
		globalThis.location = previousLocation;
	}
});
