import type { Page } from '@playwright/test';

async function getUserCode(page: Page, clientId: string, clientSecret: string) {
	const response = await page.request
		.post('/api/oidc/device/authorize', {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			},
			form: {
				client_id: clientId,
				client_secret: clientSecret,
				scope: 'openid profile email'
			}
		})
		.then((r) => r.json());

	return response.user_code;
}

export default {
	getUserCode
};
