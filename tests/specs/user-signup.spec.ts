import test, { expect, type Page } from '@playwright/test';
import { signupTokens, userGroups, users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

async function setSignupMode(
	page: Page,
	mode: 'Disabled' | 'Signup with token' | 'Open Signup',
	signout = true
) {
	await page.goto('/settings/admin/application-configuration');

	await page.getByRole('button', { name: 'Expand card' }).nth(1).click();
	await page.getByRole('button', { name: 'Enable User Signups' }).click();
	await page.getByRole('option', { name: mode }).click();
	await page.getByRole('button', { name: 'Save' }).nth(1).click();

	await expect(page.locator('[data-type="success"]').last()).toHaveText(
		'User creation settings updated successfully.'
	);

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

		await page.getByLabel('Create options').getByRole('button').click();
		await page.getByRole('menuitem', { name: 'Create Signup Token' }).click();
		await page.getByLabel('Expiration').click();
		await page.getByRole('option', { name: 'week' }).click();

		await page.getByLabel('Usage Limit').fill('8');

		await page.getByLabel('User Groups').click();
		await page.getByRole('option', { name: userGroups.designers.name }).click();
		await page.getByLabel('User Groups').click();

		await page.getByRole('button', { name: 'Create', exact: true }).click();
		await page.getByRole('button', { name: 'Close' }).click();

		await page.getByLabel('Create options').getByRole('button').click();
		await page.getByRole('menuitem', { name: 'View Active Signup Tokens' }).click();
		await page.getByLabel('Manage Signup Tokens').getByRole('button', { name: 'View' }).click();

		await page.getByRole('menuitemcheckbox', { name: 'User Groups' }).click();

		const row = page.getByRole('row').last();
		await expect(row.getByRole('cell', { name: '0 of 8' })).toBeVisible();
		const dateInAWeek = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toLocaleDateString('en-US');
		await expect(row.getByRole('cell', { name: dateInAWeek })).toBeVisible();
		await expect(row.getByRole('cell', { name: userGroups.developers.name })).toBeVisible();
	});
});

test.describe('Initial User Signup', () => {
	test.beforeEach(async ({ page }) => {
		await page.context().clearCookies();
	});

	test('Initial Signup - success flow', async ({ page }) => {
		await cleanupBackend(true);
		await page.goto('/setup');
		await page.getByLabel('First name').fill('Jane');
		await page.getByLabel('Last name').fill('Smith');
		await page.getByLabel('Username').fill('janesmith');
		await page.getByLabel('Email').fill('jane.smith@test.com');
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await page.waitForURL('/signup/add-passkey');
		await expect(page.getByText('Set up your passkey')).toBeVisible();
	});

	test('Initial Signup - setup already completed', async ({ page }) => {
		await cleanupBackend();
		await page.goto('/setup');
		await page.getByLabel('First name').fill('Test');
		await page.getByLabel('Last name').fill('User');
		await page.getByLabel('Username').fill('testuser123');
		await page.getByLabel('Email').fill(users.tim.email);
		await page.getByRole('button', { name: 'Sign Up' }).click();
		await expect(page.getByText('Setup already completed')).toBeVisible();
	});
});

test.describe('User Signup', () => {
	test.beforeEach(async () => await cleanupBackend());

	test.describe('Signup Flows', () => {
		test('Signup is disabled - shows error message', async ({ page }) => {
			await setSignupMode(page, 'Disabled');

			await page.goto('/signup');

			await expect(page.getByText('User signups are currently disabled')).toBeVisible();
		});

		test('Signup with token - success flow', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto(`/st/${signupTokens.valid.token}`);

			await page.getByLabel('First name').fill('John');
			await page.getByLabel('Last name').fill('Doe');
			await page.getByLabel('Username').fill('johndoe');
			await page.getByLabel('Email').fill('john.doe@test.com');

			await page.getByRole('button', { name: 'Sign Up' }).click();

			await page.waitForURL('/signup/add-passkey');
			await expect(page.getByText('Set up your passkey')).toBeVisible();

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
			await page.getByRole('button', { name: 'Sign Up' }).click();

			await expect(page.getByText('Token is invalid or expired.')).toBeVisible();
		});

		test('Signup with token - no token in URL shows error', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto('/signup');

			await expect(
				page.getByText('A valid signup token is required to create an account.')
			).toBeVisible();
		});

		test('Open signup - success flow', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await expect(page.getByText('Create your account to get started')).toBeVisible();

			await page.getByLabel('First name').fill('Jane');
			await page.getByLabel('Last name').fill('Smith');
			await page.getByLabel('Username').fill('janesmith');
			await page.getByLabel('Email').fill('jane.smith@test.com');

			await page.getByRole('button', { name: 'Sign Up' }).click();

			await page.waitForURL('/signup/add-passkey');
			await expect(page.getByText('Set up your passkey')).toBeVisible();
		});

		test('Open signup - validation errors', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByRole('button', { name: 'Sign Up' }).click();

			await expect(page.getByText('Invalid email address').first()).toBeVisible();
		});

		test('Open signup - duplicate email shows error', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByLabel('First name').fill('Test');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('testuser123');
			await page.getByLabel('Email').fill(users.tim.email);

			await page.getByRole('button', { name: 'Sign Up' }).click();

			await expect(page.getByText('Email is already in use.')).toBeVisible();
		});

		test('Open signup - duplicate username shows error', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');

			await page.getByLabel('First name').fill('Test');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill(users.tim.username);
			await page.getByLabel('Email').fill('newuser@test.com');

			await page.getByRole('button', { name: 'Sign Up' }).click();

			await expect(page.getByText('Username is already in use.')).toBeVisible();
		});

		test('Complete signup flow with passkey creation', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');
			await page.getByLabel('First name').fill('Complete');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('completeuser');
			await page.getByLabel('Email').fill('complete.user@test.com');
			await page.getByRole('button', { name: 'Sign Up' }).click();

			await page.waitForURL('/signup/add-passkey');

			await (await passkeyUtil.init(page)).addPasskey('timNew');
			await page.getByRole('button', { name: 'Add Passkey' }).click();

			await page.waitForURL('/settings/account');
			await expect(page.getByText('Single Passkey Configured')).toBeVisible();
		});

		test('Skip passkey creation during signup', async ({ page }) => {
			await setSignupMode(page, 'Open Signup');

			await page.goto('/signup');
			await page.getByLabel('First name').fill('Skip');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('skipuser');
			await page.getByLabel('Email').fill('skip.user@test.com');
			await page.getByRole('button', { name: 'Sign Up' }).click();

			await page.waitForURL('/signup/add-passkey');

			await page.getByRole('button', { name: 'Skip for now' }).click();

			await expect(page.getByText('Skip Passkey Setup')).toBeVisible();
			await page.getByRole('button', { name: 'Skip for now' }).nth(1).click();

			await page.waitForURL('/settings/account');
			await expect(page.getByText('Passkey missing')).toBeVisible();
		});

		test('Token usage limit is enforced', async ({ page }) => {
			await setSignupMode(page, 'Signup with token');

			await page.goto(`/st/${signupTokens.fullyUsed.token}`);
			await page.getByLabel('First name').fill('Complete');
			await page.getByLabel('Last name').fill('User');
			await page.getByLabel('Username').fill('completeuser');
			await page.getByLabel('Email').fill('complete.user@test.com');
			await page.getByRole('button', { name: 'Sign Up' }).click();

			await expect(page.getByText('Token is invalid or expired.')).toBeVisible();
		});
	});
});
