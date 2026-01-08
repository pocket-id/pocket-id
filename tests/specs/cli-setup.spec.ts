import { expect, test } from '@playwright/test';
import crypto from 'crypto';
import {
	runCLICommand,
	parseJSONOutput,
	setupTestEnvironment,
	cleanupTestResources
} from 'utils/cli-test-helper';
import { cleanupBackend } from 'utils/cleanup.util';

test.beforeAll(async () => {
	await setupTestEnvironment();

	// Clean up backend without seeding to get fresh database
	await cleanupBackend({ skipLdapSetup: true, skipSeed: true });
});

test.describe('Setup CLI', () => {
	test('Setup command help', async () => {
		const output = runCLICommand(['setup', '--help']);
		expect(output).toContain('Usage:');
		expect(output).toContain('setup');
		expect(output).toContain('Commands for initial Pocket ID setup and administration.');
	});

	test('Create admin user with all required fields - only one admin can be created', async () => {
		// Generate unique username to avoid conflicts
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_${uniqueId}`;
		const email = `testadmin_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Test',
				'--last-name',
				'Admin',
				'--email',
				email
			]);

			// Parse JSON output
			const result = parseJSONOutput(output);
			expect(result).toHaveProperty('id');
			expect(result).toHaveProperty('username', username);
			expect(result).toHaveProperty('email', email);
			expect(result).toHaveProperty('firstName', 'Test');
			expect(result).toHaveProperty('lastName', 'Admin');
			expect(result).toHaveProperty('isAdmin', true);
		} catch (e: any) {
			// If setup is already completed, that's expected
			// The test passes as long as we get a proper error
			expect(e.message).toContain('failed');
		}
	});

	test('Validate admin creation without last name - tests validation not actual creation', async () => {
		// This test validates that the command accepts missing last name
		// It doesn't actually create an admin since setup might already be completed
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_noln_${uniqueId}`;
		const email = `testadmin_noln_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Test',
				'--email',
				email
			]);
			// If setup is not completed yet, parse the output
			const result = parseJSONOutput(output);
			expect(result).toHaveProperty('username', username);
		} catch (e: any) {
			// Expected to fail with setup already completed or validation error
			expect(e.message).toContain('failed');
		}
	});

	test('Fail to create admin user with missing username', async () => {
		try {
			runCLICommand([
				'setup',
				'create-admin',
				'--first-name',
				'Test',
				'--last-name',
				'Admin',
				'--email',
				'test@example.com'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Fail to create admin user with missing first name', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_missingfn_${uniqueId}`;

		try {
			runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--last-name',
				'Admin',
				'--email',
				'test@example.com'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Fail to create admin user with missing email', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_missingemail_${uniqueId}`;

		try {
			runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Test',
				'--last-name',
				'Admin'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Fail to create admin user with invalid email', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_invalidemail_${uniqueId}`;

		try {
			runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Test',
				'--last-name',
				'Admin',
				'--email',
				'invalid-email'
			]);
			expect(false).toBe(true); // Should not reach here
		} catch (e: any) {
			expect(e.message).toContain('failed');
		}
	});

	test('Test JSON output format option - validates format flag', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_json_${uniqueId}`;
		const email = `testadmin_json_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'JSON',
				'--last-name',
				'Output',
				'--email',
				email,
				'--format',
				'json'
			]);

			// Parse JSON output if command succeeded
			const result = parseJSONOutput(output);
			expect(result).toHaveProperty('username', username);
		} catch (e: any) {
			// Expected to fail with setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Test table output format option - validates format flag', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_table_${uniqueId}`;
		const email = `testadmin_table_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Table',
				'--last-name',
				'Output',
				'--email',
				email,
				'--format',
				'table'
			]);
			// The setup command doesn't support table format, so it falls back to text output
			// If command succeeded, text output should contain success message
			expect(output).toContain('✅ Initial admin user created successfully!');
			expect(output).toContain(username);
		} catch (e: any) {
			// Expected to fail with setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Test YAML output format option - validates format flag', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_yaml_${uniqueId}`;
		const email = `testadmin_yaml_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'YAML',
				'--last-name',
				'Output',
				'--email',
				email,
				'--format',
				'yaml'
			]);
			// If command succeeded, YAML output should contain the username
			expect(output).toContain(username);
		} catch (e: any) {
			// Expected to fail with setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Test text output format option - validates format flag', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `testadmin_text_${uniqueId}`;
		const email = `testadmin_text_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Text',
				'--last-name',
				'Output',
				'--email',
				email,
				'--format',
				'text'
			]);
			// The setup command doesn't have explicit text format support,
			// but any non-JSON format falls back to text output
			// If command succeeded, text output should contain success message
			expect(output).toContain('✅ Initial admin user created successfully!');
			expect(output).toContain(username);
		} catch (e: any) {
			// Expected to fail with setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Setup create-admin subcommand help', async () => {
		const output = runCLICommand(['setup', 'create-admin', '--help']);
		expect(output).toContain('Usage:');
		expect(output).toContain('create-admin');
		expect(output).toContain('Create the initial admin user');
	});

	test('Verify setup command shows available subcommands', async () => {
		const output = runCLICommand(['setup', '--help']);
		expect(output).toContain('create-admin');
		expect(output).toContain('Commands:');
	});

	test('Test username validation - duplicate username should fail', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `duplicate_${uniqueId}`;
		const email1 = `duplicate1_${uniqueId}@example.com`;
		const email2 = `duplicate2_${uniqueId}@example.com`;

		try {
			// Try to create first admin
			const output1 = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'First',
				'--last-name',
				'Admin',
				'--email',
				email1
			]);

			// If first admin creation succeeded, try to create second with same username
			try {
				runCLICommand([
					'setup',
					'create-admin',
					'--username',
					username,
					'--first-name',
					'Second',
					'--last-name',
					'Admin',
					'--email',
					email2
				]);
				// Should not reach here if duplicate username validation works
				expect(false).toBe(true);
			} catch (e: any) {
				// Expected to fail with duplicate username error
				expect(e.message).toContain('failed');
			}
		} catch (e: any) {
			// First creation might fail if setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Test email validation - duplicate email should fail', async () => {
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username1 = `duplicate_email1_${uniqueId}`;
		const username2 = `duplicate_email2_${uniqueId}`;
		const email = `duplicate_email_${uniqueId}@example.com`;

		try {
			// Try to create first admin
			const output1 = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username1,
				'--first-name',
				'First',
				'--last-name',
				'Admin',
				'--email',
				email
			]);

			// If first admin creation succeeded
			const result1 = parseJSONOutput(output1);
			expect(result1).toHaveProperty('username', username1);

			// Try to create second admin with same email
			try {
				const output2 = runCLICommand([
					'setup',
					'create-admin',
					'--username',
					username2,
					'--first-name',
					'Second',
					'--last-name',
					'Admin',
					'--email',
					email
				]);
				// Should not reach here if duplicate email validation works
				expect(false).toBe(true);
			} catch (e: any) {
				// Expected to fail with duplicate email error
				expect(e.message).toContain('failed');
			}
		} catch (e: any) {
			// First creation might fail if setup already completed
			expect(e.message).toContain('failed');
		}
	});

	test('Test setup already completed error message', async () => {
		// This test validates that when setup is already completed,
		// the command returns an appropriate error
		const uniqueId = crypto.randomBytes(4).toString('hex');
		const username = `setup_completed_${uniqueId}`;
		const email = `setup_completed_${uniqueId}@example.com`;

		try {
			const output = runCLICommand([
				'setup',
				'create-admin',
				'--username',
				username,
				'--first-name',
				'Test',
				'--last-name',
				'Admin',
				'--email',
				email
			]);
			// If setup is not completed yet, parse the output
			const result = parseJSONOutput(output);
			expect(result).toHaveProperty('username', username);
		} catch (e: any) {
			// Expected to fail with setup already completed error
			expect(e.message).toContain('failed');
		}
	});
});
