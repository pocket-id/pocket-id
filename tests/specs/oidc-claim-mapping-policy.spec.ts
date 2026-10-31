import test, { expect } from '@playwright/test';
import { oidcClients, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

test.beforeEach(async ({ page }) => {
	await cleanupBackend();
	await setAppConfigValues(page.request, { enabledFeatures: '["claimMappings"]' });
});

test('Map preferred username to email in the access token of an assigned client', async ({
	page
}) => {
	const client = oidcClients.nextcloud;
	const user = users.tim;

	// Create the policy, which starts as a copy of the built-in one
	await page.goto('/settings/admin/claim-mappings');
	await page.getByRole('button', { name: 'Add policy' }).click();
	await page.getByLabel('Name').fill('Email as username');
	await page.getByRole('dialog').getByRole('button', { name: 'Add policy' }).click();

	await page.waitForURL('/settings/admin/claim-mappings/*');
	await expect(page.getByRole('heading', { name: 'Email as username' })).toBeVisible();

	// Source preferred_username from the email and add it to the access token
	await page.getByRole('button', { name: 'Edit preferred_username' }).click();
	const sheet = page.getByRole('dialog');
	await sheet.getByLabel('User field').click();
	await page.getByRole('option', { name: 'Email', exact: true }).click();
	await sheet.getByRole('checkbox', { name: 'Access token' }).click();
	await sheet.getByRole('button', { name: 'Done' }).click();
	await expect(sheet).not.toBeVisible();

	// The preview shows the unsaved change right away
	const preview = page.getByRole('complementary', { name: 'Preview' });
	await preview.getByRole('combobox').click();
	await page.getByRole('option', { name: user.displayName, exact: true }).click();
	await preview.getByRole('tab', { name: 'Access token' }).click();
	await expect(preview.getByLabel('Token as JSON')).toContainText(
		`"preferred_username": "${user.email}"`
	);

	await saveUnsavedChanges(page);

	// Assign the client to the policy
	await page.getByRole('tab', { name: 'Clients' }).click();
	await page.getByRole('button', { name: 'Assign clients…' }).click();
	const picker = page.getByRole('dialog');
	await picker.getByText(client.name, { exact: true }).click();
	await picker.getByRole('button', { name: 'Assign 1 client' }).click();
	await expect(page.getByRole('button', { name: `Unassign ${client.name}` })).toBeVisible();

	// The client's form names the policy it now uses
	await page.getByRole('link', { name: client.name }).click();
	await expect(
		page.getByRole('combobox', { name: 'Claim mapping policy', exact: true })
	).toContainText('Email as username');
});

test('The built-in policy is read-only', async ({ page }) => {
	await page.goto('/settings/admin/claim-mappings');
	await page.getByRole('link', { name: 'Built-in Policy' }).click();

	await expect(page.getByText('The built-in policy can’t be changed')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Edit preferred_username' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Add claim' })).toHaveCount(0);
});

test('Reserved claim names are rejected', async ({ page }) => {
	await page.goto('/settings/admin/claim-mappings');
	await page.getByRole('button', { name: 'Add policy' }).click();
	await page.getByLabel('Name').fill('Reserved');
	await page.getByRole('dialog').getByRole('button', { name: 'Add policy' }).click();
	await page.waitForURL('/settings/admin/claim-mappings/*');

	await page.getByRole('button', { name: 'Add claim' }).last().click();
	const sheet = page.getByRole('dialog');
	await sheet.getByLabel('Claim name').fill('iss');
	await sheet.getByRole('button', { name: 'Add claim' }).click();
	await expect(
		sheet.getByText('iss is set by Pocket ID itself and can’t be mapped.')
	).toBeVisible();
});

test('Claim mappings stay hidden while the feature is off', async ({ page }) => {
	await setAppConfigValues(page.request, { enabledFeatures: '[]' });
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);

	await expect(page.getByRole('link', { name: 'Claim mappings' })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Claims', exact: true })).toHaveCount(0);
});
