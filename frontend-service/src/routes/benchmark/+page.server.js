import { readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

export function load() {
	const benchmarkDir = resolve(__dirname, '..', '..', '..', '..', 'benchmark');

	// Load HRIS results
	const hrisPath = resolve(benchmarkDir, 'results.json');
	let hrisReport = {};
	try {
		const raw = readFileSync(hrisPath, 'utf-8');
		hrisReport = JSON.parse(raw);
	} catch (/** @type {any} */ e) {
		hrisReport = { error: `Failed to load results.json: ${e?.message ?? String(e)}` };
	}

	// Load Smart City results
	const smartcityPath = resolve(benchmarkDir, 'results-smartcity.json');
	let smartcityReport = {};
	try {
		const raw = readFileSync(smartcityPath, 'utf-8');
		smartcityReport = JSON.parse(raw);
	} catch (/** @type {any} */ e) {
		smartcityReport = { error: `Failed to load results-smartcity.json: ${e?.message ?? String(e)}` };
	}

	return {
		reports: {
			hris: hrisReport,
			smartcity: smartcityReport,
		}
	};
}
