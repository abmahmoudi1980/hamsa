import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			csp: {
				mode: 'hash',
				directives: {
					'default-src': ['self'],
					'base-uri': ['self'],
					'connect-src': ['self'],
					'font-src': ['self'],
					'form-action': ['self'],
					'frame-ancestors': ['none'],
					'img-src': ['self', 'blob:', 'data:'],
					'object-src': ['none'],
					'script-src': ['self'],
					'style-src': ['self'],
					'style-src-attr': ['unsafe-inline']
				}
			},
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// `fallback: 'index.html'` + `ssr = false` (see src/routes/+layout.ts) is
			// SvelteKit's SPA mode: every route is served by the same index.html and
			// client-side routing takes over. Required here because the app is
			// authenticated-only — there is nothing meaningful to prerender.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		port: 5173,
		strictPort: true,
		proxy: {
			// Proxying /api through the dev server makes every request same-origin,
			// so the browser needs no CORS grant at all in development and dev
			// cannot drift from production's same-origin setup. It also means the
			// default relative API base works unchanged.
			'/api': {
				target: process.env.HAMSA_API_ORIGIN ?? 'http://localhost:8080',
				changeOrigin: true
			},
			// Uploaded files (receipts, photos) come from the same place.
			'/files': {
				target: process.env.HAMSA_API_ORIGIN ?? 'http://localhost:8080',
				changeOrigin: true
			}
		}
	},
	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'client',
					browser: {
						enabled: true,
						provider: playwright(),
						instances: [{ browser: 'chromium', headless: true }]
					},
					include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
					exclude: ['src/lib/server/**']
				}
			},

			{
				extends: './vite.config.ts',
				test: {
					name: 'server',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
