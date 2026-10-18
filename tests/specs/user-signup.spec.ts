import test, { expect, type Page } from '@playwright/test';
import { signupTokens, userGroups, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

const signupModeOptions = {
	Disabled: 'Only admins',
	'Signup with token': 'People with a signup token',
	'Open Signup': 'Anyone'
};

async function setSignupMode(
	page: Page,
	mode: 'Disabled' | 'Signup with token' | 'Open Signup',
	signout = true
) {
	await page.goto('/settings/admin/application-configuration/signup');

	const option = page.getByRole('radio', { name: new RegExp(`^${signupModeOptions[mode]}`) });
	if (!(await option.isChecked())) {
		await option.click();
		await saveUnsavedChanges(page);
	}

	if (signout) {
		await page.context().clearCookies();
		await page.goto('/login');
	}
}

test.describe('Signup Token Creation', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await setSignupMode(page, 'Signup with token', false);
	});

	test('Create signup token', async ({ page }) => {
		await page.goto('/settings/admin/users');

		await page.getByRole('button', { name: 'Signup tokens' }).click();
		const sheet = page.getByRole('dialog', { name: 'Signup tokens' });
		await sheet.getByRole('button', { name: 'Create signup token' }).click();

		await page.getByLabel('Expires after').click();
		await page.getByRole('option', { name: '1 week' }).click();

		await page.getByLabel('Usage limit').fill('8');

		await page.getByLabel('Groups').click();
		await page.getByRole('option', { name: userGroups.developers.friendlyName }).click();
		await page.getByRole('option', { name: userGroups.designers.friendlyName }).click();
		await page.getByLabel('Groups').click();

		await page.getByRole('button', { name: 'Create signup token' }).click();
		await expect(page.getByRole('dialog', { name: 'Signup token created' })).toBeVisible();
		await expect(page.getByTestId('signup-token-link')).toContainText('/st/');
		await page.getByRole('button', { name: 'Done' }).click();

		const row = page.getByRole('dialog', { name: 'Signup tokens' }).getByText('0 of 8 used');
		await expect(row).toBeVisible();
		await expect(row).toContainText(userGroups.developers.friendlyName);
		await expect(row).toContainText(userGroups.designers.friendlyName);
	});
});

test.describe('Initial User Signup', () => {
	test.beforeEach(async ({ page }) => {
		await page.context().clearCookies();
	});

	test('Initial Signup - success flow', async ({ page }) => {
		await cleanupBackend({ skipSeed: true });
		await page.goto('/setup');
		await page.getByLabel('First name').fill('Jane');
		await page.getByLabel('Last name').fill('Smith');
		await page.getByLabel('Username').fill('janesmith');
		await page.getByLabel('Email').fill('jane.smith@test.com');
		await page.getByRole('button', { name: 'Create admin account' }).click();

		// The new admin picks the optional features before adding a passkey
		await page.waitForURL('/signup/welcome');
		await expect(
			page.getByRole('heading', { name: 'Which features do you want to enable?' })
		).toBeVisible();
		await page.getByRole('checkbox', { name: /^APIs/ }).click();
		await page.getByRole('button', { name: 'Continue' }).click();

		await page.waitForURL('/signup/add-passkey');
		await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();
		const features = await (await page.request.get('/api/features')).json();
		expect(features.apis.enabled).toBe(true);
		expect(features.apiKeys.enabled).toBe(false);
	});

	test('Initial Signup - concurrent requests create one administrator', async ({ request }) => {
		await cleanupBackend({ skipSeed: true });

		const requestCount = 20;
		const responses = await Promise.all(
			Array.from({ length: requestCount }, (_, index) =>
				request.post('/api/signup/setup', {
					data: {
						username: `race-admin-${index}`,
						email: `race-admin-${index}@example.invalid`,
						firstName: 'Race',
						lastName: `${index}`
					}
				})
			)
		);

		const successfulResponses = responses.filter((response) => response.status() === 200);
		const rejectedResponses = responses.filter((response) => response.status() === 409);

		expect(successfulResponses).toHaveLength(1);
		expect(rejectedResponses).toHaveLength(requestCount - 1);
		await expect(successfulResponses[0].json()).resolves.toMatchObject({ isAdmin: true });

		const usersResponse = await request.get('/api/users');
		expect(usersResponse.status()).toBe(200);
		const users = await usersResponse.json();
		expect(users.data).toHaveLength(1);
		expect(users.data[0]).toMatchObject({ isAdmin: true });
	});

	test('Initial Signup - setup route unavailable after completion', async ({ page }) => {
		await cleanupBackend();
		await page.goto('/setup');
		await expect(page.getByText('This page doesn’t exist.')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Create admin account' })).not.toBeVisible();
	});
});

test.describe('User Signup', () => {
	test.beforeEach(async () => await cleanupBackend());

	test.describe('Signup Flows', () => {
		test('Signup is disabled - shows error message', async ({ page }) => {
			await setSignupMode(page, 'Disabled');

			await page.goto('/signup');

			await expect(page.getByRole('heading', { name: 'Sign-ups are closed' })).toBeVisible();
		});

		test('Signup with token - success flow', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto(`/st/${signupTokens.valid.token}`);

			await page.getByLabel('First name').fill('John');
			await page.getByLabel('Last name').fill('Doe');
			await page.getByLabel('Username').fill('johndoe');
			await page.getByLabel('Email').fill('john.doe@test.com');

			await page.getByRole('button', { name: 'Create account' }).click();

			await page.waitForURL('/signup/add-passkey');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();

			const response = await page.request.get('/api/users/me').then((res) => res.json());
			expect(response.userGroups.map((g) => g.id)).toContain(userGroups.developers.id);
		});

		test('Signup with token - invalid token shows error', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto('/st/invalid-token-123');
			await page.getByLabel('First name').fill('Complete');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('completeuser');
			await page.getByLabel('Email').fill('complete.user@test.com');
			await page.getByRole('button', { name: 'Create account' }).click();

			await expect(page.getByText('Token is invalid or expired', { exact: true })).toBeVisible();
		});

		test('Signup with token - no token in URL shows error', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto('/signup');

			await expect(page.getByRole('heading', { name: 'Sign-ups are invite-only' })).toBeVisible();
		});

		test('Open signup - success flow', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await expect(page.getByText('You’ll add a passkey next.')).toBeVisible();

			await page.getByLabel('First name').fill('Jane');
			await page.getByLabel('Last name').fill('Smith');
			await page.getByLabel('Username').fill('janesmith');
			await page.getByLabel('Email').fill('jane.smith@test.com');

			await page.getByRole('button', { name: 'Create account' }).click();

			await page.waitForURL('/signup/add-passkey');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();
		});

		test('Open signup - validation errors', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByRole('button', { name: 'Create account' }).click();

			await expect(page.getByText('Invalid email address').first()).toBeVisible();
		});

		test('Open signup - duplicate email shows error', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByLabel('First name').fill('Test');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('testuser123');
			await page.getByLabel('Email').fill(users.tim.email);

			await page.getByRole('button', { name: 'Create account' }).click();

			await expect(page.getByText('Email is already in use', { exact: true })).toBeVisible();
		});

		test('Open signup - duplicate username shows error', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByLabel('First name').fill('Test');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill(users.tim.username);
			await page.getByLabel('Email').fill('newuser@test.com');

			await page.getByRole('button', { name: 'Create account' }).click();

			await expect(page.getByText('Username is already in use', { exact: true })).toBeVisible();
		});

		test('Complete signup flow with passkey creation', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');
			await page.getByLabel('First name').fill('Complete');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('completeuser');
			await page.getByLabel('Email').fill('complete.user@test.com');
			await page.getByRole('button', { name: 'Create account' }).click();

			await page.waitForURL('/signup/add-passkey');

			await (await passkeyUtil.init(page)).addPasskey('timNew');
			await page.getByRole('button', { name: 'Add passkey' }).click();

			await page.waitForURL('/settings/account');
			await expect(page.getByText('Add a backup passkey')).toBeVisible();
		});

		test('Dismiss passkey setup during signup for one day', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');
			await page.getByLabel('First name').fill('Skip');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('skipuser');
			await page.getByLabel('Email').fill('skip.user@test.com');
			await page.getByRole('button', { name: 'Create account' }).click();

			await page.waitForURL('/signup/add-passkey');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();

			// Cancelling the confirmation must keep passkey setup open
			await page.getByRole('button', { name: 'Skip for now' }).click();
			const dialog = page.getByRole('alertdialog', { name: 'Skip adding a passkey?' });
			await dialog.getByRole('button', { name: 'Cancel' }).click();
			await page.reload();
			await expect(page).toHaveURL('/signup/add-passkey');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();

			// Confirming dismissal allows settings access without creating a passkey
			const dismissedAt = Date.now();
			await page.clock.setFixedTime(dismissedAt);
			await page.getByRole('button', { name: 'Skip for now' }).click();
			await dialog.getByRole('button', { name: 'Skip for now' }).click();

			await page.waitForURL('/settings/account');
			await expect(page.getByText('No passkeys yet')).toBeVisible();
			await page.reload();
			await expect(page).toHaveURL('/settings/account');
			await expect(page.getByText('No passkeys yet')).toBeVisible();

			// Dismissal persists across settings pages until the one-day grace period expires
			await page.clock.setFixedTime(dismissedAt + 24 * 60 * 60 * 1000 - 1000);
			await page.goto('/settings/apps');
			await expect(page).toHaveURL('/settings/apps');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).not.toBeVisible();

			await page.clock.setFixedTime(dismissedAt + 24 * 60 * 60 * 1000);
			await page.reload();
			await expect(page).toHaveURL('/signup/add-passkey');
			await expect(page.getByRole('heading', { name: 'Add a passkey' })).toBeVisible();
		});

		test('Open signup with required email verification restricts the new account', async ({
			page
		}) => {
			// Required verification needs an SMTP server, but nothing listens on it, so the email sent at signup just fails
			await setAppConfigValues(page.request, {
				emailVerification: 'required',
				smtpHost: 'localhost',
				smtpFrom: 'pocket-id@test.com'
			});
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');
			await page.getByLabel('First name').fill('Pending');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('pendinguser');
			await page.getByLabel('Email').fill('pending.user@test.com');
			await page.getByRole('button', { name: 'Create account' }).click();

			await page.waitForURL('/signup/add-passkey');
			await page.getByRole('button', { name: 'Skip for now' }).click();
			await page
				.getByRole('alertdialog', { name: 'Skip adding a passkey?' })
				.getByRole('button', { name: 'Skip for now' })
				.click();

			await page.waitForURL('/settings/account');
			await expect(page.getByText('Verify your email address to use your account')).toBeVisible();
			await expect(page.getByText('your account will be deleted')).toBeVisible();

			await page.goto('/settings/apps');
			await expect(page).toHaveURL('/settings/account');
		});

		test('Token usage limit is enforced', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto(`/st/${signupTokens.fullyUsed.token}`);
			await page.getByLabel('First name').fill('Complete');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('completeuser');
			await page.getByLabel('Email').fill('complete.user@test.com');
			await page.getByRole('button', { name: 'Create account' }).click();

			await expect(page.getByText('Token is invalid or expired', { exact: true })).toBeVisible();
		});
	});
});
