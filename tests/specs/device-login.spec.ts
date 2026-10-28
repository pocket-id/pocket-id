import { expect, test, type Browser } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';
import passkeyUtil from '../utils/passkey.util';

test.beforeEach(async () => {
	await cleanupBackend();
});

test('approves the QR link after requester review and fresh reauthentication', async ({
	browser,
	page
}) => {
	const waiting = await openWaitingDevice(browser, '/settings/apps');
	const loginCode = new RegExp(
		`${waiting.request.userCode.substring(0, 4)}\\s*–\\s*${waiting.request.userCode.substring(4, 8)}`
	);
	try {
		await (await passkeyUtil.init(page)).addPasskey();
		await expect(waiting.page.getByTestId('device-login-code')).toHaveText(loginCode);
		await expect(waiting.page.getByLabel('Device login QR code')).toBeVisible();

		await page.goto(waiting.request.verificationUriComplete);

		await expect(page.getByTestId('device-login-code')).toHaveText(loginCode);
		await expect(page.getByRole('heading', { name: /Chrome/ })).toBeVisible();
		const ipAddress = page.locator('dt', { hasText: 'IP address' }).locator('..').locator('dd');
		await expect(ipAddress).not.toHaveText('Unknown');

		const decisionWithoutReauthentication = await page.request.post(
			'/api/device-login/verification/decision',
			{
				data: { code: waiting.request.userCode, decision: 'approve' }
			}
		);
		expect(decisionWithoutReauthentication.status()).toBe(401);

		const reauthenticationRequest = page.waitForRequest('/api/webauthn/reauthenticate');
		await page.getByRole('button', { name: 'Approve' }).click();
		await reauthenticationRequest;
		await expect(page.getByRole('heading', { name: 'Signed in' })).toBeVisible();
		await waiting.page.waitForURL('/settings/apps');
	} finally {
		await waiting.context.close();
	}
});

test('authenticates a signed-out primary device before manual-code approval', async ({
	browser
}) => {
	const waiting = await openWaitingDevice(browser, '/settings/account');
	const primaryContext = await browser.newContext({
		baseURL: test.info().project.use.baseURL,
		storageState: { cookies: [], origins: [] }
	});
	const primaryPage = await primaryContext.newPage();

	try {
		await (await passkeyUtil.init(primaryPage)).addPasskey();
		await primaryPage.goto('/device');
		await primaryPage.getByRole('textbox', { name: 'Device code' }).fill(waiting.request.userCode);
		await primaryPage.getByRole('button', { name: 'Continue', exact: true }).click();

		await primaryPage.getByRole('button', { name: 'Approve' }).click();
		await expect(primaryPage.getByRole('heading', { name: 'Signed in' })).toBeVisible();
		await waiting.page.waitForURL('/settings/account');
	} finally {
		await primaryContext.close();
		await waiting.context.close();
	}
});

test('denies a pending request without passkey reauthentication', async ({ browser, page }) => {
	const waiting = await openWaitingDevice(browser);

	try {
		let reauthenticationCalled = false;
		await page.route('/api/webauthn/reauthenticate', async (route) => {
			reauthenticationCalled = true;
			await route.continue();
		});

		await page.goto(waiting.request.verificationUriComplete);
		await page.getByRole('button', { name: 'Deny' }).click();

		await expect(page.getByRole('heading', { name: 'Request denied' })).toBeVisible();
		await expect(
			waiting.page.getByText('The sign-in was denied on your other device.')
		).toBeVisible();
		expect(reauthenticationCalled).toBe(false);
	} finally {
		await waiting.context.close();
	}
});

async function openWaitingDevice(browser: Browser, redirect = '/settings') {
	const context = await browser.newContext({
		baseURL: test.info().project.use.baseURL,
		storageState: { cookies: [], origins: [] }
	});
	const page = await context.newPage();
	const createResponse = page.waitForResponse(
		(response) =>
			response.request().method() === 'POST' &&
			response.url().endsWith('/api/device-login/requests')
	);

	await page.goto(`/login/alternative?redirect=${encodeURIComponent(redirect)}`);
	await expect(page.getByText('Use another device', { exact: true })).toBeVisible();
	await page.getByText('Use another device', { exact: true }).click();

	const response = await createResponse;
	expect(response.status()).toBe(201);
	const request = await response.json();
	expect(request.userCode).toMatch(/^P[ABCDEFGHJKMNPQRSTUVWXYZ0123456789]{7}$/);
	return {
		context,
		page,
		request
	};
}
