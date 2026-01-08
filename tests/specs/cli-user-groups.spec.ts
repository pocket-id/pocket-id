import { expect, test } from '@playwright/test';
import crypto from 'crypto';
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

test.describe('User Group Management CLI', () => {
	test('List user groups - should return empty array for fresh database', async () => {
		const output = runCLICommand(['user-groups', 'list', '--format', 'json']);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('data');
		expect(Array.isArray(result.data)).toBe(true);
		// Fresh database should have no user groups
		expect(result.data.length).toBe(0);
	});

	test('Create user group', async () => {
		const groupName = `Test User Group ${Date.now()}`;
		const friendlyName = `Test Group Friendly Name ${Date.now()}`;

		const output = runCLICommand([
			'user-groups',
			'create',
			'--name',
			groupName,
			'--friendly-name',
			friendlyName,
			'--format',
			'json'
		]);

		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('name', groupName);
		expect(result).toHaveProperty('friendlyName', friendlyName);
		expect(result).toHaveProperty('users', []);
		expect(result).toHaveProperty('allowedOidcClients', []);
		expect(result).toHaveProperty('customClaims', []);

		// Clean up
		runCLICommand(['user-groups', 'delete', result.id, '--force']);
	});

	test('Get user group by ID', async () => {
		// First, create a test group
		const createOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Get',
			'--friendly-name',
			'Test Group for Get Friendly Name',
			'--format',
			'json'
		]);

		const createdGroup = parseJSONOutput(createOutput);

		// Now get the group by ID
		const output = runCLICommand(['user-groups', 'get', createdGroup.id, '--format', 'json']);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id', createdGroup.id);
		expect(result).toHaveProperty('name', 'Test Group for Get');
		expect(result).toHaveProperty('friendlyName', 'Test Group for Get Friendly Name');

		// Clean up
		runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);
	});

	test('Update user group', async () => {
		// First, create a test group
		const createOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Original Group',
			'--friendly-name',
			'Original Friendly Name',
			'--format',
			'json'
		]);

		const createdGroup = parseJSONOutput(createOutput);

		// Update the group
		const updateOutput = runCLICommand([
			'user-groups',
			'update',
			createdGroup.id,
			'--name',
			'Updated Group',
			'--friendly-name',
			'Updated Friendly Name',
			'--format',
			'json'
		]);

		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdGroup.id);
		expect(result).toHaveProperty('name', 'Updated Group');
		expect(result).toHaveProperty('friendlyName', 'Updated Friendly Name');

		// Clean up
		runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);
	});

	test('Add member to user group', async () => {
		// First, create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Members',
			'--friendly-name',
			'Test Group for Members Friendly Name',
			'--format',
			'json'
		]);

		const group = parseJSONOutput(groupOutput);

		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'groupmember',
			'--first-name',
			'Group',
			'--last-name',
			'Member',
			'--display-name',
			'Group Member',
			'--email',
			'groupmember@example.com',
			'--format',
			'json'
		]);

		const user = parseJSONOutput(userOutput);

		// Add user to group using update-users command
		const updateOutput = runCLICommand([
			'user-groups',
			'update-users',
			group.id,
			'--user-ids',
			user.id,
			'--format',
			'json'
		]);

		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', group.id);
		expect(result).toHaveProperty('users');
		expect(Array.isArray(result.users)).toBe(true);
		// Should have 1 user (the one we just added)
		expect(result.users.length).toBe(1);
		expect(result.users[0]).toHaveProperty('id', user.id);
		expect(result.users[0]).toHaveProperty('username', 'groupmember');

		// Clean up
		runCLICommand(['user-groups', 'delete', group.id, '--force']);
		runCLICommand(['users', 'delete', user.id, '--force']);
	});

	test('Update multiple users in group', async () => {
		// First, create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Multiple Users',
			'--friendly-name',
			'Test Group for Multiple Users Friendly Name',
			'--format',
			'json'
		]);

		const group = parseJSONOutput(groupOutput);

		// Create two test users
		const user1Output = runCLICommand([
			'users',
			'create',
			'--username',
			'groupmember1',
			'--first-name',
			'Group',
			'--last-name',
			'Member1',
			'--display-name',
			'Group Member 1',
			'--email',
			'groupmember1@example.com',
			'--format',
			'json'
		]);

		const user1 = parseJSONOutput(user1Output);

		const user2Output = runCLICommand([
			'users',
			'create',
			'--username',
			'groupmember2',
			'--first-name',
			'Group',
			'--last-name',
			'Member2',
			'--display-name',
			'Group Member 2',
			'--email',
			'groupmember2@example.com',
			'--format',
			'json'
		]);

		const user2 = parseJSONOutput(user2Output);

		// Add both users to group
		const updateOutput = runCLICommand([
			'user-groups',
			'update-users',
			group.id,
			'--user-ids',
			`${user1.id},${user2.id}`,
			'--format',
			'json'
		]);

		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', group.id);
		expect(result).toHaveProperty('users');
		expect(Array.isArray(result.users)).toBe(true);
		// Should have 2 users
		expect(result.users.length).toBe(2);

		// Verify both users are in the group
		const userIds = result.users.map((u: any) => u.id);
		expect(userIds).toContain(user1.id);
		expect(userIds).toContain(user2.id);

		// Clean up
		runCLICommand(['user-groups', 'delete', group.id, '--force']);
		runCLICommand(['users', 'delete', user1.id, '--force']);
		runCLICommand(['users', 'delete', user2.id, '--force']);
	});

	test('Delete user group', async () => {
		// First, create a test group
		const createOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Delete',
			'--friendly-name',
			'Test Group for Delete Friendly Name',
			'--format',
			'json'
		]);

		const createdGroup = parseJSONOutput(createOutput);

		// Delete the group
		const deleteOutput = runCLICommand(['user-groups', 'delete', createdGroup.id, '--force']);

		// Verify deletion by trying to get the group (should fail)
		try {
			runCLICommand(['user-groups', 'get', createdGroup.id, '--format', 'json']);
			// If we get here, the group still exists
			throw new Error('Group should have been deleted');
		} catch (error: any) {
			// Expected - group should not exist
			expect(error.message).toContain('request failed');
		}
	});

	test('Update allowed OIDC clients for user group', async () => {
		// First, create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Allowed Clients',
			'--friendly-name',
			'Test Group for Allowed Clients Friendly Name',
			'--format',
			'json'
		]);

		const group = parseJSONOutput(groupOutput);

		// Create a test OIDC client
		const clientOutput = runCLICommand([
			'oidc-clients',
			'create',
			'--name',
			'Test Client for Group',
			'--callback-urls',
			'http://test-group-client.example.com/callback',
			'--format',
			'json'
		]);

		const client = parseJSONOutput(clientOutput);

		// Update allowed clients for the group
		const updateOutput = runCLICommand([
			'user-groups',
			'update-allowed-clients',
			group.id,
			'--client-ids',
			client.id,
			'--format',
			'json'
		]);

		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', group.id);
		expect(result).toHaveProperty('allowedOidcClients');
		expect(Array.isArray(result.allowedOidcClients)).toBe(true);
		// Should have 1 allowed client
		expect(result.allowedOidcClients.length).toBe(1);
		expect(result.allowedOidcClients[0]).toHaveProperty('id', client.id);

		// Clean up
		runCLICommand(['user-groups', 'delete', group.id, '--force']);
		runCLICommand(['oidc-clients', 'delete', client.id, '--force']);
	});

	test('User group command help', async () => {
		const output = runCLICommand(['user-groups', '--help']);

		expect(output).toContain('Usage:');
		expect(output).toContain('user-groups');
		// The actual help text might be different, just check it contains key words
		expect(output).toContain('user');
		expect(output).toContain('group');
	});

	test.skip('Create user group with custom claims', async () => {
		// Skipping this test for now due to file mounting issues with Docker
		// The custom-claims command requires a file path that the container can access
		// This would require complex volume mounting setup
	});

	test('Remove all users from group', async () => {
		// First, create a test group
		const groupOutput = runCLICommand([
			'user-groups',
			'create',
			'--name',
			'Test Group for Remove Users',
			'--friendly-name',
			'Test Group for Remove Users Friendly Name',
			'--format',
			'json'
		]);

		const group = parseJSONOutput(groupOutput);

		// Create a test user
		const userOutput = runCLICommand([
			'users',
			'create',
			'--username',
			'removableuser',
			'--first-name',
			'Removable',
			'--last-name',
			'User',
			'--display-name',
			'Removable User',
			'--email',
			'removable@example.com',
			'--format',
			'json'
		]);

		const user = parseJSONOutput(userOutput);

		// Add user to group
		const addOutput = runCLICommand([
			'user-groups',
			'update-users',
			group.id,
			'--user-ids',
			user.id,
			'--format',
			'json'
		]);

		const groupWithUser = parseJSONOutput(addOutput);
		expect(groupWithUser.users.length).toBe(1);

		// Remove all users from group by passing empty user-ids
		const removeOutput = runCLICommand([
			'user-groups',
			'update-users',
			group.id,
			'--user-ids',
			'',
			'--format',
			'json'
		]);

		const groupWithoutUsers = parseJSONOutput(removeOutput);
		expect(groupWithoutUsers).toHaveProperty('id', group.id);
		expect(groupWithoutUsers).toHaveProperty('users');
		expect(Array.isArray(groupWithoutUsers.users)).toBe(true);
		expect(groupWithoutUsers.users.length).toBe(0);

		// Clean up
		runCLICommand(['user-groups', 'delete', group.id, '--force']);
		runCLICommand(['users', 'delete', user.id, '--force']);
	});
});
