import test, { expect, type Page, type Locator } from '@playwright/test';
import { oidcClients } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import * as oidcUtil from '../utils/oidc.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => await cleanupBackend());

function createUrlParams(oidcClient: { id: string; callbackUrl: string }) {
	return new URLSearchParams({
		client_id: oidcClient.id,
		response_type: 'code',
		scope: 'openid profile email',
		redirect_uri: oidcClient.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa',
		nonce: 'P1gN3PtpKHJgKUVcLpLjm'
	});
}

async function tabTo(page: Page, target: Locator, maxTabs = 10) {
	// The autofocus buttons focus themselves via setTimeout(100) after mount; wait that
	// out so it cannot yank focus away mid-walk and race the assertions below
	await page.waitForTimeout(150);
	await page.keyboard.press('Tab');
	for (
		let i = 0;
		i < maxTabs && !(await target.evaluate((el) => el === document.activeElement));
		i++
	) {
		await page.keyboard.press('Tab');
	}
	await expect(target).toBeFocused();
}

test('account menu is not rendered on the authorize interaction screen', async ({ page }) => {
	await page.goto(`/authorize?${createUrlParams(oidcClients.immich).toString()}`);

	await expect(page.getByTestId('scopes')).toBeVisible();
	await expect(page.getByRole('button', { name: 'My Account' })).toHaveCount(0);
});

test('account menu still renders on settings pages', async ({ page }) => {
	await page.goto('/settings/account');

	await expect(page.getByRole('button', { name: 'My Account' })).toBeVisible();
});

test('keyboard focus starts on Sign in and Tab moves to Cancel without detours', async ({
	page
}) => {
	await page.goto(`/authorize?${createUrlParams(oidcClients.immich).toString()}`);

	const signIn = page.getByRole('button', { name: 'Sign in' });
	const cancel = page.getByRole('link', { name: 'Cancel' });

	await expect(signIn).toBeFocused();
	await page.keyboard.press('Tab');
	await expect(cancel).toBeFocused();
	await page.keyboard.press('Shift+Tab');
	await expect(signIn).toBeFocused();
});

test('focus moves back to Sign in after switching to a different account', async ({ page }) => {
	await page.goto(
		`/authorize?${createUrlParams(oidcClients.nextcloud).toString()}&prompt=select_account`
	);

	await expect(page.getByTestId('account-selection')).toBeVisible();

	const useDifferentAccount = page.getByRole('button', { name: 'Use a different account' });
	await tabTo(page, useDifferentAccount);
	await page.keyboard.press('Enter');

	await expect(page.getByRole('button', { name: 'Sign in' })).toBeFocused();
});

test('Use a different account shows a visible focus indicator', async ({ page }) => {
	await page.goto(
		`/authorize?${createUrlParams(oidcClients.nextcloud).toString()}&prompt=select_account`
	);

	await expect(page.getByTestId('account-selection')).toBeVisible();

	const useDifferentAccount = page.getByRole('button', { name: 'Use a different account' });
	await tabTo(page, useDifferentAccount);

	const focusIndicator = await useDifferentAccount.evaluate((el) => {
		const style = getComputedStyle(el);
		return {
			textDecorationLine: style.textDecorationLine,
			boxShadow: style.boxShadow
		};
	});
	expect(focusIndicator.textDecorationLine).toContain('underline');
	expect(focusIndicator.boxShadow).toContain('3px');
});

test('Alternative sign in methods link shows a visible focus indicator', async ({ page }) => {
	await page.context().clearCookies();
	await page.goto('/login');

	const alternativeLink = page.getByRole('link', { name: 'Alternative Sign In Methods' });
	await tabTo(page, alternativeLink);

	const focusIndicator = await alternativeLink.evaluate((el) => {
		const style = getComputedStyle(el);
		return {
			textDecorationLine: style.textDecorationLine,
			boxShadow: style.boxShadow
		};
	});
	expect(focusIndicator.textDecorationLine).toContain('underline');
	expect(focusIndicator.boxShadow).toContain('3px');
});

test('keyboard-only account switch signs in as another user and completes consent', async ({
	page
}) => {
	test.setTimeout(60000);
	const oidcClient = oidcClients.nextcloud;

	// The virtual authenticator only holds craig's passkey, so the passkey ceremony
	// after the account switch can only sign in as craig, never as tim
	const passkey = await passkeyUtil.init(page);
	await passkey.addPasskey('craig');

	// Regression guard for the original bug: useDifferentAccount used to complete the
	// interaction step after logging out, which always failed with 401
	let sawInteractionUnauthorized = false;
	page.on('response', (res) => {
		if (res.status() === 401 && res.url().includes('/interactions/')) {
			sawInteractionUnauthorized = true;
		}
	});

	const callbackUrl = await oidcUtil.interceptCallbackRedirect(
		page,
		new URL(oidcClient.callbackUrl).pathname,
		async () => {
			await page.goto(`/authorize?${createUrlParams(oidcClient).toString()}&prompt=select_account`);
			await expect(page.getByTestId('account-selection')).toBeVisible();

			const useDifferentAccount = page.getByRole('button', { name: 'Use a different account' });
			await tabTo(page, useDifferentAccount);
			await page.keyboard.press('Enter');

			const signIn = page.getByRole('button', { name: 'Sign in' });
			await expect(signIn).toBeFocused();
			await page.keyboard.press('Enter');

			// Craig has no seeded authorization for this client, so the consent step appears.
			// Disabling the button while isLoading blurs it, so navigate back by keyboard.
			await expect(page.getByTestId('scopes')).toBeVisible();
			await tabTo(page, signIn);
			await page.keyboard.press('Enter');
		},
		30000
	);

	expect(callbackUrl.searchParams.get('code')).toBeTruthy();
	expect(callbackUrl.searchParams.get('error')).toBeNull();
	expect(sawInteractionUnauthorized).toBe(false);
});
