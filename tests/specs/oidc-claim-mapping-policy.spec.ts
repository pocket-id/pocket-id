import test, { expect } from '@playwright/test';
import { oidcClients, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

test.beforeEach(async () => await cleanupBackend());

test('Map preferred username to email in the access token of an assigned client', async ({
	page
}) => {
	const client = oidcClients.nextcloud;
	const user = users.tim;

	// Create the policy
	await page.goto('/settings/admin/oidc-claim-mapping-policies');
	await page.getByRole('button', { name: 'Add policy' }).click();
	await page.getByLabel('Name').fill('Email as username');
	await page.getByRole('button', { name: 'Save', exact: true }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'Claim mapping policy created successfully'
	);
	await page.waitForURL('/settings/admin/oidc-claim-mapping-policies/*');
	await expect(page.getByLabel('Name')).toHaveValue('Email as username');

	// Source preferred_username from the email and add it to the access token
	await page
		.getByRole('row', { name: 'preferred_username' })
		.getByRole('button', { name: 'Edit' })
		.click();

	const mappingDialog = page.getByRole('dialog');
	await mappingDialog.getByRole('button', { name: 'Source value' }).click();
	await page.getByRole('option', { name: 'Email', exact: true }).click();
	await mappingDialog.getByRole('switch', { name: 'Access Token' }).click();
	await mappingDialog.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(mappingDialog).not.toBeVisible();

	await expect(page.getByRole('row', { name: 'preferred_username' })).toContainText('Email');
	await saveUnsavedChanges(page);

	// Assign the client to the policy
	await page.getByRole('tab', { name: 'OIDC Clients' }).click();
	await page.getByRole('button', { name: 'Assign a client' }).click();
	await page.getByRole('dialog').getByRole('row', { name: client.name }).click();

	await expect(
		page.locator('[data-type="success"]', {
			hasText: 'Claim mapping policy clients updated successfully'
		})
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: `Unassign ${client.name} from this policy` })
	).toBeVisible();

	// The preview shows the email as preferred_username in the access token
	await page.getByRole('tab', { name: 'OIDC Data Preview' }).click();
	await page.getByRole('button', { name: 'Show', exact: true }).click();

	const previewDialog = page.getByRole('dialog');
	await previewDialog.getByRole('combobox').first().click();
	await page.getByRole('option', { name: user.username, exact: true }).click();
	await expect(previewDialog).toContainText(`Preview for ${user.displayName}`);

	await previewDialog.getByRole('tab', { name: 'Access Token' }).click();
	const accessTokenPanel = previewDialog.getByRole('tabpanel');
	const preferredUsername = accessTokenPanel
		.locator('div')
		.filter({ has: page.getByText('preferred_username', { exact: true }) })
		.last();
	await expect(preferredUsername).toContainText(user.email);
});
