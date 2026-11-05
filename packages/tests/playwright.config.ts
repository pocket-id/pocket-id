import { defineConfig, devices } from '@playwright/test';
import 'dotenv/config';
import { authFile } from './utils/fs.util';

const shard = process.env.E2E_SHARD;

/**
 * See https://playwright.dev/docs/test-configuration.
 */
export default defineConfig({
	outputDir: shard ? `./.output/shard-${shard}` : './.output',
	timeout: 10000,
	testDir: './specs',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	// Each shard writes its own blob because the blob reporter wipes its output folder on start
	reporter: shard
		? [['blob', { outputDir: `.blob/shard-${shard}` }], ['github']]
		: [['line'], ['html', { open: 'never', outputFolder: '.report' }]],
	use: {
		baseURL: process.env.APP_URL ?? 'http://localhost:1511',
		video: 'retain-on-failure',
		trace: 'on-first-retry'
	},
	projects: [
		{ name: 'cli', testMatch: /cli\.spec\.ts/ },
		{ name: 'auth-setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'browser-chrome',
			use: { ...devices['Desktop Chrome'], storageState: authFile },
			testIgnore: /cli\.spec\.ts/,
			dependencies: ['auth-setup']
		}
	],
	globalSetup: './specs/fixtures/global.setup.ts',
	globalTeardown: './specs/fixtures/global.teardown.ts'
});
