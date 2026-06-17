import { readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

export function load() {
	// Read results.json from the benchmark directory (4 levels up from src/routes/benchmark/ to project root)
	const resultsPath = resolve(__dirname, '..', '..', '..', '..', 'benchmark', 'results.json');
	let report = {};
	try {
		const raw = readFileSync(resultsPath, 'utf-8');
		report = JSON.parse(raw);
	} catch (/** @type {any} */ e) {
		report = { error: `Failed to load results.json: ${e?.message ?? String(e)}` };
	}

	return { report };
}