import test, { expect, type Browser } from '@playwright/test';
import { users } from '../data';
import authUtil, { signedOutState } from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => await cleanupBackend());

async function signInElsewhere(browser: Browser, username: 'tim' | 'craig') {
	const context = await browser.newContext({ storageState: signedOutState });
	const page = await context.newPage();
	await authUtil.changeUser(page, username);
	return { context, page };
}

for (const { choice, persistent, remember } of [
	{ choice: 'Stay signed in', persistent: true, remember: true },
	{ choice: "Don't stay signed in", persistent: false, remember: true },
	{ choice: "Don't stay signed in", persistent: false, remember: false }
]) {
	test(`Ask whether to stay signed in: ${choice}, ${remember ? 'remembered' : 'asked again'}`, async ({
		browser
	}) => {
		const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
		const page = await context.newPage();
		// A cookie without an expiry is dropped when the browser closes
		const expectSessionCookie = async () => {
			const cookie = (await context.cookies()).find((c) => c.name === 'session_token');
			expect(cookie!.expires > 0).toBe(persistent);
		};

		await page.goto('/login');
		await (await passkeyUtil.init(page)).addPasskey();
		await page.getByRole('button', { name: 'Sign in with passkey' }).click();

		await expect(page.getByRole('heading', { name: 'Stay signed in?' })).toBeVisible();
		const dontAskAgain = page.getByRole('checkbox', { name: "Don't ask again in this browser" });
		await expect(dontAskAgain).toBeChecked();
		if (!remember) await dontAskAgain.click();
		await page.getByRole('button', { name: choice, exact: true }).click();
		await page.waitForURL('/settings/**');
		await expectSessionCookie();

		await context.clearCookies();
		await page.goto('/login');
		await page.getByRole('button', { name: 'Sign in with passkey' }).click();
		if (remember) {
			await page.waitForURL('/settings/**');
			await expectSessionCookie();
		} else {
			await expect(page.getByRole('heading', { name: 'Stay signed in?' })).toBeVisible();
		}
		await context.close();
	});
}

test('Sign out of another device', async ({ page, browser }) => {
	const other = await signInElsewhere(browser, 'tim');

	await page.goto('/settings/audit-log');
	const devices = page.getByRole('list', { name: 'Signed-in devices' });
	await expect(devices.getByRole('listitem').first()).toContainText('This device');

	// The other sessions are ordered by their last activity, so the device that just signed in comes right after this one
	const otherDevice = devices.getByRole('listitem').nth(1);
	await otherDevice.getByRole('button').first().click();
	await expect(page.getByRole('dialog').getByText('Passkey', { exact: true })).toBeVisible();
	await page.getByRole('dialog').getByRole('button', { name: 'Sign out', exact: true }).click();
	await page
		.getByRole('alertdialog')
		.getByRole('button', { name: 'Sign out', exact: true })
		.click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Device signed out');

	await other.page.goto('/settings/account');
	await other.page.waitForURL(/\/login/);
	await other.context.close();
});

test('Sign out of all other devices', async ({ page, browser }) => {
	const other = await signInElsewhere(browser, 'tim');

	await page.goto('/settings/audit-log');
	await page.getByRole('button', { name: 'Sign out other devices' }).click();
	await page
		.getByRole('alertdialog')
		.getByRole('button', { name: 'Sign out other devices' })
		.click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Signed out of all other devices');

	const devices = page.getByRole('list', { name: 'Signed-in devices' });
	await expect(devices.getByRole('listitem')).toHaveCount(1);
	await expect(page.getByRole('button', { name: 'Sign out other devices' })).toHaveCount(0);

	await other.page.goto('/settings/account');
	await other.page.waitForURL(/\/login/);
	await other.context.close();
});

test('Signing out revokes the session', async ({ browser }) => {
	const other = await signInElsewhere(browser, 'tim');
	const signedIn = await other.context.storageState();

	const logout = await other.context.request.post('/api/webauthn/logout');
	expect(logout.status()).toBe(204);

	// A copy of the cookie taken before signing out must not work anymore
	const copy = await browser.newContext({ storageState: signedIn });
	const res = await copy.request.get('/api/users/me');
	expect(res.status()).toBe(401);

	await copy.close();
	await other.context.close();
});

test('Admin signs a user out everywhere', async ({ page, browser }) => {
	const craig = await signInElsewhere(browser, 'craig');

	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await page.getByRole('tab', { name: /^Devices/ }).click();
	await page.getByRole('button', { name: 'Sign out everywhere' }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Sign out everywhere' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Craig was signed out everywhere');
	await expect(page.getByText('Craig isn’t signed in anywhere.')).toBeVisible();

	await craig.page.goto('/settings/account');
	await craig.page.waitForURL(/\/login/);
	await craig.context.close();
});
