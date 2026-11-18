import { defineConfig, devices } from '@playwright/test';
import 'dotenv/config';
import { createHash, X509Certificate } from 'node:crypto';
import { readFileSync } from 'node:fs';
import tls from 'node:tls';
import { authFile, pathFromRoot } from './utils/fs.util';

const shard = process.env.E2E_SHARD;

// The stack serves HTTPS with a self-signed certificate, which the tests trust instead of ignoring certificate errors
const testCertificatePEM = readFileSync(pathFromRoot('setup/tls/cert.pem'), 'utf8');
const testCertificateSPKI = createHash('sha256')
	.update(new X509Certificate(testCertificatePEM).publicKey.export({ type: 'spki', format: 'der' }))
	.digest('base64');
tls.setDefaultCACertificates([...tls.getCACertificates('default'), testCertificatePEM]);

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
		baseURL: process.env.APP_URL ?? 'https://localhost:1511',
		channel: 'chromium',
		launchOptions: {
			args: [
				// Linux has no hardware key Chrome supports, so it binds sessions to software keys, and the quota would throttle the tests' refreshes
				'--enable-features=DeviceBoundSessions:RefreshQuota/false/CheckSubdomainRegistration/false,EnableBoundSessionCredentialsSoftwareKeysForManualTesting,UseUnexportableKeyServiceInBrowserProcess',
				`--ignore-certificate-errors-spki-list=${testCertificateSPKI}`
			]
		},
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
