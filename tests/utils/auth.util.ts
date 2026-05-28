import type { Page } from '@playwright/test';
import passkeyUtil from './passkey.util';

async function finishAuthentication(page: Page) {
	if (new URL(page.url()).pathname === '/login') {
		await page
			.getByRole('button', { name: 'Authenticate' })
			.click({ timeout: 1000 })
			.catch((error: unknown) => {
				if (new URL(page.url()).pathname === '/login') throw error;
			});
	}

	await page.waitForURL('/settings/**');
}

async function authenticate(page: Page) {
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey();
	await finishAuthentication(page);
}

async function changeUser(page: Page, username: keyof typeof passkeyUtil.passkeys) {
	await page.context().clearCookies();
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey(username);
	await finishAuthentication(page);
}

export default { authenticate, changeUser };
