import test, { expect, Page } from '@playwright/test';
import { oidcClients, userGroups } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

const defaultTokenLifetimes = {
	accessTokenDurationMinutes: 60,
	refreshTokenDurationMinutes: 30 * 24 * 60
};

test.beforeEach(async () => await cleanupBackend());

test.describe('Create OIDC client', () => {
	async function createClientTest(page: Page, clientId?: string) {
		const oidcClient = oidcClients.pingvinShare;
		await page.goto('/settings/admin/oidc-clients');
		await page.getByRole('button', { name: 'Add OIDC Client' }).click();

		await page.getByLabel('Name').fill(oidcClient.name);
		await page.getByLabel('Description').fill(oidcClient.description);
		await page.getByLabel('Client Launch URL').fill(oidcClient.launchURL);

		await page.getByRole('button', { name: 'Add' }).first().click();
		await page.getByTestId('callback-url-1').fill(oidcClient.callbackUrl);

		await page.getByRole('button', { name: 'Add another' }).click();
		await page.getByTestId('callback-url-2').fill(oidcClient.secondCallbackUrl);

		await page.locator('[role="tab"][data-value="light-logo"]').first().click();
		await page.setInputFiles('#oidc-client-logo-light', 'resources/images/pingvin-share-logo.png');
		await page.locator('[role="tab"][data-value="dark-logo"]').first().click();
		await page.setInputFiles('#oidc-client-logo-dark', 'resources/images/pingvin-share-logo.png');

		if (clientId) {
			await page.getByRole('button', { name: 'Show Advanced Options' }).click();
			await page.getByLabel('Client ID').fill(clientId);
		}

		await page.getByRole('button', { name: 'Save' }).click();

		await expect(page.locator('[data-type="success"]')).toHaveText(
			'OIDC client created successfully'
		);

		const resolvedClientId = (await page.getByTestId('client-id').innerText()).trim();
		const clientSecret = (await page.getByTestId('client-secret').innerText()).trim();

		if (clientId) {
			expect(resolvedClientId).toBe(clientId);
		} else {
			expect(resolvedClientId).toMatch(/^[\w-]{36}$/);
		}

		expect(clientSecret).toMatch(/^\w{32}$/);

		await expect(page.getByLabel('Name')).toHaveValue(oidcClient.name);
		await expect(page.getByLabel('Description')).toHaveValue(oidcClient.description);
		await expect(page.getByTestId('callback-url-1')).toHaveValue(oidcClient.callbackUrl);
		await expect(page.getByTestId('callback-url-2')).toHaveValue(oidcClient.secondCallbackUrl);
		await expect(page.getByRole('img', { name: `${oidcClient.name} logo` }).first()).toBeVisible();

		const res = await page.request.get(`/api/oidc/clients/${resolvedClientId}/logo`);
		expect(res.ok()).toBeTruthy();
	}

	test('with auto-generated client ID', async ({ page }) => {
		await createClientTest(page);
	});

	test('with custom client ID', async ({ page }) => {
		await createClientTest(page, '123e4567-e89b-12d3-a456-426614174000');
	});
});

test('Edit OIDC client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${oidcClient.id}`);

	await page.getByLabel('Name').fill('Nextcloud updated');
	await page.getByLabel('Description').fill('Updated description');
	await page.getByTestId('callback-url-1').first().fill('http://nextcloud-updated/auth/callback');
	await page.locator('[role="tab"][data-value="light-logo"]').first().click();
	await page.setInputFiles('#oidc-client-logo-light', 'resources/images/cloud-logo.png');
	await page.locator('[role="tab"][data-value="dark-logo"]').first().click();
	await page.setInputFiles('#oidc-client-logo-dark', 'resources/images/cloud-logo.png');
	await page.getByLabel('Client Launch URL').fill(oidcClient.launchURL);
	const clientForm = page.getByLabel('Name').locator('xpath=ancestor::form');
	await clientForm.getByRole('button', { name: 'Save' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client updated successfully'
	);
	await expect(page.getByRole('img', { name: 'Nextcloud updated logo' }).first()).toBeVisible();
	await page.request
		.get(`/api/oidc/clients/${oidcClient.id}/logo`)
		.then((res) => expect.soft(res.status()).toBe(200));
});

test('Displays OIDC client endpoints from discovery configuration', async ({ page }) => {
	const oidcConfiguration = {
		issuer: 'https://id.example.com',
		authorization_endpoint: 'https://id.example.com/authorize',
		token_endpoint: 'http://pocket-id:1411/api/oidc/token',
		userinfo_endpoint: 'http://pocket-id:1411/api/oidc/userinfo',
		end_session_endpoint: 'https://id.example.com/api/oidc/end-session',
		jwks_uri: 'http://pocket-id:1411/.well-known/jwks.json'
	};

	await page.route('**/.well-known/openid-configuration', async (route) => {
		await route.fulfill({ json: oidcConfiguration });
	});
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);
	await page.getByRole('button', { name: 'Show more details' }).click();

	await expect(page.getByText(oidcConfiguration.token_endpoint, { exact: true })).toBeVisible();
	await expect(page.getByText(oidcConfiguration.userinfo_endpoint, { exact: true })).toBeVisible();
	await expect(page.getByText(oidcConfiguration.jwks_uri, { exact: true })).toBeVisible();
});

test('Update OIDC client token lifetimes', async ({ page }) => {
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);

	const card = page.getByTestId('token-lifetimes-card');
	const accessLifetime = card.getByLabel('Access token lifetime', { exact: true });
	const accessUnit = card.getByLabel('Access token lifetime unit');
	const refreshLifetime = card.getByLabel('Refresh token inactivity timeout', { exact: true });
	const refreshUnit = card.getByLabel('Refresh token inactivity timeout unit');

	await expect(accessLifetime).toHaveValue('1');
	await expect(accessUnit).toHaveText('Hours');
	await expect(refreshLifetime).toHaveValue('30');
	await expect(refreshUnit).toHaveText('Days');

	await accessUnit.click();
	await page.getByRole('option', { name: 'Minutes' }).click();
	await expect(accessLifetime).toHaveValue('60');
	await accessLifetime.fill('90');

	await refreshUnit.click();
	await page.getByRole('option', { name: 'Hours' }).click();
	await expect(refreshLifetime).toHaveValue('720');
	await refreshLifetime.fill('336');

	await card.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('OIDC client updated successfully', { exact: true })).toBeVisible();

	await page.reload();
	await expect(card.getByLabel('Access token lifetime', { exact: true })).toHaveValue('90');
	await expect(card.getByLabel('Access token lifetime unit')).toHaveText('Minutes');
	await expect(card.getByLabel('Refresh token inactivity timeout', { exact: true })).toHaveValue(
		'14'
	);
	await expect(card.getByLabel('Refresh token inactivity timeout unit')).toHaveText('Days');

	await card.getByLabel('Access token lifetime', { exact: true }).fill('0');
	await card.getByRole('button', { name: 'Save' }).click();
	await expect(card.getByText('Token lifetime must be at least 1 minute.')).toBeVisible();

	await card.getByLabel('Access token lifetime', { exact: true }).fill('525601');
	await card.getByRole('button', { name: 'Save' }).click();
	await expect(card.getByText('Token lifetime cannot exceed 365 days.')).toBeVisible();

	await card.getByLabel('Access token lifetime', { exact: true }).fill('1.5');
	await card.getByRole('button', { name: 'Save' }).click();
	await expect(card.getByText('Token lifetime must use whole-minute increments.')).toBeVisible();

	await card.getByLabel('Access token lifetime', { exact: true }).fill('60');
	await card.getByLabel('Refresh token inactivity timeout', { exact: true }).fill('30');
	await card.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('OIDC client updated successfully', { exact: true })).toBeVisible();
});

test('Update OIDC client federated credentials', async ({ page }) => {
	const client = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${client.id}`);

	const card = page.getByTestId('federated-credentials-card');
	await card.getByRole('button', { name: 'Create', exact: true }).click();
	await card.getByLabel('Issuer').fill('https://issuer.example.com');
	await card.getByLabel('Subject').fill('workload-client');
	await card.getByLabel('Audience').fill('https://pocket-id.example.com');

	const cardUpdate = page.waitForResponse(
		(response) =>
			response.request().method() === 'PUT' &&
			response.url().endsWith(`/api/oidc/clients/${client.id}`)
	);
	await card.getByRole('button', { name: 'Save' }).click();
	expect((await cardUpdate).ok()).toBeTruthy();

	await page.reload();
	await expect(card.getByLabel('Issuer')).toHaveValue('https://issuer.example.com');
	await expect(card.getByLabel('Subject')).toHaveValue('workload-client');
	await expect(card.getByLabel('Audience')).toHaveValue('https://pocket-id.example.com');

	// Saving the main client form must preserve credentials managed by the separate card
	const description = page.getByLabel('Description');
	await description.fill('Updated without replacing federated credentials');
	const clientForm = description.locator('xpath=ancestor::form');
	const formUpdate = page.waitForResponse(
		(response) =>
			response.request().method() === 'PUT' &&
			response.url().endsWith(`/api/oidc/clients/${client.id}`)
	);
	await clientForm.getByRole('button', { name: 'Save' }).click();
	expect((await formUpdate).ok()).toBeTruthy();

	await page.reload();
	await expect(card.getByLabel('Issuer')).toHaveValue('https://issuer.example.com');
});

test('Create new OIDC client secret', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto(`/settings/admin/oidc-clients/${oidcClient.id}`);

	await page.getByLabel('Create new client secret').click();
	await page.getByRole('button', { name: 'Generate' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'New client secret created successfully'
	);
	expect((await page.getByTestId('client-secret').textContent())?.length).toBe(32);
});

test('Delete OIDC client', async ({ page }) => {
	const oidcClient = oidcClients.nextcloud;
	await page.goto('/settings/admin/oidc-clients');

	await page
		.getByRole('row', { name: oidcClient.name })
		.getByRole('button', { name: 'Toggle menu' })
		.click();

	await page.getByRole('menuitem', { name: 'Delete' }).click();

	await page.getByRole('button', { name: 'Delete' }).click();

	await expect(page.locator('[data-type="success"]')).toHaveText(
		'OIDC client deleted successfully'
	);
	await expect(page.getByRole('row', { name: oidcClient.name })).not.toBeVisible();
});

test('Filter OIDC clients by PAR requirement', async ({ page, request }) => {
	const parClient = oidcClients.parClient;

	// Enable PAR on the PAR test client
	await request.put(`/api/oidc/clients/${parClient.id}`, {
		data: {
			...defaultTokenLifetimes,
			name: parClient.name,
			callbackURLs: [parClient.callbackUrl],
			logoutCallbackURLs: [],
			isPublic: false,
			pkceEnabled: false,
			pkceSupported: false,
			requiresReauthentication: false,
			requiresPushedAuthorizationRequests: true,
			credentials: { federatedIdentities: [] },
			isGroupRestricted: false
		}
	});

	await page.goto('/settings/admin/oidc-clients');

	// Open PAR filter and select "Yes"
	await page.getByTestId('facet-par-trigger').click();
	await page.getByTestId('facet-par-option-true').click();

	// Only the PAR client should be visible
	await expect(page.getByRole('row', { name: parClient.name })).toBeVisible();
	await expect(page.getByRole('row', { name: oidcClients.nextcloud.name })).not.toBeVisible();

	// Deselect "Yes" and select "No" to invert the filter
	await page.getByTestId('facet-par-option-true').click();
	await expect(page.getByRole('row', { name: oidcClients.nextcloud.name })).toBeVisible();
	await page.getByTestId('facet-par-option-false').click();

	// PAR client should be hidden, others visible
	await expect(page.getByRole('row', { name: oidcClients.nextcloud.name })).toBeVisible();
	await expect(page.getByRole('row', { name: parClient.name })).not.toBeVisible();
});

test('Update OIDC client allowed user groups', async ({ page }) => {
	await page.goto(`/settings/admin/oidc-clients/${oidcClients.nextcloud.id}`);
	await page.getByRole('tab', { name: 'Allowed user groups' }).click();

	await page.getByRole('button', { name: 'Restrict' }).click();

	await page.getByRole('row', { name: userGroups.designers.name }).getByRole('checkbox').click();
	await page.getByRole('row', { name: userGroups.developers.name }).getByRole('checkbox').click();

	await page.getByRole('button', { name: 'Save' }).click();

	await expect(page.getByText('Allowed user groups updated successfully')).toBeVisible();

	await page.reload();

	await expect(
		page.getByRole('row', { name: userGroups.designers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked');
	await expect(
		page.getByRole('row', { name: userGroups.developers.name }).getByRole('checkbox')
	).toHaveAttribute('data-state', 'checked');
});
