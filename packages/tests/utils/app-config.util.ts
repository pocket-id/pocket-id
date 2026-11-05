import { expect, type APIRequestContext } from '@playwright/test';

// Overrides application configuration values through the API, keeping every other value unchanged
// The request context must be authenticated as an admin
export async function setAppConfigValues(
	request: APIRequestContext,
	values: Record<string, string>
) {
	const configResponse = await request.get('/api/application-configuration/all');
	expect(configResponse.ok()).toBe(true);

	const config = Object.fromEntries(
		((await configResponse.json()) as Array<{ key: string; value: string }>).map(
			({ key, value }) => [key, value]
		)
	);

	const updateResponse = await request.put('/api/application-configuration', {
		data: { ...config, ...values }
	});
	expect(updateResponse.ok()).toBe(true);
}
