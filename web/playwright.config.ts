import { defineConfig } from '@playwright/test';

/**
 * E2E runs against the real static bundle behind the same SPA fallback the
 * production nginx config provides (`scripts/serve-spa.mjs`), so CI exercises
 * the deployed behaviour rather than a dev-server shortcut.
 */
export default defineConfig({
	testDir: 'e2e',
	testMatch: '**/*.e2e.{ts,js}',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? 'github' : 'list',
	// A manager session is shared across specs via storageState, so they must not
	// race each other against one database.
	workers: 1,
	use: {
		baseURL: 'http://localhost:4173',
		locale: 'fa-IR',
		timezoneId: 'Asia/Tehran',
		trace: 'on-first-retry'
	},
	projects: [
		{ name: 'setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'chromium',
			dependencies: ['setup'],
			use: {
				...(process.env.CI ? { headless: true } : {}),
				storageState: 'e2e/.auth/manager.json'
			}
		}
	],
	webServer: {
		command: 'npm run build && npm run preview',
		port: 4173,
		reuseExistingServer: !process.env.CI,
		timeout: 180_000
	}
});
