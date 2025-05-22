import { test as setup } from '@playwright/test';
import authUtil from './utils/auth.util';
import { cleanupBackend } from './utils/cleanup.util';

const authFile = 'tests/.auth/user.json';

setup('authenticate', async ({ page }) => {
	await cleanupBackend();

	await authUtil.authenticate(page);
	await page.waitForURL('/settings/account');

	await page.context().storageState({ path: authFile });
});
