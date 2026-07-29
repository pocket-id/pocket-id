import test, { expect, type Browser } from '@playwright/test';
import { oneTimeAccessTokens } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import { pathFromRoot } from '../utils/fs.util';

test.beforeEach(async () => await cleanupBackend());

// Disable authentication for these tests
test.use({ storageState: { cookies: [], origins: [] } });

test('Sign in with login code', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => !t.expired)[0];
	await page.goto(`/lc/${token.token}`);

	await page.waitForURL('/settings/account');
});

test('Sign in with expired login code fails', async ({ page }) => {
	const token = oneTimeAccessTokens.filter((t) => t.expired)[0];
	await page.goto(`/lc/${token.token}`);

	await expect(page.getByRole('paragraph')).toHaveText(
		'Token is invalid or expired. Please try again.'
	);
});

test('Sign in with login code entered manually', async ({ page }) => {
	const token = oneTimeAccessTokens.find((t) => !t.expired)!;
	await page.goto('/lc');

	await page.getByPlaceholder('Code').fill(token.token);
	await page.getByText('Submit').click();

	await page.waitForURL('/settings/account');
});

test('Sign in with login code entered manually fails', async ({ page }) => {
	const token = oneTimeAccessTokens.find((t) => t.expired)!;
	await page.goto('/lc');

	await page.getByPlaceholder('Code').fill(token.token);
	await page.getByText('Submit').click();

	await expect(page.getByRole('paragraph')).toHaveText(
		'Token is invalid or expired. Please try again.'
	);
});

test('Sign in with login code entered manually when email login is enabled', async ({
	browser,
	page
}) => {
	await setEmailLoginEnabled(browser);

	const token = oneTimeAccessTokens.find((t) => !t.expired)!;
	await page.goto('/lc');

	await page.getByText('I have a longer code').click();
	await page.getByPlaceholder('Code').fill(token.token);
	await page.getByText('Submit').click();

	await page.waitForURL('/settings/account');
});

async function setEmailLoginEnabled(browser: Browser) {
	const context = await browser.newContext({
		baseURL: test.info().project.use.baseURL,
		storageState: pathFromRoot('.tmp/auth/user.json')
	});
	const page = await context.newPage();

	try {
		const configResponse = await page.request.get('/api/application-configuration/all');
		expect(configResponse.ok()).toBe(true);

		const config = Object.fromEntries(
			((await configResponse.json()) as Array<{ key: string; value: string }>).map(
				({ key, value }) => [key, value]
			)
		);
		config.emailOneTimeAccessAsUnauthenticatedEnabled = 'true';

		const updateResponse = await page.request.put('/api/application-configuration', {
			data: config
		});
		expect(updateResponse.ok()).toBe(true);
	} finally {
		await context.close();
	}
}
