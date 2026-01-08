import { expect, test } from '@playwright/test';
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

test.describe('Custom Claims CLI', () => {
	test('Get custom claim suggestions', async () => {
		const output = runCLICommand(['custom-claims', 'suggestions', '--format', 'json']);
		const result = parseJSONOutput(output);

		expect(Array.isArray(result)).toBe(true);
	});

	test.skip('Update custom claims for user', async () => {
		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'customclaimuser',
			'--first-name',
			'Custom',
			'--last-name',
			'Claims',
			'--display-name',
			'Custom Claims User',
			'--email',
			'customclaims@example.com',
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(userOutput);

		// Update custom claims - CLI expects array of {key, value} objects
		const customClaims = [
			{ key: 'department', value: 'Engineering' },
			{ key: 'role', value: 'Developer' },
			{ key: 'customField', value: 'Custom Value' }
		];

		const updateOutput = runCLICommand([
			'custom-claims',
			'update-user',
			createdUser.id,
			'--claims',
			JSON.stringify(customClaims),
			'--format',
			'json'
		]);
		const updateResult = parseJSONOutput(updateOutput);

		expect(updateResult).toHaveProperty('customClaims');
		// Convert array to object for matching
		const expectedClaims = customClaims.reduce(
			(obj, claim) => {
				obj[claim.key] = claim.value;
				return obj;
			},
			{} as Record<string, string>
		);
		expect(updateResult.customClaims).toMatchObject(expectedClaims);

		// Clean up
		runCLICommand(['users', 'delete', createdUser.id, '--force']);
	});

	test.skip('Get custom claims for user', async () => {
		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'getclaimuser',
			'--first-name',
			'Get',
			'--last-name',
			'Claims',
			'--display-name',
			'Get Claims User',
			'--email',
			'getclaims@example.com',
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(userOutput);

		// Set custom claims first - CLI expects array of {key, value} objects
		const customClaims = [
			{ key: 'department', value: 'Marketing' },
			{ key: 'title', value: 'Manager' }
		];

		runCLICommand([
			'custom-claims',
			'update-user',
			createdUser.id,
			'--claims',
			JSON.stringify(customClaims),
			'--format',
			'json'
		]);

		// Get custom claims
		const getOutput = runCLICommand([
			'custom-claims',
			'get-user',
			createdUser.id,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(getOutput);

		expect(result).toHaveProperty('customClaims');
		// Convert array to object for matching
		const expectedClaims = customClaims.reduce(
			(obj, claim) => {
				obj[claim.key] = claim.value;
				return obj;
			},
			{} as Record<string, string>
		);
		expect(result.customClaims).toMatchObject(expectedClaims);

		// Clean up
		runCLICommand(['users', 'delete', createdUser.id, '--force']);
	});

	test.skip('Clear custom claims for user', async () => {
		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'clearclaimuser',
			'--first-name',
			'Clear',
			'--last-name',
			'Claims',
			'--display-name',
			'Clear Claims User',
			'--email',
			'clearclaims@example.com',
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(userOutput);

		// Set custom claims first - CLI expects array of {key, value} objects
		const customClaims = [
			{ key: 'department', value: 'Sales' },
			{ key: 'region', value: 'North America' }
		];

		runCLICommand([
			'custom-claims',
			'update-user',
			createdUser.id,
			'--claims',
			JSON.stringify(customClaims),
			'--format',
			'json'
		]);

		// Clear custom claims
		const clearOutput = runCLICommand([
			'custom-claims',
			'clear-user',
			createdUser.id,
			'--format',
			'json'
		]);
		const clearResult = parseJSONOutput(clearOutput);

		expect(clearResult).toHaveProperty('customClaims');
		expect(clearResult.customClaims).toEqual({});

		// Verify claims are cleared
		const getOutput = runCLICommand([
			'custom-claims',
			'get-user',
			createdUser.id,
			'--format',
			'json'
		]);
		const getResult = parseJSONOutput(getOutput);

		expect(getResult).toHaveProperty('customClaims');
		expect(getResult.customClaims).toEqual({});

		// Clean up
		runCLICommand(['users', 'delete', createdUser.id, '--force']);
	});

	test('Custom claims command help', async () => {
		const output = runCLICommand(['custom-claims', '--help']);
		expect(output).toContain('Usage:');
		expect(output).toContain('custom-claims');
	});

	test('Update custom claims with invalid JSON', async () => {
		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'invalidjsonuser',
			'--first-name',
			'Invalid',
			'--last-name',
			'JSON',
			'--display-name',
			'Invalid JSON User',
			'--email',
			'invalidjson@example.com',
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(userOutput);

		// Try to update with invalid JSON
		try {
			runCLICommand([
				'custom-claims',
				'update-user',
				createdUser.id,
				'--claims',
				'invalid-json',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}

		// Clean up
		runCLICommand(['users', 'delete', createdUser.id, '--force']);
	});

	test('Get custom claims for non-existent user', async () => {
		const nonExistentUserId = '00000000-0000-0000-0000-000000000000';
		try {
			runCLICommand(['custom-claims', 'get-user', nonExistentUserId, '--format', 'json']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			// The command should fail
			expect(e).toBeDefined();
		}
	});

	test.skip('Update custom claims for group', async () => {
		// Create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'customclaimgroup',
			'--friendly-name',
			'Custom Claim Group',
			'--format',
			'json'
		]);
		const createdGroup = parseJSONOutput(groupOutput);

		// Update custom claims for group - CLI expects array of {key, value} objects
		const groupClaims = [
			{ key: 'team', value: 'Backend' },
			{ key: 'location', value: 'Remote' }
		];

		const updateOutput = runCLICommand([
			'custom-claims',
			'update-user-group',
			createdGroup.id,
			'--claims',
			JSON.stringify(groupClaims).replace(/"/g, '\\"'),
			'--format',
			'json'
		]);
		const updateResult = parseJSONOutput(updateOutput);

		expect(updateResult).toHaveProperty('customClaims');
		// Convert array to object for matching
		const expectedGroupClaims = groupClaims.reduce(
			(obj, claim) => {
				obj[claim.key] = claim.value;
				return obj;
			},
			{} as Record<string, string>
		);
		expect(updateResult.customClaims).toMatchObject(expectedGroupClaims);

		// Clean up
		runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);
	});

	test.skip('Get custom claims for group', async () => {
		// Create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'getclaimgroup',
			'--friendly-name',
			'Get Claim Group',
			'--format',
			'json'
		]);
		const createdGroup = parseJSONOutput(groupOutput);

		// Set custom claims first
		const groupClaims = [
			{ key: 'budgetCode', value: 'DEV-2024' },
			{ key: 'manager', value: 'John Doe' }
		];

		runCLICommand([
			'custom-claims',
			'update-user-group',
			createdGroup.id,
			'--claims',
			JSON.stringify(groupClaims),
			'--format',
			'json'
		]);

		// Get custom claims
		const getOutput = runCLICommand([
			'custom-claims',
			'get-user-group',
			createdGroup.id,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(getOutput);

		expect(result).toHaveProperty('customClaims');
		expect(result.customClaims).toMatchObject(groupClaims);

		// Clean up
		runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);
	});

	test.skip('Clear custom claims for group', async () => {
		// Create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'clearclaimgroup',
			'--friendly-name',
			'Clear Claim Group',
			'--format',
			'json'
		]);
		const createdGroup = parseJSONOutput(groupOutput);

		// Set custom claims first
		const groupClaims = [
			{ key: 'budgetCode', value: 'DEV-2024' },
			{ key: 'manager', value: 'John Doe' }
		];

		const updateOutput = runCLICommand([
			'custom-claims',
			'update-user-group',
			createdGroup.id,
			'--claims',
			JSON.stringify(groupClaims),
			'--format',
			'json'
		]);

		// Clear custom claims
		const clearOutput = runCLICommand([
			'custom-claims',
			'clear-user-group',
			createdGroup.id,
			'--format',
			'json'
		]);
		const clearResult = parseJSONOutput(clearOutput);

		expect(clearResult).toHaveProperty('customClaims');
		expect(clearResult.customClaims).toEqual({});

		// Clean up
		runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);
	});
});
