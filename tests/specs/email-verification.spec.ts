import base, { expect, type APIRequestContext } from '@playwright/test';
import { emailVerificationTokens, oidcClients, users } from '../data';
import { setAppConfigValues } from '../utils/app-config.util';
import authUtil from '../utils/auth.util';
import { cleanupBackend } from '../utils/cleanup.util';
import { startSmtpSink, verificationLinkPath, type SmtpSink } from '../utils/smtp.util';
import { saveUnsavedChanges } from '../utils/unsaved-changes.util';

const smtpSinkPort = 18125;

const test = base.extend<{ smtp: SmtpSink }>({
	smtp: async ({}, use) => {
		const sink = await startSmtpSink(smtpSinkPort);
		try {
			await use(sink);
		} finally {
			await sink.close();
		}
	}
});

// Requires email verification and points Pocket ID at the SMTP sink, so the verification links can be followed
async function requireEmailVerificationWithSmtp(request: APIRequestContext) {
	await setAppConfigValues(request, {
		emailVerification: 'required',
		smtpHost: 'host.docker.internal',
		smtpPort: String(smtpSinkPort),
		smtpFrom: 'pocket-id@test.com',
		smtpTls: 'none'
	});
}

// Builds the query of an authorization code request for the Nextcloud client
function nextcloudAuthorizeParams() {
	const oidcClient = oidcClients.nextcloud;
	return new URLSearchParams({
		client_id: oidcClient.id,
		response_type: 'code',
		scope: 'openid profile email',
		redirect_uri: oidcClient.callbackUrl,
		state: 'nXx-6Qr-owc1SHBa',
		nonce: 'P1gN3PtpKHJgKUVcLpLjm'
	});
}

// Marks Craig's email as verified through the admin API
async function markCraigVerified(request: APIRequestContext) {
	const craig = await request.get(`/api/users/${users.craig.id}`).then((r) => r.json());
	const response = await request.put(`/api/users/${users.craig.id}`, {
		data: { ...craig, emailVerified: true }
	});
	expect(response.ok()).toBe(true);
}

test.describe('Required email verification', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await requireEmailVerificationWithSmtp(page.request);

		// Craig is a regular user whose email is not verified yet
		await authUtil.changeUser(page, 'craig');
	});

	test('Unverified user can only reach their account', async ({ page }) => {
		await page.goto('/settings/apps');
		await page.waitForURL('/settings/account');

		await expect(page.getByText('Verify your email address to use your account')).toBeVisible();
		await expect(page.getByRole('link', { name: 'My Apps' })).toHaveCount(0);

		const response = await page.request.get('/api/oidc/users/me/clients');
		expect(response.status()).toBe(403);
	});

	test('Unverified user cannot sign in to a client', async ({ page }) => {
		await page.goto(`/authorize?${nextcloudAuthorizeParams().toString()}`);

		await expect(page).toHaveURL(/\/interaction\/error\?error=/);
		await expect(
			page.getByRole('paragraph').filter({
				hasText: 'You must verify your email address before you can access this service.'
			})
		).toBeVisible();

		await page.getByRole('link', { name: 'Verify email address' }).click();
		await page.waitForURL('/settings/account');
	});

	test('Unverified user is stopped right after signing in during authorization', async ({
		page
	}) => {
		// Sign out so the sign-in happens on the interaction page, with the passkey the setup already registered for Craig
		await page.context().clearCookies();
		await page.goto(`/authorize?${nextcloudAuthorizeParams().toString()}`);
		await page.getByRole('button', { name: 'Sign in' }).click();

		// Completing the sign-in step already requires a verified email, so the consent screen is never shown
		await expect(
			page.getByText('You must verify your email address before you can sign in to applications.')
		).toBeVisible();
		await expect(page.getByTestId('scopes')).toHaveCount(0);

		await page.getByRole('link', { name: 'Verify email address' }).click();
		await page.waitForURL('/settings/account');
	});

	test('Verifying the email restores full access', async ({ page }) => {
		const token = emailVerificationTokens.find((t) => !t.expired)!.token;
		await page.goto(`/verify-email?token=${token}`);
		await page.waitForURL('/settings/account?emailVerificationState=success');

		await page.goto('/settings/apps');
		await expect(page).toHaveURL('/settings/apps');
		await expect(page.getByRole('link', { name: 'My Apps' })).toBeVisible();
	});
});

test.describe('Email change with required verification', () => {
	test.beforeEach(async ({ page }) => {
		await cleanupBackend();
		await requireEmailVerificationWithSmtp(page.request);
		await markCraigVerified(page.request);
		await authUtil.changeUser(page, 'craig');
	});

	test('A new email takes effect once it is confirmed', async ({ page, smtp }) => {
		const newEmail = 'craig.new@test.com';
		await page.goto('/settings/account');
		await page.getByLabel('Email').fill(newEmail);
		await saveUnsavedChanges(page);

		// The verified email stays in effect and the account keeps full access until the change is confirmed
		await expect(page.getByText('Confirm your new email address')).toBeVisible();
		await expect(page.getByLabel('Email')).toHaveValue(users.craig.email);
		await page.goto('/settings/apps');
		await expect(page).toHaveURL('/settings/apps');

		await page.goto(verificationLinkPath(smtp, newEmail));
		await page.waitForURL('/settings/account?emailVerificationState=success');

		await expect(page.getByLabel('Email')).toHaveValue(newEmail);
		await expect(page.getByText('Confirm your new email address')).toHaveCount(0);
	});

	test('A canceled email change can no longer be confirmed', async ({ page, smtp }) => {
		const newEmail = 'craig.canceled@test.com';
		await page.goto('/settings/account');
		await page.getByLabel('Email').fill(newEmail);
		await saveUnsavedChanges(page);

		await page.getByRole('button', { name: 'Cancel change' }).click();
		await expect(page.getByText('Email change canceled.')).toBeVisible();
		await expect(page.getByText('Confirm your new email address')).toHaveCount(0);

		await page.goto(verificationLinkPath(smtp, newEmail));
		await page.waitForURL(
			'/settings/account?emailVerificationState=Email+verification+token+is+invalid'
		);
		await expect(page.getByLabel('Email')).toHaveValue(users.craig.email);
	});
});

test('Signing up sends a verification link that unlocks the account', async ({ page, smtp }) => {
	await cleanupBackend();
	await requireEmailVerificationWithSmtp(page.request);
	await setAppConfigValues(page.request, { allowUserSignups: 'open' });
	await page.context().clearCookies();

	const email = 'verified.signup@test.com';
	await page.goto('/signup');
	await page.getByLabel('First name').fill('Verified');
	await page.getByLabel('Last name').fill('Signup');
	await page.getByLabel('Username').fill('verifiedsignup');
	await page.getByLabel('Email').fill(email);
	await page.getByRole('button', { name: 'Sign Up' }).click();

	await page.waitForURL('/signup/add-passkey');
	await page.getByRole('button', { name: 'Skip for now' }).click();
	await page
		.getByRole('alertdialog', { name: 'Skip Passkey Setup' })
		.getByRole('button', { name: 'Skip for now' })
		.click();
	await page.waitForURL('/settings/account');

	await page.goto(verificationLinkPath(smtp, email));
	await page.waitForURL('/settings/account?emailVerificationState=success');

	await page.goto('/settings/apps');
	await expect(page).toHaveURL('/settings/apps');
});
