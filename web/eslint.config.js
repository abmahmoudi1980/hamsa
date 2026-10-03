import prettier from 'eslint-config-prettier';
import path from 'node:path';
import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import { defineConfig, includeIgnoreFile } from 'eslint/config';
import globals from 'globals';
import ts from 'typescript-eslint';

const gitignorePath = path.resolve(import.meta.dirname, '.gitignore');

export default defineConfig(
	includeIgnoreFile(gitignorePath),
	js.configs.recommended,
	ts.configs.recommended,
	svelte.configs.recommended,
	prettier,
	svelte.configs.prettier,
	{
		languageOptions: { globals: { ...globals.browser, ...globals.node } },
		rules: {
			// typescript-eslint strongly recommend that you do not use the no-undef lint rule on TypeScript projects.
			// see: https://typescript-eslint.io/troubleshooting/faqs/eslint/#i-get-errors-from-the-no-undef-rule-about-global-variables-not-being-defined-even-though-there-are-no-typescript-errors
			'no-undef': 'off'
		}
	},
	{
		files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: {
				projectService: true,
				extraFileExtensions: ['.svelte'],
				parser: ts.parser
			}
		}
	},
	{
		// Project rules. These encode product invariants that are easy to break
		// by accident and expensive to catch in review.
		rules: {
			'no-restricted-syntax': [
				'error',
				// RTL purity: the app is right-to-left, so a physical
				// margin/padding/text-align utility silently misplaces its
				// element. Logical equivalents (ms-/me-/ps-/pe-/start-/end-) must
				// be used instead. Checked on class attributes, since that is
				// where the utility lives.
				{
					selector: 'Literal[value=/(^|[\\s"\'`])(ml|mr|pl|pr|left|right)-/]',
					message:
						'Use logical utilities in RTL (ms-, me-, ps-, pe-, start-, end-, text-start, text-end) instead of physical ones.'
				},
				{
					selector: 'Literal[value=/(^|[\\s"\'`])text-(left|right)([\\s"\'`]|$)/]',
					message: 'Use text-start / text-end so the alignment follows the document direction.'
				},
				// A bare /files/... URL cannot authenticate: GET /files/{id} is
				// Bearer-gated, so the browser sends no Authorization header and
				// the image silently fails to load. Files must be fetched through
				// useFileUrl and rendered as an object URL.
				{
					selector: 'Literal[value=/["\'`]\\/files\\//]',
					message:
						'File URLs are auth-gated; fetch with the bearer token and render an object URL (see useFileUrl).'
				},
				// A bare Date.parse on a wire value is the usual way a Jalali
				// requirement gets broken: the API speaks Gregorian ISO, and the
				// built-in also introduces timezone drift.
				{
					selector: "CallExpression[callee.property.name='parse'][object.object.name='Date']",
					message: 'Parse wire dates with fromIsoDate from #lib/format/jalali.'
				}
			],
			// Money safety: an amount is a Toman string and must never be coerced
			// to a JS number (float precision loss on large building totals).
			'no-restricted-globals': [
				'error',
				{ name: 'parseInt', message: 'Use parseToman from #lib/format/money.' },
				{ name: 'parseFloat', message: 'Use parseToman from #lib/format/money.' }
			]
		}
	}
);
