import type { Page } from '@playwright/test';
import passkeyUtil from './passkey.util';

async function authenticate(page: Page) {
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey();

	await page.getByRole('button', { name: 'Authenticate' }).click();
}

async function changeUser(page: Page, username: keyof typeof passkeyUtil.passkeys) {
	await page.context().clearCookies();
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey(username);
	await page.getByRole('button', { name: 'Authenticate' }).click();
}

export default { authenticate, changeUser };
