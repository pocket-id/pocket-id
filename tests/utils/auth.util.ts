import type { Page } from '@playwright/test';
import passkeyUtil from './passkey.util';

async function finishAuthentication(page: Page, url = '/settings/**') {
	const waitForAuthentication = page.waitForURL(url);

	if (new URL(page.url()).pathname === '/login') {
		const clickAuthenticate = page
			.getByRole('button', { name: 'Authenticate' })
			.click()
			.catch((error: unknown) => {
				if (new URL(page.url()).pathname === '/login') throw error;
			});

		await Promise.race([waitForAuthentication, clickAuthenticate]);
	}

	await waitForAuthentication;
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

export default { authenticate, changeUser, finishAuthentication };
