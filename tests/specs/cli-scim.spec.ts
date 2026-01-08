import { expect, test } from '@playwright/test';
import fs from 'fs';
import os from 'os';
import path from 'path';
import {
	runCLICommand,
	parseJSONOutput,
	setupTestEnvironment,
	createAdminUserAndApiKey,
	cleanupTestResources,
	generateTestUsername,
	generateTestEmail
} from 'utils/cli-test-helper';
import { cleanupBackend } from 'utils/cleanup.util';

test.beforeAll(async () => {
	await setupTestEnvironment();

	// Clean up backend without seeding to get fresh database
	await cleanupBackend({ skipLdapSetup: true, skipSeed: true });

	await createAdminUserAndApiKey();
});

test.describe('SCIM CLI', () => {
	test('SCIM command help', async () => {
		const output = runCLICommand(['scim', '--help']);

		expect(output).toContain('Usage:');
		expect(output).toContain('scim');
		expect(output).toContain('Create, update, delete, and sync SCIM service providers.');
	});

	test.skip('Create SCIM service provider - basic test', async () => {
		// First create an OIDC client to use for SCIM
		const clientOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test SCIM Client',
			'--callback-urls',
			'http://localhost:3000/callback',
			'--logout-callback-urls',
			'http://localhost:3000',
			'--format',
			'json'
		]);

		const client = parseJSONOutput(clientOutput);
		const clientId = client.id;

		// Create SCIM provider
		const scimUrl = 'http://scim-test.example.com/scim/v2';
		const authToken = `test-token-${Date.now()}`;

		const output = runCLICommand([
			'scim',
			'create',
			'--endpoint',
			scimUrl,
			'--token',
			authToken,
			'--oidc-client-id',
			clientId,
			'--format',
			'json'
		]);

		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('endpoint', scimUrl);
		expect(result).toHaveProperty('token', authToken);
		expect(result).toHaveProperty('oidcClientId', clientId);
		expect(result).toHaveProperty('enabled', true);
		expect(result).toHaveProperty('createdAt');

		// Store for cleanup
		const providerId = result.id;

		// Clean up
		runCLICommand(['scim', 'delete', providerId, '--force']);
		runCLICommand(['oidc-clients', 'delete', clientId, '--force']);
	});

	test.skip('Update SCIM service provider', async () => {
		// First create an OIDC client
		const clientOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test SCIM Update Client',
			'--callback-urls',
			'http://localhost:3000/callback',
			'--logout-callback-urls',
			'http://localhost:3000',
			'--format',
			'json'
		]);

		const client = parseJSONOutput(clientOutput);
		const clientId = client.id;

		// Create SCIM provider
		const createOutput = runCLICommand([
			'scim',
			'create',
			'--endpoint',
			'http://original.example.com/scim/v2',
			'--token',
			'original-token',
			'--oidc-client-id',
			clientId,
			'--format',
			'json'
		]);

		const createdProvider = parseJSONOutput(createOutput);

		// Update the provider
		const updateOutput = runCLICommand([
			'scim',
			'update',
			createdProvider.id,
			'--endpoint',
			'http://updated.example.com/scim/v2',
			'--token',
			'updated-token',
			'--format',
			'json'
		]);

		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdProvider.id);
		expect(result).toHaveProperty('endpoint', 'http://updated.example.com/scim/v2');
		expect(result).toHaveProperty('token', 'updated-token');
		expect(result).toHaveProperty('oidcClientId', clientId);

		// Clean up
		runCLICommand(['scim', 'delete', createdProvider.id, '--force']);
		runCLICommand(['oidc-clients', 'delete', clientId, '--force']);
	});

	test.skip('Delete SCIM service provider with force flag', async () => {
		// First create an OIDC client
		const clientOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test SCIM Delete Client',
			'--callback-urls',
			'http://localhost:3000/callback',
			'--logout-callback-urls',
			'http://localhost:3000',
			'--format',
			'json'
		]);

		const client = parseJSONOutput(clientOutput);
		const clientId = client.id;

		// Create SCIM provider
		const createOutput = runCLICommand([
			'scim',
			'create',
			'--endpoint',
			'http://test-delete.example.com/scim/v2',
			'--token',
			'test-token-delete',
			'--oidc-client-id',
			clientId,
			'--format',
			'json'
		]);

		const createdProvider = parseJSONOutput(createOutput);

		// Delete the provider with force flag
		const deleteOutput = runCLICommand(['scim', 'delete', createdProvider.id, '--force']);

		// Should return success message
		expect(deleteOutput).toContain('deleted successfully');

		// Clean up OIDC client
		runCLICommand(['oidc-clients', 'delete', clientId, '--force']);
	});

	test('SCIM subcommand help', async () => {
		// Test help for each subcommand
		const createHelp = runCLICommand(['scim', 'create', '--help']);
		expect(createHelp).toContain('Create a new SCIM service provider');
		expect(createHelp).toContain('--endpoint');
		expect(createHelp).toContain('--oidc-client-id');

		const updateHelp = runCLICommand(['scim', 'update', '--help']);
		expect(updateHelp).toContain('Update an existing SCIM service provider');

		const deleteHelp = runCLICommand(['scim', 'delete', '--help']);
		expect(deleteHelp).toContain('Delete a SCIM service provider by ID');

		const syncHelp = runCLICommand(['scim', 'sync', '--help']);
		expect(syncHelp).toContain('Trigger synchronization for a SCIM service provider');
	});

	test.skip('Create SCIM provider validation - missing required fields', async () => {
		// First create an OIDC client
		const clientOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test SCIM Validation Client',
			'--callback-urls',
			'http://localhost:3000/callback',
			'--logout-callback-urls',
			'http://localhost:3000',
			'--format',
			'json'
		]);

		const client = parseJSONOutput(clientOutput);
		const clientId = client.id;

		// Try to create SCIM provider without endpoint (should fail)
		try {
			runCLICommand([
				'scim',
				'create',
				'--token',
				'test-token',
				'--oidc-client-id',
				clientId,
				'--format',
				'json'
			]);
			throw new Error('Should have failed due to missing endpoint');
		} catch (error: any) {
			// Expected - missing required endpoint
			expect(error.message).toContain('failed');
		}

		// Try to create SCIM provider without oidc-client-id (should fail)
		try {
			runCLICommand([
				'scim',
				'create',
				'--endpoint',
				'http://test.example.com/scim/v2',
				'--token',
				'test-token',
				'--format',
				'json'
			]);
			throw new Error('Should have failed due to missing oidc-client-id');
		} catch (error: any) {
			// Expected - missing required oidc-client-id
			expect(error.message).toContain('failed');
		}

		// Clean up OIDC client
		runCLICommand(['oidc-clients', 'delete', clientId, '--force']);
	});
});
