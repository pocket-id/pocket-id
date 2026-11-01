import { test as setup } from '@playwright/test';
import { pathFromRoot } from 'utils/fs.util';
import { sessionToken } from '../../data';
import playwrightConfig from '../../playwright.config';
import { cleanupBackend } from '../../utils/cleanup.util';

const authFile = pathFromRoot('.tmp/auth/user.json');

setup('authenticate', async ({ page }) => {
	await cleanupBackend();

	await page.context().addCookies([
		{
			name: 'session_token',
			value: sessionToken,
			url: playwrightConfig.use!.baseURL,
			httpOnly: true,
			sameSite: 'Lax'
		}
	]);
	await page.goto('/settings/account');
	await page.waitForURL('/settings/account');

	await page.context().storageState({ path: authFile });
});
