import test, { expect, type Page } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';
import { unsavedChangesBar } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

// Changes the mail server, which decides who can read the login codes sent to admins and so needs sudo mode
async function saveMailServer(page: Page) {
	await page.goto('/settings/admin/application-configuration/email');
	await page.getByLabel('Host').fill('smtp.sudo.test');
	await page.getByLabel('Port').fill('587');
	await page.getByLabel('From address').fill('sudo@example.com');
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	const dialog = page.getByRole('alertdialog');
	await expect(dialog.getByText('Confirm it’s you')).toBeVisible();
	return dialog;
}

async function savedMailServer(page: Page) {
	const config = await (await page.request.get('/api/admin/application-configuration')).json();
	return config.find((variable: { key: string }) => variable.key === 'smtpHost').value;
}

test('A change that needs sudo mode is made after confirming with the passkey', async ({
	page
}) => {
	// The seeded session is still in sudo mode, so the first attempt is refused here like the server refuses it once sudo mode has ended
	let attempts = 0;
	await page.route('/api/admin/application-configuration', async (route) => {
		if (route.request().method() !== 'PATCH' || ++attempts > 1) return route.continue();
		await route.fulfill({ status: 403, json: { code: 'sudo_required', error: 'Sudo required' } });
	});
	await (await passkeyUtil.init(page)).addPasskey();

	const dialog = await saveMailServer(page);
	const reauthentication = page.waitForResponse('/api/webauthn/reauthenticate');
	await dialog.getByRole('button', { name: 'Continue with passkey' }).click();
	expect((await reauthentication).status()).toBe(204);

	// The refused request runs once more by itself
	await expect(unsavedChangesBar(page).getByText('Changes saved', { exact: true })).toBeVisible();
	await expect(dialog).toHaveCount(0);
	expect(await savedMailServer(page)).toBe('smtp.sudo.test');
});

test('Cancelling the passkey prompt leaves the change undone', async ({ page }) => {
	await page.route('/api/admin/application-configuration', async (route) => {
		if (route.request().method() !== 'PATCH') return route.continue();
		await route.fulfill({ status: 403, json: { code: 'sudo_required', error: 'Sudo required' } });
	});

	const dialog = await saveMailServer(page);
	await dialog.getByRole('button', { name: 'Cancel' }).click();

	await expect(dialog).toHaveCount(0);
	expect(await savedMailServer(page)).not.toBe('smtp.sudo.test');
});
