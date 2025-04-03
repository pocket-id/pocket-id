import type { Page } from '@playwright/test';
import passkeyUtil from './passkey.util';

async function authenticate(page: Page) {
	await page.goto('/login');

	await (await passkeyUtil.init(page)).addPasskey();

	await page.getByRole('button', { name: 'Authenticate' }).click();
}

export default { authenticate };
