import { execFileSync, ExecFileSyncOptions } from 'child_process';
import { pathFromRoot, tmpDir } from './fs.util';

// Constants
export const containerName = 'pocket-id';
export const setupDir = pathFromRoot('setup');
export const dockerCommandMaxBuffer = 100 * 1024 * 1024;

// Types
export type DatabaseMode = 'sqlite' | 'postgres' | 's3';

// State
let mode: DatabaseMode = 'sqlite';
let apiKey: string = '';

// Helper functions for Docker operations
export function runDockerCommand(args: string[], options?: ExecFileSyncOptions): string {
	return execFileSync('docker', args, {
		cwd: setupDir,
		stdio: 'pipe',
		maxBuffer: dockerCommandMaxBuffer,
		...options
	})
		.toString()
		.trim();
}

export function runDockerComposeCommand(args: string[]): string {
	return runDockerComposeCommandRaw(args).toString().trim();
}

export function runDockerComposeCommandRaw(args: string[], options?: ExecFileSyncOptions): Buffer {
	return execFileSync('docker', dockerComposeArgs(args), {
		cwd: setupDir,
		stdio: 'pipe',
		maxBuffer: dockerCommandMaxBuffer,
		...options
	}) as Buffer;
}

export function dockerComposeArgs(args: string[]): string[] {
	let dockerComposeFile = 'docker-compose.yml';
	switch (mode) {
		case 'postgres':
			dockerComposeFile = 'docker-compose-postgres.yml';
			break;
		case 's3':
			dockerComposeFile = 'docker-compose-s3.yml';
			break;
	}
	return ['compose', '-f', dockerComposeFile, ...args];
}

// Helper to run CLI commands
export function runCLICommand(args: string[], options?: ExecFileSyncOptions): string {
	const fullArgs = dockerComposeArgs(['run', '--rm', containerName, '/app/pocket-id', ...args]);

	// Add endpoint flag to connect to the Pocket-ID service in Docker network
	fullArgs.push('--endpoint', 'http://pocket-id:1411');

	if (apiKey) {
		fullArgs.push('--api-key', apiKey);
	}

	return execFileSync('docker', fullArgs, {
		cwd: setupDir,
		stdio: 'pipe',
		maxBuffer: dockerCommandMaxBuffer,
		...options
	})
		.toString()
		.trim();
}

// Helper to parse JSON output
export function parseJSONOutput(output: string): any {
	try {
		return JSON.parse(output);
	} catch (e) {
		// Try to find JSON in the output
		const jsonMatch = output.match(/\{[\s\S]*\}/);
		if (jsonMatch) {
			return JSON.parse(jsonMatch[0]);
		}
		throw new Error(`Failed to parse JSON from output: ${output.substring(0, 100)}...`);
	}
}

// Setup functions
export async function setupTestEnvironment(): Promise<void> {
	// Determine mode
	const dockerComposeLs = runDockerCommand(['compose', 'ls', '--format', 'json']);
	if (dockerComposeLs.includes('postgres')) {
		mode = 'postgres';
	} else if (dockerComposeLs.includes('s3')) {
		mode = 's3';
	}
	console.log(`Running CLI tests in ${mode.toUpperCase()} mode`);
}

export async function createAdminUserAndApiKey(): Promise<void> {
	// Create admin user first (no authentication needed for setup command)
	console.log('Creating admin user...');
	const adminOutput = runCLICommand(
		[
			'setup',
			'create-admin',
			'--username',
			'testadmin',
			'--first-name',
			'Test',
			'--last-name',
			'Admin',
			'--email',
			'testadmin@example.com',
			'--format',
			'json'
		],
		{ stdio: 'pipe' }
	);

	console.log('Admin creation output (first 200 chars):', adminOutput.substring(0, 200));

	// Parse admin creation response to get user ID
	let adminUser;
	try {
		adminUser = parseJSONOutput(adminOutput);
	} catch (e) {
		console.error('Failed to parse admin creation output:', e);
		console.error('Raw output:', adminOutput);
		throw e;
	}

	console.log('Admin user created:', adminUser.id);

	// The setup create-admin command returns plain text output, not JSON
	// It tells us to use api-key generate to create an API key
	// Use api-key generate command (direct database access, no authentication needed)
	console.log('Creating API key using api-key generate...');
	const apiKeyOutput = runCLICommand([
		'api-key',
		'generate',
		'testadmin',
		'--name',
		'Test CLI API Key',
		'--show-token',
		'--format',
		'json'
	]);

	console.log('API key generation output (first 200 chars):', apiKeyOutput.substring(0, 200));

	// Parse API key response to get the token
	let apiKeyResponse;
	try {
		apiKeyResponse = parseJSONOutput(apiKeyOutput);
	} catch (e) {
		console.error('Failed to parse API key generation output:', e);
		console.error('Raw output:', apiKeyOutput);
		throw e;
	}

	// Set the API key for subsequent commands
	if (apiKeyResponse.token) {
		apiKey = apiKeyResponse.token;
		console.log('API key set successfully');
	} else {
		console.error('No token in API key response:', apiKeyResponse);
		throw new Error('Failed to get API token from api-key generate command');
	}
}

// Getter functions for state
export function getMode(): DatabaseMode {
	return mode;
}

export function getApiKey(): string {
	return apiKey;
}

export function setApiKey(newApiKey: string): void {
	apiKey = newApiKey;
}

// Utility functions
export function generateUniqueId(): string {
	return Math.random().toString(36).substring(2, 10);
}

export function generateTestUsername(prefix: string = 'testuser'): string {
	return `${prefix}_${generateUniqueId()}`;
}

export function generateTestEmail(prefix: string = 'test'): string {
	return `${prefix}_${generateUniqueId()}@example.com`;
}

// Cleanup utility
export async function cleanupTestResources(resourceIds: {
	users?: string[];
	groups?: string[];
	clients?: string[];
	apiKeys?: string[];
	scimProviders?: string[];
}): Promise<void> {
	const { users = [], groups = [], clients = [], apiKeys = [], scimProviders = [] } = resourceIds;

	// Clean up in reverse order to avoid foreign key constraints
	for (const scimId of scimProviders) {
		try {
			runCLICommand(['scim', 'delete', scimId, '--force']);
		} catch (e) {
			console.log(`Failed to delete SCIM provider ${scimId}:`, e.message);
		}
	}

	for (const clientId of clients) {
		try {
			runCLICommand(['oidc-clients', 'delete', clientId, '--force']);
		} catch (e) {
			console.log(`Failed to delete OIDC client ${clientId}:`, e.message);
		}
	}

	for (const groupId of groups) {
		try {
			runCLICommand(['user-groups', 'delete', groupId, '--force']);
		} catch (e) {
			console.log(`Failed to delete user group ${groupId}:`, e.message);
		}
	}

	for (const userId of users) {
		try {
			runCLICommand(['users', 'delete', userId, '--force']);
		} catch (e) {
			console.log(`Failed to delete user ${userId}:`, e.message);
		}
	}

	for (const apiKeyId of apiKeys) {
		try {
			runCLICommand(['api-key', 'delete', apiKeyId, '--force']);
		} catch (e) {
			console.log(`Failed to delete API key ${apiKeyId}:`, e.message);
		}
	}
}
