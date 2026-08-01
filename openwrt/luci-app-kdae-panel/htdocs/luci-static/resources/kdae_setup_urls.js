'use strict';
'require baseclass';

function normalizeHostname(hostname) {
	return String(hostname || '').replace(/^\[|\]$/g, '').toLowerCase();
}

return baseclass.extend({
	parse: function (content, currentHostname) {
		var raw = String(content || '');
		if (!raw.trim())
			return { present: false, links: [], invalidCount: 0 };

		var current = normalizeHostname(currentHostname);
		var entries = [];
		var seen = {};
		var invalidCount = 0;
		raw.split(/\r?\n/).forEach(function (value) {
			var line = value.trim();
			if (!line)
				return;
			try {
				var parsed = new URL(line);
				if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') ||
					!parsed.hostname || parsed.username || parsed.password) {
					invalidCount++;
					return;
				}
				if (seen[parsed.href])
					return;
				seen[parsed.href] = true;
				entries.push({
					value: line,
					hostname: normalizeHostname(parsed.hostname),
				});
			} catch (err) {
				invalidCount++;
			}
		});

		entries.sort(function (left, right) {
			var leftCurrent = left.hostname === current ? 0 : 1;
			var rightCurrent = right.hostname === current ? 0 : 1;
			if (leftCurrent !== rightCurrent)
				return leftCurrent - rightCurrent;
			return left.value.localeCompare(right.value);
		});
		return {
			present: true,
			links: entries.map(function (entry) { return entry.value; }),
			invalidCount: invalidCount,
		};
	},
});
