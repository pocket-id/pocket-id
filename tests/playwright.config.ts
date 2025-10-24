import { defineConfig, devices } from '@playwright/test';
import 'dotenv/config';

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
	outputDir: './.output',
	timeout: 10000,
	testDir: './specs',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	reporter: process.env.CI
		? [['html', { outputFolder: '.report' }], ['github']]
		: [['line'], ['html', { open: 'never', outputFolder: '.report' }]],
	use: {
		baseURL: process.env.APP_URL ?? 'http://localhost:1411',
		video: 'retain-on-failure',
		trace: 'on-first-retry'
	},
	projects: [
		{ name: 'cli', testMatch: /cli\.spec\.ts/ },
		{ name: 'auth-setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], storageState: '.tmp/auth/user.json' },
			dependencies: ['auth-setup']
		}
	],
	globalSetup: './specs/fixtures/global.setup.ts',
	globalTeardown: './specs/fixtures/global.teardown.ts'
});
