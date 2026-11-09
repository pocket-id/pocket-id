import { expect, test, type APIRequestContext } from '@playwright/test';
import { users } from '../data';
import { cleanupBackend } from '../utils/cleanup.util';

const initialize = {
	jsonrpc: '2.0',
	id: 1,
	method: 'initialize',
	params: {
		protocolVersion: '2025-11-25',
		capabilities: {},
		clientInfo: { name: 'e2e', version: '1' }
	}
};

async function rpc(api: APIRequestContext, method: string, params?: object) {
	const response = await api.post('/mcp', {
		headers: { Accept: 'application/json, text/event-stream' },
		data: { jsonrpc: '2.0', id: 2, method, params }
	});
	expect(response.status()).toBe(200);
	return (await response.json()).result;
}

test.describe('MCP server', () => {
	test.beforeEach(async () => {
		await cleanupBackend();
	});

	test('Unauthenticated clients are pointed to the protected resource metadata', async ({
		playwright,
		baseURL
	}) => {
		const api = await playwright.request.newContext({
			baseURL,
			storageState: { cookies: [], origins: [] }
		});

		const response = await api.post('/mcp', {
			headers: { Accept: 'application/json, text/event-stream' },
			data: initialize
		});
		expect(response.status()).toBe(401);
		const challenge = response.headers()['www-authenticate'];
		const metadataURL = challenge.match(/resource_metadata="([^"]+)"/)![1];

		// MCP clients refuse metadata whose resource isn't the endpoint URL
		const metadata = await (await api.get(metadataURL)).json();
		expect(metadata.resource).toBe(`${baseURL}/mcp`);
		expect(metadata.authorization_servers).toEqual([baseURL]);
		expect(metadata.scopes_supported).toContain('users:read');

		await api.dispose();
	});

	test('An API key only sees and calls the tools of its scopes', async ({
		page,
		playwright,
		baseURL
	}) => {
		await page.goto('/settings/account/api-keys');
		await page.getByRole('button', { name: 'Create API key' }).click();
		await page.getByLabel('Name').fill('Assistant');
		await page.getByRole('tab', { name: 'Custom' }).click();
		await page
			.getByRole('radiogroup', { name: 'Users', exact: true })
			.getByRole('radio', { name: 'Read' })
			.click();
		await page.getByRole('dialog').getByRole('button', { name: 'Create API key' }).click();
		const sheet = page.getByRole('dialog', { name: 'API key created' });
		const token = (await sheet.getByRole('textbox', { name: 'API key' }).textContent())!.trim();

		const api = await playwright.request.newContext({
			baseURL,
			storageState: { cookies: [], origins: [] },
			extraHTTPHeaders: { 'X-API-Key': token }
		});

		const tools = await rpc(api, 'tools/list');
		const names = tools.tools.map((tool: { name: string }) => tool.name);
		expect(names).toContain('search_users');
		expect(names).toContain('search_docs');
		expect(names).not.toContain('create_user');
		expect(names).not.toContain('search_groups');

		const result = await rpc(api, 'tools/call', {
			name: 'search_users',
			arguments: { query: users.craig.username }
		});
		expect(result.isError).toBeFalsy();
		expect(result.structuredContent.data).toEqual([
			expect.objectContaining({ id: users.craig.id, email: users.craig.email })
		]);

		await api.dispose();
	});
});
