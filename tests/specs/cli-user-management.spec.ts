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

test.describe('User Management CLI', () => {
	test('Create user with required fields', async () => {
		const username = generateTestUsername();
		const email = generateTestEmail();

		const output = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Test',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('username', username);
		expect(result).toHaveProperty('firstName', 'Test');
		expect(result).toHaveProperty('lastName', 'User');
		expect(result).toHaveProperty('email', email);
		expect(result).toHaveProperty('displayName', 'Test User');
		expect(result).toHaveProperty('isAdmin', false);

		// Clean up
		await cleanupTestResources({ users: [result.id] });
	});

	test('Create user with all optional fields', async () => {
		const username = generateTestUsername('fulluser');
		const email = generateTestEmail('fulluser');

		const output = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Full',
			'--last-name',
			'User',
			'--display-name',
			'Full Test User',
			'--email',
			email,
			'--admin',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('username', username);
		expect(result).toHaveProperty('firstName', 'Full');
		expect(result).toHaveProperty('lastName', 'User');
		expect(result).toHaveProperty('displayName', 'Full Test User');
		expect(result).toHaveProperty('email', email);
		expect(result).toHaveProperty('isAdmin', true);

		// Clean up
		await cleanupTestResources({ users: [result.id] });
	});

	test('List users', async () => {
		// First create a test user
		const username = generateTestUsername('listuser');
		const email = generateTestEmail('listuser');

		const createOutput = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'List',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(createOutput);

		// List users
		const listOutput = runCLICommand(['users', 'list', '--format', 'json']);
		const result = parseJSONOutput(listOutput);

		expect(result).toHaveProperty('data');
		expect(Array.isArray(result.data)).toBe(true);
		expect(result.data.length).toBeGreaterThan(0);

		// Should contain our created user
		const foundUser = result.data.find((user: any) => user.id === createdUser.id);
		expect(foundUser).toBeDefined();
		expect(foundUser).toHaveProperty('username', username);

		// Clean up
		await cleanupTestResources({ users: [createdUser.id] });
	});

	test('Get user by ID', async () => {
		// First create a test user
		const username = generateTestUsername('getuser');
		const email = generateTestEmail('getuser');

		const createOutput = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Get',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(createOutput);

		// Get user by ID
		const getOutput = runCLICommand(['users', 'get', createdUser.id, '--format', 'json']);
		const result = parseJSONOutput(getOutput);

		expect(result).toHaveProperty('id', createdUser.id);
		expect(result).toHaveProperty('username', username);
		expect(result).toHaveProperty('firstName', 'Get');
		expect(result).toHaveProperty('lastName', 'User');
		expect(result).toHaveProperty('email', email);

		// Clean up
		await cleanupTestResources({ users: [createdUser.id] });
	});

	test('Update user', async () => {
		// First create a test user
		const username = generateTestUsername('updateuser');
		const email = generateTestEmail('updateuser');

		const createOutput = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Update',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(createOutput);

		// Update user
		const newEmail = generateTestEmail('updated');
		const updateOutput = runCLICommand([
			'users',
			'update',
			createdUser.id,
			'--username',
			createdUser.username,
			'--first-name',
			'Updated',
			'--last-name',
			'Name',
			'--display-name',
			'Updated Display Name',
			'--email',
			newEmail,
			'--admin=true',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdUser.id);
		expect(result).toHaveProperty('firstName', 'Updated');
		expect(result).toHaveProperty('lastName', 'Name');
		expect(result).toHaveProperty('displayName', 'Updated Display Name');
		expect(result).toHaveProperty('email', newEmail);
		expect(result).toHaveProperty('isAdmin', true);

		// Clean up
		await cleanupTestResources({ users: [createdUser.id] });
	});

	test('Delete user', async () => {
		// First create a test user
		const username = generateTestUsername('deleteuser');
		const email = generateTestEmail('deleteuser');

		const createOutput = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Delete',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(createOutput);

		// Delete user
		const deleteOutput = runCLICommand(['users', 'delete', createdUser.id, '--force']);
		expect(deleteOutput).toContain('deleted');

		// Verify user is deleted
		try {
			runCLICommand(['users', 'get', createdUser.id, '--format', 'json']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Create user with duplicate username fails', async () => {
		const username = generateTestUsername('duplicate');
		const email1 = generateTestEmail('duplicate1');
		const email2 = generateTestEmail('duplicate2');

		// Create first user
		const createOutput1 = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'First',
			'--last-name',
			'User',
			'--email',
			email1,
			'--format',
			'json'
		]);
		const user1 = parseJSONOutput(createOutput1);

		// Try to create second user with same username
		try {
			runCLICommand([
				'users',
				'create',
				'--username',
				username,
				'--first-name',
				'Second',
				'--last-name',
				'User',
				'--display-name',
				'Second User',
				'--email',
				email2,
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}

		// Clean up
		await cleanupTestResources({ users: [user1.id] });
	});

	test('Create user with duplicate email fails', async () => {
		const username1 = generateTestUsername('duplicate1');
		const username2 = generateTestUsername('duplicate2');
		const email = generateTestEmail('duplicate');

		// Create first user
		const createOutput1 = runCLICommand([
			'users',
			'create',
			'--username',
			username1,
			'--first-name',
			'First',
			'--last-name',
			'User',
			'--email',
			email,
			'--format',
			'json'
		]);
		const user1 = parseJSONOutput(createOutput1);

		// Try to create second user with same email
		try {
			runCLICommand([
				'users',
				'create',
				'--username',
				username2,
				'--first-name',
				'Second',
				'--last-name',
				'User',
				'--display-name',
				'Second User',
				'--email',
				email,
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}

		// Clean up
		await cleanupTestResources({ users: [user1.id] });
	});

	test('User management command help', async () => {
		const output = runCLICommand(['users', '--help']);
		expect(output).toContain('Usage:');
		expect(output).toContain('users');
		expect(output).toContain('Create, list, update, and delete users.');
	});

	test('Create user with invalid email fails', async () => {
		const username = generateTestUsername('invalidemail');

		try {
			runCLICommand([
				'users',
				'create',
				'--username',
				username,
				'--first-name',
				'Invalid',
				'--last-name',
				'Email',
				'--email',
				'invalid-email',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Get non-existent user fails', async () => {
		const nonExistentId = '00000000-0000-0000-0000-000000000000';

		try {
			runCLICommand(['users', 'get', nonExistentId, '--format', 'json']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Update non-existent user fails', async () => {
		const nonExistentId = '00000000-0000-0000-0000-000000000000';

		try {
			runCLICommand([
				'users',
				'update',
				nonExistentId,
				'--first-name',
				'Updated',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Delete non-existent user fails', async () => {
		const nonExistentId = '00000000-0000-0000-0000-000000000000';

		try {
			runCLICommand(['users', 'delete', nonExistentId, '--force']);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Create user with missing required field fails', async () => {
		const username = generateTestUsername('missingfield');

		try {
			// Missing email
			runCLICommand([
				'users',
				'create',
				'--username',
				username,
				'--first-name',
				'Missing',
				'--last-name',
				'Field',
				'--format',
				'json'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('List users with pagination', async () => {
		// Create multiple users
		const user1Output = runCLICommand([
			'users',
			'create',
			'--username',
			generateTestUsername('page1'),
			'--first-name',
			'Page',
			'--last-name',
			'One',
			'--email',
			generateTestEmail('page1'),
			'--format',
			'json'
		]);
		const user1 = parseJSONOutput(user1Output);

		const user2Output = runCLICommand([
			'users',
			'create',
			'--username',
			generateTestUsername('page2'),
			'--first-name',
			'Page',
			'--last-name',
			'Two',
			'--email',
			generateTestEmail('page2'),
			'--format',
			'json'
		]);
		const user2 = parseJSONOutput(user2Output);

		// List with limit
		const listOutput = runCLICommand(['users', 'list', '--limit', '1', '--format', 'json']);
		const result = parseJSONOutput(listOutput);

		expect(result).toHaveProperty('data');
		expect(Array.isArray(result.data)).toBe(true);
		expect(result.data.length).toBe(1);

		// Clean up
		await cleanupTestResources({ users: [user1.id, user2.id] });
	});

	test('Create user with custom display name', async () => {
		const username = generateTestUsername('customdisplay');
		const email = generateTestEmail('customdisplay');

		const output = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Custom',
			'--last-name',
			'Display',
			'--display-name',
			'Custom Display Name Here',
			'--email',
			email,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('username', username);
		expect(result).toHaveProperty('displayName', 'Custom Display Name Here');
		expect(result).toHaveProperty('firstName', 'Custom');
		expect(result).toHaveProperty('lastName', 'Display');

		// Clean up
		await cleanupTestResources({ users: [result.id] });
	});

	test('Create user without admin flag defaults to non-admin', async () => {
		const username = generateTestUsername('nonadmin');
		const email = generateTestEmail('nonadmin');

		const output = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Non',
			'--last-name',
			'Admin',
			'--email',
			email,
			'--format',
			'json'
		]);
		const result = parseJSONOutput(output);

		expect(result).toHaveProperty('id');
		expect(result).toHaveProperty('isAdmin', false);

		// Clean up
		await cleanupTestResources({ users: [result.id] });
	});

	test('Update user to remove admin status', async () => {
		// First create an admin user
		const username = generateTestUsername('removeadmin');
		const email = generateTestEmail('removeadmin');

		const createOutput = runCLICommand([
			'users',
			'create',
			'--username',
			username,
			'--first-name',
			'Admin',
			'--last-name',
			'User',
			'--email',
			email,
			'--admin',
			'--format',
			'json'
		]);
		const createdUser = parseJSONOutput(createOutput);
		expect(createdUser).toHaveProperty('isAdmin', true);

		// Update to remove admin status
		const updateOutput = runCLICommand([
			'users',
			'update',
			createdUser.id,
			'--username',
			createdUser.username,
			'--first-name',
			createdUser.firstName,
			'--last-name',
			createdUser.lastName,
			'--display-name',
			createdUser.displayName,
			'--email',
			createdUser.email,
			'--admin=false',
			'--format',
			'json'
		]);
		const result = parseJSONOutput(updateOutput);

		expect(result).toHaveProperty('id', createdUser.id);
		expect(result).toHaveProperty('isAdmin', false);

		// Clean up
		await cleanupTestResources({ users: [createdUser.id] });
	});
});
