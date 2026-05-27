import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('login page starts passkey autofill when supported', async ({ page, context }) => {
	await context.clearCookies();

	await page.addInitScript(() => {
		Object.defineProperty(globalThis, 'PublicKeyCredential', {
			value: (globalThis as any).PublicKeyCredential ?? function PublicKeyCredential() {},
			configurable: true
		});
		Object.defineProperty(
			(globalThis as any).PublicKeyCredential,
			'isConditionalMediationAvailable',
			{
				value: () => Promise.resolve(true),
				configurable: true
			}
		);
	});

	let loginOptionsRequested = false;
	await page.route('/api/webauthn/login/start', async (route) => {
		loginOptionsRequested = true;
		await route.continue();
	});

	await page.goto('/login');

	await expect(page.getByRole('textbox', { name: 'Passkeys' })).toHaveAttribute(
		'autocomplete',
		'username webauthn'
	);
	await expect.poll(() => loginOptionsRequested).toBe(true);
});
