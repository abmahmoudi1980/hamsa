/**
 * Builds the self-hosted Vazirmatn web fonts.
 *
 * The Flutter client ships three TTFs (~123 KB each). Serving those directly
 * would cost ~370 KB of the initial payload on exactly the Iranian mobile
 * networks this product targets, so this script:
 *   1. subsets each face to the glyphs the Persian UI can render, and
 *   2. converts TTF → WOFF2 (typically a further ~50% saving).
 *
 * Output lands in static/fonts/ and is committed, so a normal `npm run build`
 * never runs this. Re-run it after changing the character set:
 *   node scripts/build-fonts.mjs
 */
import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import subsetFont from 'subset-font';

const here = dirname(fileURLToPath(import.meta.url));
const sourceDir = join(here, '..', '..', 'mobile', 'assets', 'fonts');
const outDir = join(here, '..', 'static', 'fonts');

/**
 * Everything the UI can display: Persian and Arabic letters (with their
 * presentation forms, which shaping produces via GSUB and therefore must be in
 * the subset), Persian/Arabic-Indic and Latin digits, the separators the money
 * and date formatters emit (U+066C, U+066B), bidi controls (U+200E/F, U+2066-69),
 * plus the Latin punctuation and letters that appear in codes and identifiers.
 */
const CHARACTERS = [
	// Persian/Arabic letters and marks
	...'آابپتثجحخدذرزژسشصضطظعغفقکگلمنوهیءًٌٍَُِّْٰپچژڢکی',
	// Persian digits ۰-۹ and Arabic-Indic ٠-٩
	...'۰۱۲۳۴۵۶۷۸۹٠١٢٣٤٥٦٧٨٩',
	// Separators, marks and punctuation
	...'٬٫،؛؟٪×÷+-_=/()[]{}<>:.,!@#$%^&*',
	// Bidi controls used by the formatters
	...'‎‏⁦⁧⁨⁩',
	// Latin letters (codes, tracking numbers, method names)
	...'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz',
	// Space variants
	...'   '
].join('');

/** Maps the Flutter asset names to CSS numeric weights. */
const WEIGHTS = { Regular: 400, Medium: 500, Bold: 700 };

async function main() {
	await mkdir(outDir, { recursive: true });

	const entries = (await readdir(sourceDir)).filter((name) => name.endsWith('.ttf'));
	if (entries.length === 0) {
		throw new Error(`No .ttf files found in ${sourceDir}`);
	}

	const written = new Set();
	for (const name of entries) {
		const face = name.replace(/^Vazirmatn-/, '').replace(/\.ttf$/, '');
		const weight = WEIGHTS[face];
		if (weight === undefined) {
			throw new Error(
				`Unexpected font file "${name}". Add it to WEIGHTS so it is not silently skipped.`
			);
		}

		const input = await readFile(join(sourceDir, name));
		const woff2 = await subsetFont(input, CHARACTERS, { targetFormat: 'woff2' });
		const outName = `Vazirmatn-${weight}.woff2`;
		await writeFile(join(outDir, outName), woff2);
		written.add(outName);

		const saved = Math.round((1 - woff2.length / input.length) * 100);
		console.log(
			`${name} -> ${outName}: ${(input.length / 1024).toFixed(0)} KB -> ${(woff2.length / 1024).toFixed(0)} KB (-${saved}%)`
		);
	}

	// Guard against a future rename silently colliding two faces on one output.
	if (written.size !== entries.length) {
		throw new Error(
			`Output collision: ${entries.length} inputs produced ${written.size} files. Weights must be unique.`
		);
	}
}

main().catch((err) => {
	console.error(err);
	process.exit(1);
});
