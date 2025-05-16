import { defineConfig, devices } from '@playwright/test';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
	outputDir: './tests/.output',
	timeout: 10000,
	testDir: './tests',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	reporter: process.env.CI
		? [['html', { outputFolder: 'tests/.report' }], ['github']]
		: [['line'], ['html', { open: 'never', outputFolder: 'tests/.report' }]],
	use: {
		baseURL: process.env.APP_URL ?? 'http://localhost:1411',
		video: 'retain-on-failure',
		trace: 'on-first-retry'
	},
	projects: [
		{ name: 'setup', testMatch: /.*\.setup\.ts/ },
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], storageState: 'tests/.auth/user.json' },
			dependencies: ['setup']
		}
	]
});
