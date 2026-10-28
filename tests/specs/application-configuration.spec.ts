import { expect, test } from '@playwright/test';
import { users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { saveUnsavedChanges, unsavedChangesBar } from '../utils/unsaved-changes.util';

const configUrl = '/settings/admin/application-configuration';

test.beforeEach(async () => {
	await cleanupBackend();
});

test('Update general configuration', async ({ page }) => {
	await page.goto(configUrl);
	await page.waitForURL(`${configUrl}/general`);

	await page.getByLabel('Application name', { exact: true }).fill('Updated Name');
	await page.getByLabel('Session duration', { exact: true }).fill('30');

	await page.getByRole('button', { name: 'Home page' }).click();
	await page.getByRole('option', { name: 'My apps' }).click();
	await expect(unsavedChangesBar(page)).toContainText('3 unsaved changes');

	await saveUnsavedChanges(page);

	await page.reload();

	await expect(page.getByLabel('Application name', { exact: true })).toHaveValue('Updated Name');
	await expect(page.getByLabel('Session duration', { exact: true })).toHaveValue('30');

	await page.getByRole('link', { name: 'Updated Name', exact: true }).click();
	await page.waitForURL('/settings/apps');
});

test('Disable automatic client secret creation', async ({ page }) => {
	await page.goto(`${configUrl}/oidc`);
	const autoCreateSecret = page.getByRole('switch', { name: 'Create a secret for new clients' });
	await expect(autoCreateSecret).toBeChecked();
	await autoCreateSecret.click();
	await saveUnsavedChanges(page);

	await page.reload();
	await expect(autoCreateSecret).not.toBeChecked();

	const response = await page.request.post('/api/oidc/clients', {
		data: { name: 'No Automatic Secret' }
	});
	expect(response.status()).toBe(201);
	const createdClient = await response.json();
	expect(createdClient.createdSecret).toBeUndefined();
	expect(createdClient.credentials.secrets ?? []).toHaveLength(0);
});

test('Turning on a feature adds its tabs', async ({ page }) => {
	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await expect(page.getByRole('tab', { name: 'General' })).toBeVisible();
	await expect(page.getByRole('tab', { name: /^Custom claims/ })).toHaveCount(0);

	await page.goto(`${configUrl}/features`);
	await page.getByRole('switch', { name: 'Custom claims' }).click();
	await expect(page.locator('[data-type="success"]')).toHaveText('Custom claims turned on');
	await expect(page.getByRole('switch', { name: 'Custom claims' })).toBeChecked();

	await page.goto(`/settings/admin/users/${users.craig.id}`);
	await expect(page.getByRole('tab', { name: /^Custom claims/ })).toBeVisible();
});

test('A feature in use cannot be turned off', async ({ page }) => {
	await page.goto(`${configUrl}/features`);

	const apis = page.getByRole('switch', { name: 'APIs' });
	await expect(apis).toBeChecked();
	await expect(apis).toBeDisabled();
	await expect(page.getByText('Stays on while any API exists.')).toBeVisible();
	await expect(page.getByRole('link', { name: 'APIs' })).toBeVisible();
});

test('Unsaved changes keep the current tab until they are saved or discarded', async ({ page }) => {
	await page.goto(`${configUrl}/general`);
	await page.getByLabel('Application name', { exact: true }).fill('Kept On Tab');

	await page.getByRole('tab', { name: 'Email' }).click();
	await expect(page.getByRole('tab', { name: 'General' })).toHaveAttribute('aria-selected', 'true');
	await expect(page).toHaveURL(`${configUrl}/general`);

	await saveUnsavedChanges(page);
	await page.getByRole('tab', { name: 'Email' }).click();
	await page.waitForURL(`${configUrl}/email`);
	await page.getByLabel('Host').fill('smtp.combined.test');
	await page.getByLabel('Port').fill('587');
	await page.getByLabel('From address').fill('combined@example.com');
	await saveUnsavedChanges(page);

	await page.getByRole('tab', { name: 'General' }).click();
	await expect(page.getByLabel('Application name', { exact: true })).toHaveValue('Kept On Tab');
	await page.getByRole('tab', { name: 'Email' }).click();
	await expect(page.getByLabel('Host')).toHaveValue('smtp.combined.test');
});

test('Invalid email configuration is not saved', async ({ page }) => {
	let updateRequests = 0;
	page.on('request', (request) => {
		if (
			request.method() === 'PUT' &&
			new URL(request.url()).pathname === '/api/application-configuration'
		) {
			updateRequests++;
		}
	});

	await page.goto(`${configUrl}/email`);
	await page.getByLabel('Host').fill('smtp.validation.test');
	await page.getByLabel('From address').fill('validation@example.com');

	await page.getByRole('button', { name: 'Save', exact: true }).click();

	const smtpPort = page.getByLabel('Port');
	await expect(
		unsavedChangesBar(page).getByText('Couldn’t save. Check the highlighted fields.')
	).toBeVisible();
	await expect(smtpPort).toHaveAttribute('aria-invalid', 'true');
	await expect(smtpPort).toBeFocused();
	expect(updateRequests).toBe(0);

	await smtpPort.fill('587');
	await expect(smtpPort).toHaveAttribute('aria-invalid', 'false');
	await saveUnsavedChanges(page);

	await page.reload();
	await expect(page.getByLabel('Host')).toHaveValue('smtp.validation.test');
	await expect(page.getByLabel('Port')).toHaveValue('587');
	await expect(page.getByLabel('From address')).toHaveValue('validation@example.com');
});

test.describe('Update sign-up configuration', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto(`${configUrl}/signup`);
	});

	test('should save sign up mode', async ({ page }) => {
		await page.getByRole('radio', { name: /^Anyone/ }).click();

		await saveUnsavedChanges(page);

		await page.reload();

		await expect(page.getByRole('radio', { name: /^Anyone/ })).toBeChecked();
	});

	test('should save default user groups for new signups', async ({ page }) => {
		const developersOption = page.getByRole('option', { name: 'Developers' });
		const designersOption = page.getByRole('option', { name: 'Designers' });

		await page.getByRole('combobox', { name: 'Default groups' }).click();
		await developersOption.click();
		await expect(developersOption).toBeChecked();
		await designersOption.click();
		await expect(designersOption).toBeChecked();
		await page.keyboard.press('Escape');

		await saveUnsavedChanges(page);

		await page.reload();

		await page.getByRole('combobox', { name: 'Default groups' }).click();

		await expect(page.getByRole('option', { name: 'Developers' })).toBeChecked();
		await expect(page.getByRole('option', { name: 'Designers' })).toBeChecked();
	});

	test('should save default custom claims for new signups', async ({ page }) => {
		await setAppConfigValues(page.request, { enabledFeatures: '["customClaims"]' });
		await page.reload();
		await page.getByRole('button', { name: 'Add custom claim' }).click();
		await page.getByPlaceholder('Key').fill('test-claim');
		await page.getByPlaceholder('Value').fill('test-value');
		await page.getByRole('button', { name: 'Add another' }).click();
		await page.getByPlaceholder('Key').nth(1).fill('another-claim');
		await page.getByPlaceholder('Value').nth(1).fill('another-value');

		await saveUnsavedChanges(page);

		await page.reload();

		await expect(page.getByPlaceholder('Key').first()).toHaveValue('test-claim');
		await expect(page.getByPlaceholder('Value').first()).toHaveValue('test-value');
		await expect(page.getByPlaceholder('Key').nth(1)).toHaveValue('another-claim');
		await expect(page.getByPlaceholder('Value').nth(1)).toHaveValue('another-value');
	});
});

test('Update passkey configuration', async ({ page }) => {
	await page.goto(`${configUrl}/passkeys`);

	const userVerification = page.getByLabel('User verification');
	const authenticatorType = page.getByLabel('Allowed authenticator type');
	const allowSyncedPasskeys = page.getByRole('switch', { name: 'Allow synced passkeys' });

	await expect(userVerification).toContainText('Required');
	await userVerification.click();
	await page.getByRole('option', { name: /^Preferred/ }).click();

	await expect(authenticatorType).toContainText('Any passkey');
	await authenticatorType.click();
	await page.getByRole('option', { name: /^Security keys only/ }).click();

	await expect(allowSyncedPasskeys).toBeChecked();
	await allowSyncedPasskeys.click();
	await expect(
		page.getByText('People whose only passkey is synced won’t be able to sign in.')
	).toBeVisible();

	await saveUnsavedChanges(page);

	const registrationResponse = await page.request.get('/api/webauthn/register/start');
	expect(registrationResponse.ok()).toBeTruthy();
	await expect(registrationResponse.json()).resolves.toMatchObject({
		authenticatorSelection: {
			authenticatorAttachment: 'cross-platform',
			userVerification: 'preferred'
		}
	});

	const loginResponse = await page.request.get('/api/webauthn/login/start');
	expect(loginResponse.ok()).toBeTruthy();
	await expect(loginResponse.json()).resolves.toMatchObject({
		userVerification: 'preferred'
	});

	await page.reload();

	await expect(page.getByLabel('User verification')).toContainText('Preferred');
	await expect(page.getByLabel('Allowed authenticator type')).toContainText('Security keys only');
	await expect(page.getByRole('switch', { name: 'Allow synced passkeys' })).not.toBeChecked();
});

test('Update email configuration', async ({ page }) => {
	await page.goto(`${configUrl}/email`);

	await page.getByLabel('Host').fill('smtp.gmail.com');
	await page.getByLabel('Port').fill('587');
	await page.getByLabel('Username').fill('test@gmail.com');
	await page.getByLabel('Password').fill('password');
	await page.getByLabel('From address').fill('test@gmail.com');
	await page.getByLabel('Sign-in notifications').click();
	await page.getByRole('option', { name: /^Unrecognized browser/ }).click();
	await page.getByLabel('Login codes requested by people').click();
	await page.getByLabel('Login codes sent by admins').click();
	await page.getByLabel('API key expiry warnings').click();
	await page.getByLabel('Email verification', { exact: true }).click();
	await page.getByRole('option', { name: /^Required/ }).click();

	await saveUnsavedChanges(page);

	await page.reload();

	await expect(page.getByLabel('Host')).toHaveValue('smtp.gmail.com');
	await expect(page.getByLabel('Port')).toHaveValue('587');
	await expect(page.getByLabel('Username')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('Password')).toHaveValue('password');
	await expect(page.getByLabel('From address')).toHaveValue('test@gmail.com');
	await expect(page.getByLabel('Sign-in notifications')).toContainText('Unrecognized browser');
	await expect(page.getByLabel('Login codes requested by people')).toBeChecked();
	await expect(page.getByLabel('Login codes sent by admins')).toBeChecked();
	await expect(page.getByLabel('API key expiry warnings')).toBeChecked();
	await expect(page.getByLabel('Email verification', { exact: true })).toContainText('Required');
});

test('Required email verification turns off treating new addresses as verified', async ({
	page
}) => {
	await page.goto(`${configUrl}/email`);

	const treatAsVerified = page.getByLabel('Treat new addresses as verified');
	await treatAsVerified.click();
	await expect(treatAsVerified).toBeChecked();

	await page.getByLabel('Email verification', { exact: true }).click();
	await page.getByRole('option', { name: /^Required/ }).click();

	await expect(treatAsVerified).not.toBeChecked();
	await expect(treatAsVerified).toBeDisabled();

	await page.getByLabel('Email verification', { exact: true }).click();
	await page.getByRole('option', { name: /^Optional/ }).click();

	await expect(treatAsVerified).toBeEnabled();
});

test('Save LDAP configuration while LDAP remains disabled', async ({ page }) => {
	await page.goto(`${configUrl}/ldap`);

	// isVisible does not wait, so wait for the status region that renders together with its action
	await expect(page.getByRole('region', { name: 'Status' })).toBeVisible();
	const disableButton = page.getByRole('button', { name: 'Disable LDAP…' });
	if (await disableButton.isVisible()) {
		await disableButton.click();
		await page.getByRole('button', { name: 'Disable LDAP', exact: true }).click();
		await expect(page.getByRole('button', { name: 'Enable LDAP…' })).toBeVisible();
	}

	const softDeleteUsers = page.getByRole('switch', { name: 'Keep people removed from LDAP' });
	const originalValue = await softDeleteUsers.isChecked();
	await softDeleteUsers.click();
	await expect(unsavedChangesBar(page)).toContainText(/\d+ unsaved changes?/);

	await page.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(softDeleteUsers).toBeChecked({ checked: originalValue });

	await softDeleteUsers.click();
	await saveUnsavedChanges(page);

	await page.reload();
	await expect(page.getByRole('button', { name: 'Enable LDAP…' })).toBeVisible();
	await expect(page.getByText('Not connected', { exact: true })).toBeVisible();
	await expect(page.getByRole('switch', { name: 'Keep people removed from LDAP' })).toBeChecked({
		checked: !originalValue
	});
});

test.describe('Update application images', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto(`${configUrl}/branding`);
	});

	test('should detect image removals as unsaved changes', async ({ page }) => {
		await page
			.getByLabel('Replace sign-in background', { exact: true })
			.setInputFiles('resources/images/clouds.jpg');
		await saveUnsavedChanges(page);

		const removeBackground = page.getByRole('button', {
			name: 'Remove sign-in background',
			exact: true
		});
		await removeBackground.click();
		await expect(removeBackground).toBeDisabled();
		await expect(unsavedChangesBar(page)).toContainText(/\d+ unsaved changes?/);

		await page.getByRole('button', { name: 'Discard', exact: true }).click();
		await expect(removeBackground).toBeEnabled();
	});

	test('should upload images and reset custom logos', async ({ page }) => {
		await page
			.getByLabel('Replace favicon', { exact: true })
			.setInputFiles('resources/images/w3-schools-favicon.ico');
		await page
			.getByLabel('Replace light logo', { exact: true })
			.setInputFiles('resources/images/pingvin-share-logo.png');
		await page
			.getByLabel('Replace dark logo', { exact: true })
			.setInputFiles('resources/images/cloud-logo.png');
		await page
			.getByLabel('Replace email logo', { exact: true })
			.setInputFiles('resources/images/pingvin-share-logo.png');
		await page
			.getByLabel('Replace default profile picture', { exact: true })
			.setInputFiles('resources/images/pingvin-share-logo.png');
		await page
			.getByLabel('Replace sign-in background', { exact: true })
			.setInputFiles('resources/images/clouds.jpg');
		await saveUnsavedChanges(page);

		await page.request
			.get('/api/application-images/favicon')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/logo?light=true')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/logo?light=false')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/email')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/background')
			.then((res) => expect.soft(res.status()).toBe(200));

		await page.getByRole('button', { name: 'Reset to default, light logo', exact: true }).click();
		await page.getByRole('button', { name: 'Reset to default, dark logo', exact: true }).click();
		const logoDeleteResponses = [true, false].map((light) =>
			page.waitForResponse((response) => {
				const url = new URL(response.url());
				return (
					response.request().method() === 'DELETE' &&
					url.pathname === '/api/application-images/logo' &&
					url.searchParams.get('light') === String(light)
				);
			})
		);
		await saveUnsavedChanges(page);
		for (const response of await Promise.all(logoDeleteResponses)) {
			expect(response.status()).toBe(204);
		}

		// Without a custom logo the endpoint falls back to the logo bundled with Pocket ID
		await page.request
			.get('/api/application-images/logo?light=true')
			.then((res) => expect.soft(res.status()).toBe(200));
		await page.request
			.get('/api/application-images/logo?light=false')
			.then((res) => expect.soft(res.status()).toBe(200));

		// The bundled logo can be skipped to check whether a custom logo is set
		await page.request
			.get('/api/application-images/logo?light=true&default=false')
			.then((res) => expect.soft(res.status()).toBe(404));
		await page.request
			.get('/api/application-images/logo?light=false&default=false')
			.then((res) => expect.soft(res.status()).toBe(404));
	});

	test('should only allow png/jpeg for email logo', async ({ page }) => {
		const emailLogoInput = page.getByLabel('Replace email logo', { exact: true });

		await emailLogoInput.setInputFiles('resources/images/cloud-logo.svg');
		await page.getByRole('button', { name: 'Save', exact: true }).click();

		await expect(
			unsavedChangesBar(page).getByText('File must be of type PNG or JPEG', { exact: true })
		).toBeVisible();
	});
});
