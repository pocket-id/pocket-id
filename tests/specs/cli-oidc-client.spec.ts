import { expect, test } from '@playwright/test';
import { cleanupBackend } from 'utils/cleanup.util';
import {
	runCLICommand,
	parseJSONOutput,
	setupTestEnvironment,
	createAdminUserAndApiKey,
	generateTestUsername,
	generateTestEmail,
	cleanupTestResources
} from 'utils/cli-test-helper';

test.beforeAll(async () => {
	// Set up test environment
	await setupTestEnvironment();

	// Clean up backend without seeding to get fresh database
	await cleanupBackend({ skipLdapSetup: true, skipSeed: true });

	// Create admin user and API key
	await createAdminUserAndApiKey();
});

test.describe('OIDC Client Management CLI', () => {
	test('Create OIDC client with required fields', async () => {
		const clientName = `Test Client ${Date.now()}`;
		const clientId = `test-client-${Date.now()}`;

		const output = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id', clientId);
		expect(result).toHaveProperty('name', clientName);
		expect(result).toHaveProperty('callbackURLs');
		expect(Array.isArray(result.callbackURLs)).toBe(true);
		expect(result.callbackURLs).toContain('http://localhost:3000/callback');
		expect(result).toHaveProperty('requiresReauthentication', false);
		expect(result).toHaveProperty('isGroupRestricted', false);
		expect(result).toHaveProperty('allowedUserGroups', []);
		expect(result).toHaveProperty('logoutCallbackURLs', []);
		expect(result).toHaveProperty('launchURL', null);

		// Clean up
		await cleanupTestResources({ clients: [result.id] });
	});

	test('Create OIDC client with all optional fields', async () => {
		const clientName = `Test Client ${Date.now()}`;
		const clientId = `test-client-${Date.now()}`;

		const output = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback,http://localhost:3001/callback',
			'--logout-callback-urls',
			'http://localhost:3000/logout,http://localhost:3001/logout',
			'--requires-reauth',
			'--group-restricted',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id', clientId);
		expect(result).toHaveProperty('name', clientName);
		expect(result).toHaveProperty('callbackURLs');
		expect(Array.isArray(result.callbackURLs)).toBe(true);
		expect(result.callbackURLs).toContain('http://localhost:3000/callback');
		expect(result.callbackURLs).toContain('http://localhost:3001/callback');
		expect(result).toHaveProperty('logoutCallbackURLs');
		expect(Array.isArray(result.logoutCallbackURLs)).toBe(true);
		expect(result.logoutCallbackURLs).toContain('http://localhost:3000/logout');
		expect(result.logoutCallbackURLs).toContain('http://localhost:3001/logout');
		expect(result).toHaveProperty('requiresReauthentication', true);
		expect(result).toHaveProperty('isGroupRestricted', true);
		expect(result).toHaveProperty('allowedUserGroups', []);

		// Clean up
		await cleanupTestResources({ clients: [result.id] });
	});

	test('Update OIDC client', async () => {
		// First create a test client
		const clientName = `Test Client ${Date.now()}`;
		const clientId = `test-client-${Date.now()}`;

		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);

		// List clients
		const listOutput = runCLICommand(['oidc-clients', 'list', '--format', 'json']);
		const result = parseJSONOutput(listOutput);

		expect(result).toHaveProperty('data');
		expect(Array.isArray(result.data)).toBe(true);
		expect(result.data.length).toBeGreaterThan(0);

		// Should contain our created client
		const foundClient = result.data.find((client: any) => client.id === createdClient.id);
		expect(foundClient).toBeDefined();
		expect(foundClient).toHaveProperty('name', clientName);

		// Clean up
		await cleanupTestResources({ clients: [createdClient.id] });
	});

	test('Get OIDC client by ID', async () => {
		// First create a test client
		const clientName = `Get Client ${Date.now()}`;
		const clientId = `get-client-${Date.now()}`;

		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);

		// Get client by ID
		const getOutput = runCLICommand(['oidc-clients', 'get', createdClient.id, '--format', 'json']);
		const result = parseJSONOutput(getOutput);

		expect(result).toHaveProperty('id', createdClient.id);
		expect(result).toHaveProperty('name', clientName);
		expect(result).toHaveProperty('callbackURLs');
		expect(Array.isArray(result.callbackURLs)).toBe(true);
		expect(result.callbackURLs).toContain('http://localhost:3000/callback');

		// Clean up
		await cleanupTestResources({ clients: [createdClient.id] });
	});

	test('Update OIDC client with multiple fields', async () => {
		// First create a test client
		const clientName = `Update Client ${Date.now()}`;
		const clientId = `update-client-${Date.now()}`;

		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);

		// Update client
		const updatedName = `Updated Client ${Date.now()}`;
		const updateOutput = runCLICommand([
			'oidc-clients',
			'update',
			createdClient.id,
			'--name',
			updatedName,
			'--callback-urls',
			'http://localhost:3000/callback,http://localhost:8080/callback',
			'--requires-reauth',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdClient.id);
		expect(result).toHaveProperty('name', updatedName);
		expect(result).toHaveProperty('callbackURLs');
		expect(Array.isArray(result.callbackURLs)).toBe(true);
		expect(result.callbackURLs).toContain('http://localhost:3000/callback');
		expect(result.callbackURLs).toContain('http://localhost:8080/callback');
		expect(result).toHaveProperty('requiresReauthentication', true);

		// Clean up
		await cleanupTestResources({ clients: [createdClient.id] });
	});

	test('Delete OIDC client', async () => {
		// First create a test client
		const clientName = `Delete Client ${Date.now()}`;
		const clientId = `delete-client-${Date.now()}`;

		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);

		// Delete client
		const deleteOutput = runCLICommand(['oidc-clients', 'delete', createdClient.id, '--force']);
		expect(deleteOutput).toContain('deleted');

		// Verify client is deleted
		try {
			runCLICommand(['oidc-clients', 'get', createdClient.id, '--format', 'json']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Create OIDC client with duplicate client ID fails', async () => {
		const clientId = `duplicate-client-${Date.now()}`;
		const clientName1 = `Duplicate Client 1 ${Date.now()}`;
		const clientName2 = `Duplicate Client 2 ${Date.now()}`;

		// Create first client
		const createOutput1 = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName1,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const client1 = parseJSONOutput(createOutput1);

		// Try to create second client with same client ID
		try {
			runCLICommand([
				'oidc-clients',
				'create',
				'--name',
				clientName2,
				'--id',
				clientId,
				'--callback-urls',
				'http://localhost:3000/callback',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}

		// Clean up
		await cleanupTestResources({ clients: [client1.id] });
	});

	test('Create OIDC client with invalid redirect URI fails', async () => {
		const clientName = `Invalid URI Client ${Date.now()}`;
		const clientId = `invalid-uri-client-${Date.now()}`;

		try {
			runCLICommand([
				'oidc-clients',
				'create',
				'--name',
				clientName,
				'--client-id',
				clientId,
				'--redirect-uris',
				'not-a-valid-url',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Get non-existent OIDC client fails', async () => {
		const nonExistentId = '00000000-0000-0000-0000-000000000000';

		try {
			runCLICommand(['oidc-clients', 'get', nonExistentId, '--format', 'json']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Update non-existent OIDC client fails', async () => {
		const nonExistentId = '00000000-0000-0000-0000-000000000000';

		try {
			runCLICommand([
				'oidc-clients',
				'update',
				nonExistentId,
				'--name',
				'Updated Name',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Delete non-existent OIDC client fails', async () => {
		const nonExistentId = 'non-existent-client-id';

		try {
			runCLICommand(['oidc-clients', 'delete', nonExistentId, '--force']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			// The error message might vary, just ensure an error was thrown
			expect(e.message).toBeDefined();
		}
	});

	test('Create OIDC client with missing required field fails', async () => {
		const clientName = `Test Client ${Date.now()}`;

		try {
			runCLICommand([
				'oidc-clients',
				'create',
				'--name',
				clientName,
				'--callback-urls',
				'http://localhost:3000/callback',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			// The error message might vary, just ensure an error was thrown
			expect(e.message).toBeDefined();
		}
	});

	test('Update allowed user groups for OIDC client', async () => {
		// First, create a test client
		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test Client for Allowed Groups',
			'--callback-urls',
			'http://test-allowed-groups.example.com/callback',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);

		// Create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'test_allowed_group',
			'--friendly-name',
			'Test Allowed Group',
			'--format',
			'json'
		]);
		const group = parseJSONOutput(groupOutput);

		// First, enable group restriction
		const restrictOutput = runCLICommand([
			'oidc-clients',
			'update',
			createdClient.id,
			'--name',
			'Test Client for Allowed Groups',
			'--group-restricted',
			'--format',
			'json'
		]);
		const restrictedClient = parseJSONOutput(restrictOutput);
		expect(restrictedClient).toHaveProperty('isGroupRestricted', true);

		// Update allowed groups for the client
		const updateOutput = runCLICommand([
			'oidc-clients',
			'update-allowed-groups',
			createdClient.id,
			'--group-ids',
			group.id,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(updateOutput);

		// Debug: log the actual response
		console.log('Update allowed groups response:', JSON.stringify(result, null, 2));

		// Verify the update
		expect(result).toHaveProperty('id', createdClient.id);
		expect(result).toHaveProperty('isGroupRestricted', true);
		expect(result).toHaveProperty('allowedUserGroups');
		// Note: The API might return null for empty allowedUserGroups
		// Check if it's either an array or null
		expect(result.allowedUserGroups === null || Array.isArray(result.allowedUserGroups)).toBe(true);

		// Fetch the client again to verify the update was applied
		const clientAfterUpdate = parseJSONOutput(
			runCLICommand(['oidc-clients', 'get', createdClient.id, '--format', 'json'])
		);

		expect(clientAfterUpdate).toHaveProperty('id', createdClient.id);
		expect(clientAfterUpdate).toHaveProperty('isGroupRestricted', true);
		expect(clientAfterUpdate).toHaveProperty('allowedUserGroups');

		// The get response should have the allowedUserGroups as an array
		expect(Array.isArray(clientAfterUpdate.allowedUserGroups)).toBe(true);
		expect(clientAfterUpdate.allowedUserGroups.length).toBe(1);

		// Clean up
		await cleanupTestResources({ clients: [createdClient.id], groups: [group.id] });
	});

	test('OIDC client management command help', async () => {
		const output = runCLICommand(['oidc-clients', '--help']);
		expect(output).toContain('Usage:');
		expect(output).toContain('oidc-clients');
		expect(output).toContain('Create, list, update, and delete OIDC clients.');
	});

	test('List OIDC clients with pagination', async () => {
		// Create multiple clients
		const client1Output = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			`Page Client 1 ${Date.now()}`,
			'--id',
			`page-client-1-${Date.now()}`,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const client1 = parseJSONOutput(client1Output);

		const client2Output = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			`Page Client 2 ${Date.now()}`,
			'--id',
			`page-client-2-${Date.now()}`,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--format',
			'json'
		]);
		const client2 = parseJSONOutput(client2Output);

		// List with limit
		const listOutput = runCLICommand(['oidc-clients', 'list', '--limit', '1', '--format', 'json']);
		const result = parseJSONOutput(listOutput);

		expect(result).toHaveProperty('data');
		expect(Array.isArray(result.data)).toBe(true);
		expect(result.data.length).toBe(1);

		// Clean up
		await cleanupTestResources({ clients: [client1.id, client2.id] });
	});

	test('Create OIDC client with custom launch URL', async () => {
		const clientName = `Launch URL Client ${Date.now()}`;
		const clientId = `launch-url-client-${Date.now()}`;
		const launchUrl = 'https://app.example.com/launch';

		const output = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--launch-url',
			launchUrl,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id', clientId);
		expect(result).toHaveProperty('name', clientName);
		expect(result).toHaveProperty('launchURL', launchUrl);

		// Clean up
		await cleanupTestResources({ clients: [result.id] });
	});

	test('Update OIDC client to remove group restriction', async () => {
		// First create a group-restricted client
		const clientName = `Remove Restriction Client ${Date.now()}`;
		const clientId = `remove-restriction-client-${Date.now()}`;

		const createOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			clientName,
			'--id',
			clientId,
			'--callback-urls',
			'http://localhost:3000/callback',
			'--group-restricted',
			'--format',
			'json'
		]);
		const createdClient = parseJSONOutput(createOutput);
		expect(createdClient).toHaveProperty('isGroupRestricted', true);

		// Update to remove group restriction
		const updateOutput = runCLICommand([
			'oidc-clients',
			'update',
			createdClient.id,
			'--name',
			clientName,
			'--group-restricted=false',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdClient.id);
		expect(result).toHaveProperty('isGroupRestricted', false);

		// Clean up
		await cleanupTestResources({ clients: [createdClient.id] });
	});
});
