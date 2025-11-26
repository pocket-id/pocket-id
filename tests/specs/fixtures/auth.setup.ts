import { test as setup } from '@playwright/test';
import { pathFromRoot } from 'utils/fs.util';
import authUtil from '../../utils/auth.util';
import { cleanupBackend } from '../../utils/cleanup.util';

const authFile = pathFromRoot('.tmp/auth/user.json');

setup('authenticate', async ({ page }) => {
	await cleanupBackend();

	await authUtil.authenticate(page);
	await page.waitForURL('/settings/account');

	await page.context().storageState({ path: authFile });
});
