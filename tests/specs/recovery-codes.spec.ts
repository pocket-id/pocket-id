import test, { expect } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

test('Generate recovery codes from account settings and sign in with one', async ({
	page,
	context
}) => {
	await page.goto('/settings/account');

	await page.getByRole('button', { name: 'Generate', exact: true }).click();

	const dialog = page.getByRole('dialog', { name: 'Recovery Codes' });
	await expect(dialog).toBeVisible();

	const codeItems = dialog.getByRole('list', { name: 'Recovery codes' }).getByRole('listitem');
	await expect(codeItems).toHaveCount(10);

	const firstCode = (await codeItems.first().innerText()).trim();
	expect(firstCode).toMatch(/^[A-Za-z0-9]{4}-[A-Za-z0-9]{4}-[A-Za-z0-9]{4}-[A-Za-z0-9]{4}$/);

	await dialog.getByRole('button', { name: "I've saved my codes" }).click();
	await expect(dialog).toBeHidden();

	await expect(page.getByText('10 of 10 codes remaining')).toBeVisible();

	// Sign out and sign back in using the recovery code.
	await context.clearCookies();
	await page.goto('/login/alternative/recovery-code');

	await page.getByLabel('Recovery code').fill(firstCode);
	await page.getByRole('button', { name: 'Submit' }).click();

	await page.waitForURL('/settings/**');

	// The used code should no longer be valid.
	await expect(page.getByText('9 of 10 codes remaining')).toBeVisible();
});

test('Reusing a recovery code fails', async ({ page, context }) => {
	await page.goto('/settings/account');
	await page.getByRole('button', { name: 'Generate', exact: true }).click();

	const dialog = page.getByRole('dialog', { name: 'Recovery Codes' });
	const firstCode = (
		await dialog
			.getByRole('list', { name: 'Recovery codes' })
			.getByRole('listitem')
			.first()
			.innerText()
	).trim();
	await dialog.getByRole('button', { name: "I've saved my codes" }).click();

	await context.clearCookies();
	await page.goto('/login/alternative/recovery-code');
	await page.getByLabel('Recovery code').fill(firstCode);
	await page.getByRole('button', { name: 'Submit' }).click();
	await page.waitForURL('/settings/**');

	await context.clearCookies();
	await page.goto('/login/alternative/recovery-code');
	await page.getByLabel('Recovery code').fill(firstCode);
	await page.getByRole('button', { name: 'Submit' }).click();

	await expect(page.getByText('Recovery code is invalid or has already been used')).toBeVisible();
});

test('Revoking recovery codes clears the batch', async ({ page }) => {
	await page.goto('/settings/account');
	await page.getByRole('button', { name: 'Generate', exact: true }).click();
	await page
		.getByRole('dialog', { name: 'Recovery Codes' })
		.getByRole('button', { name: "I've saved my codes" })
		.click();

	await expect(page.getByText('10 of 10 codes remaining')).toBeVisible();

	await page.getByRole('button', { name: 'Revoke', exact: true }).click();
	await page.getByRole('alertdialog').getByRole('button', { name: 'Revoke' }).click();

	await expect(page.getByRole('button', { name: 'Generate', exact: true })).toBeVisible();
});
