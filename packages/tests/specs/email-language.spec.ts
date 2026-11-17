import { expect, type APIRequestContext } from '@playwright/test';
import { users } from '../data';
import playwrightConfig from '../playwright.config';
import { setAppConfigValues } from '../utils/app-config.util';
import { signedOutState } from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { authFile } from '../utils/fs.util';
import { lastEmailTo, smtpSinkConfig, test } from '../utils/smtp.util';

// A browser of its own does not inherit the project options, so it gets the base URL explicitly
const baseURL = playwrightConfig.use!.baseURL;

test.beforeEach(async () => await cleanupBackend());

async function currentLocale(request: APIRequestContext) {
	const user = await request.get('/api/users/me').then((r) => r.json());
	return user.locale;
}

test('Signing up saves the browser language', async ({ browser, page }) => {
	await setAppConfigValues(page.request, { allowUserSignups: 'open' });

	// The UI shows German, so the form is filled by the order of its fields
	const context = await browser.newContext({
		baseURL,
		locale: 'de-DE',
		storageState: signedOutState
	});
	const germanPage = await context.newPage();
	await germanPage.goto('/signup');

	const fields = germanPage.getByRole('textbox');
	await fields.nth(0).fill('Jana');
	await fields.nth(1).fill('Schmidt');
	await fields.nth(2).fill('janaschmidt');
	await fields.nth(3).fill('jana.schmidt@test.com');
	await germanPage.locator('button[type="submit"]').click();
	await germanPage.waitForURL('/signup/add-passkey');

	expect(await currentLocale(germanPage.request)).toBe('de');
	await context.close();
});

test('Signing in saves the browser language of a user without one', async ({ browser }) => {
	const context = await browser.newContext({ baseURL, locale: 'de-DE', storageState: authFile });
	// The shared sign-in state can hold the language cookie of the English browser that created it, which would win over the browser language
	await context.clearCookies({ name: 'locale' });
	const page = await context.newPage();
	await page.goto('/settings/account');

	await expect.poll(() => currentLocale(page.request)).toBe('de');
	await context.close();
});

test.describe('Login code by email', () => {
	test.use({ storageState: signedOutState });

	test('The email arrives and its code signs the user in', async ({ browser, page, smtp }) => {
		// Changing the configuration needs an admin session
		const admin = await browser.newContext({ baseURL, storageState: authFile });
		await setAppConfigValues(admin.request, {
			...smtpSinkConfig,
			emailOneTimeAccessAsUnauthenticatedEnabled: 'true'
		});
		await admin.close();

		await page.goto('/login/alternative/email');
		await page.getByRole('textbox').fill(users.tim.email);
		await page.locator('button[type="submit"]').click();
		await page.waitForURL('/login/alternative/code');

		// The email is sent in the background
		await expect.poll(() => smtp.emails.length).toBe(1);
		const email = lastEmailTo(smtp, users.tim.email);
		expect(email.subject).toBe('Login Code');
		expect(email.body).toContain('<html dir="ltr" lang="en">');
		expect(email.body).toContain('Your login code');

		const code = email.body.match(/\/lc\/([A-Za-z0-9]+)/)?.[1];
		expect(code).toBeDefined();
		await page.goto(`/lc/${code}`);
		await page.waitForURL('/settings/account');
	});
});
