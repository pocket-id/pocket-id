import test, { expect, type BrowserContext, type Page } from '@playwright/test';
import { oneTimeAccessTokens } from '../data';
import { signedOutState } from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

// The seeded session of the other tests is never bound, so these tests sign in themselves
test.use({ storageState: signedOutState });

const bindingCookieName = /^__Host-session_binding_/;

async function bindingCookie(context: BrowserContext) {
	return (await context.cookies()).find((c) => bindingCookieName.test(c.name));
}

async function signInBound(page: Page) {
	const token = oneTimeAccessTokens.find((t) => !t.expired)!;
	await page.goto(`/lc/${token.token}`);
	await page.waitForURL('/settings/account');

	// Chrome registers the device key in the background and fetches the first binding cookie on its next request to Pocket ID
	await expect
		.poll(async () => {
			await page.evaluate(() => fetch('/api/users/me'));
			return !!(await bindingCookie(page.context()));
		})
		.toBe(true);
}

test('Signing in binds the session to the device', async ({ page }) => {
	await signInBound(page);

	await page.goto('/settings/audit-log');
	await page.getByRole('button', { name: /This device/ }).click();
	const deviceBound = page
		.getByRole('dialog')
		.getByRole('term')
		.filter({ hasText: /^Device-bound$/ })
		.locator('..')
		.getByRole('definition');
	await expect(deviceBound).toHaveText('Yes');
});

test('The browser renews a missing binding cookie', async ({ page, context }) => {
	await signInBound(page);

	// Chrome holds the request back until it has refreshed the binding cookie with the device key
	await context.clearCookies({ name: bindingCookieName });
	const status = await page.evaluate(() => fetch('/api/users/me').then((r) => r.status));
	expect(status).toBe(200);
	expect(await bindingCookie(context)).toBeDefined();
});

test('A copied session cookie does not work without the binding cookie', async ({
	page,
	context,
	playwright
}) => {
	await signInBound(page);
	const sessionToken = (await context.cookies()).find((c) => c.name === '__Host-session_token')!;
	const binding = (await bindingCookie(context))!;

	// A client without the device key only gets as far as the session cookie takes it
	const copy = async (cookie: string) => {
		const client = await playwright.request.newContext({
			baseURL: test.info().project.use.baseURL,
			extraHTTPHeaders: { Cookie: cookie }
		});
		return (await client.get('/api/users/me')).status();
	};
	expect(await copy(`${sessionToken.name}=${sessionToken.value}`)).toBe(401);
	expect(
		await copy(`${sessionToken.name}=${sessionToken.value}; ${binding.name}=${binding.value}`)
	).toBe(200);
});
