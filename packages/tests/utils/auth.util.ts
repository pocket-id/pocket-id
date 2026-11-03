import type { Browser, Page } from '@playwright/test';
import passkeyUtil from './passkey.util';

async function authenticate(page: Page) {
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey();

	await page.getByRole('button', { name: 'Sign in with passkey' }).click();
}

async function changeUser(page: Page, username: keyof typeof passkeyUtil.passkeys) {
	await page.context().clearCookies();
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey(username);
	await page.getByRole('button', { name: 'Sign in with passkey' }).click();
	await page.waitForURL('/settings/**');
}

// Signs in from a separate browser, which gets a Pocket ID session of its own
async function signInElsewhere(browser: Browser, username: keyof typeof passkeyUtil.passkeys) {
	const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
	const page = await context.newPage();
	await changeUser(page, username);
	return { context, page };
}

export default { authenticate, changeUser, signInElsewhere };
