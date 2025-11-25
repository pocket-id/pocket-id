import { expect, test } from '@playwright/test';
import AdmZip from 'adm-zip';
import { execFileSync, ExecFileSyncOptions } from 'child_process';
import crypto from 'crypto';
import { users } from 'data';
import fs from 'fs';
import path from 'path';
import { cleanupBackend } from 'utils/cleanup.util';
import { pathFromRoot, tmpDir } from 'utils/fs.util';

const containerName = 'pocket-id';
const setupDir = pathFromRoot('setup');
const exampleExportPath = pathFromRoot('resources/export');
const dockerCommandMaxBuffer = 100 * 1024 * 1024;
let mode: 'sqlite' | 'postgres' | 's3' = 'sqlite';

test.beforeAll(() => {
	const dockerComposeLs = runDockerCommand(['compose', 'ls', '--format', 'json']);
	if (dockerComposeLs.includes('postgres')) {
		mode = 'postgres';
	} else if (dockerComposeLs.includes('s3')) {
		mode = 's3';
	}
	console.log(`Running CLI tests in ${mode.toUpperCase()} mode`);
});

test('Export', async ({ baseURL }) => {
	// Reset the backend but with LDAP setup because the example export has no LDAP data
	await cleanupBackend({ skipLdapSetup: true });

	// Fetch the profile pictures because they get generated on demand
	await Promise.all([
		fetch(`${baseURL}/api/users/${users.craig.id}/profile-picture.png`),
		fetch(`${baseURL}/api/users/${users.tim.id}/profile-picture.png`)
	]);

	// Export the data from the seeded container
	const exportPath = path.join(tmpDir, 'export.zip');
	const extractPath = path.join(tmpDir, 'export-extracted');

	runExport(exportPath);
	unzipExport(exportPath, extractPath);

	compareExports(exampleExportPath, extractPath);
});

test('Export via stdout', async ({ baseURL }) => {
	await cleanupBackend({ skipLdapSetup: true });

	await Promise.all([
		fetch(`${baseURL}/api/users/${users.craig.id}/profile-picture.png`),
		fetch(`${baseURL}/api/users/${users.tim.id}/profile-picture.png`)
	]);

	const stdoutBuffer = runExportToStdout();
	const stdoutExtractPath = path.join(tmpDir, 'export-stdout-extracted');
	unzipExportBuffer(stdoutBuffer, stdoutExtractPath);

	compareExports(exampleExportPath, stdoutExtractPath);
});

test('Import', async () => {
	// Reset the backend without seeding
	await cleanupBackend({ skipSeed: true });

	// Run the import with the example export data
	const exampleExportArchivePath = path.join(tmpDir, 'example-export.zip');
	archiveExampleExport(exampleExportArchivePath);

	try {
		runDockerComposeCommand(['stop', containerName]);
		runImport(exampleExportArchivePath);
	} finally {
		runDockerComposeCommand(['up', '-d', containerName]);
	}

	// Export again from the imported instance
	const exportPath = path.join(tmpDir, 'export.zip');
	const exportExtracted = path.join(tmpDir, 'export-extracted');
	runExport(exportPath);
	unzipExport(exportPath, exportExtracted);

	compareExports(exampleExportPath, exportExtracted);
});

test('Import via stdin', async () => {
	await cleanupBackend({ skipSeed: true });

	const exampleExportArchivePath = path.join(tmpDir, 'example-export-stdin.zip');
	const exampleExportBuffer = archiveExampleExport(exampleExportArchivePath);

	try {
		runDockerComposeCommand(['stop', containerName]);
		runImportFromStdin(exampleExportBuffer);
	} finally {
		runDockerComposeCommand(['up', '-d', containerName]);
	}

	const exportPath = path.join(tmpDir, 'export-from-stdin.zip');
	const exportExtracted = path.join(tmpDir, 'export-from-stdin-extracted');
	runExport(exportPath);
	unzipExport(exportPath, exportExtracted);

	compareExports(exampleExportPath, exportExtracted);
});

function compareExports(dir1: string, dir2: string): void {
	const hashes1 = hashAllFiles(dir1);
	const hashes2 = hashAllFiles(dir2);

	const files1 = Object.keys(hashes1).sort();
	const files2 = Object.keys(hashes2).sort();
	expect(files2).toEqual(files1);

	for (const file of files1) {
		expect(hashes2[file], `${file} hash should match`).toEqual(hashes1[file]);
	}

	// Compare database.json contents
	const expectedData = loadJSON(path.join(dir1, 'database.json'));
	const actualData = loadJSON(path.join(dir2, 'database.json'));

	// Check special fields
	validateSpecialFields(actualData);

	// Normalize and compare
	const normalizedExpected = normalizeJSON(expectedData);
	const normalizedActual = normalizeJSON(actualData);
	expect(normalizedActual).toEqual(normalizedExpected);
}

function archiveExampleExport(outputPath: string): Buffer {
	fs.rmSync(outputPath, { force: true });

	const zip = new AdmZip();
	const files = fs.readdirSync(exampleExportPath);
	for (const file of files) {
		const filePath = path.join(exampleExportPath, file);
		if (fs.statSync(filePath).isFile()) {
			zip.addLocalFile(filePath);
		} else if (fs.statSync(filePath).isDirectory()) {
			zip.addLocalFolder(filePath, file);
		}
	}

	const buffer = zip.toBuffer();
	fs.writeFileSync(outputPath, buffer);
	return buffer;
}

// Helper to load JSON files
function loadJSON(path: string) {
	return JSON.parse(fs.readFileSync(path, 'utf-8'));
}

function normalizeJSON(obj: any): any {
	if (typeof obj === 'string') {
		try {
			// Normalize JSON strings
			const parsed = JSON.parse(atob(obj));
			return JSON.stringify(normalizeJSON(parsed));
		} catch {
			return obj;
		}
	}

	if (Array.isArray(obj)) {
		// Sort arrays to make order irrelevant
		return obj
			.map(normalizeJSON)
			.sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b)));
	} else if (obj && typeof obj === 'object') {
		const ignoredKeys = ['id', 'created_at', 'expires_at', 'credentials', 'provider', 'version'];

		// Sort and normalize object keys, skipping ignored ones
		return Object.keys(obj)
			.filter((key) => !ignoredKeys.includes(key))
			.sort()
			.reduce(
				(acc, key) => {
					acc[key] = normalizeJSON(obj[key]);
					return acc;
				},
				{} as Record<string, any>
			);
	}

	return obj;
}

function validateSpecialFields(obj: any): void {
	if (Array.isArray(obj)) {
		for (const item of obj) validateSpecialFields(item);
	} else if (obj && typeof obj === 'object') {
		for (const [key, value] of Object.entries(obj)) {
			if (key === 'id') {
				expect(isUUID(value), `Expected '${value}' to be a valid UUID`).toBe(true);
			} else if (key === 'created_at' || key === 'expires_at') {
				expect(
					isValidISODate(value),
					`Expected '${key}' = ${value} to be a valid UNIX timestamp`
				).toBe(true);
			} else if (key === 'provider') {
				expect(
					['postgres', 'sqlite'].includes(value as string),
					`Expected 'provider' to be either 'postgres' or 'sqlite', got '${value}'`
				).toBe(true);
			} else if (key === 'version') {
				expect(value).toBeGreaterThanOrEqual(20251001000000);
			} else {
				validateSpecialFields(value);
			}
		}
	}
}

function isUUID(value: any): boolean {
	if (typeof value !== 'string') return false;
	const uuidRegex = /^[^-]{8}-[^-]{4}-[^-]{4}-[^-]{4}-[^-]{12}$/;
	return uuidRegex.test(value);
}

function isValidISODate(value: any): boolean {
	const isoRegex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/;

	if (!isoRegex.test(value)) return false;

	const date = new Date(value);
	return !isNaN(date.getTime());
}

function runImport(pathToFile: string) {
	const importContainerId = runDockerComposeCommand([
		'run',
		'-d',
		'-v',
		`${pathToFile}:/app/pocket-id-export.zip`,
		containerName,
		'/app/pocket-id',
		'import',
		'--yes'
	]);
	try {
		runDockerCommand(['wait', importContainerId]);
	} finally {
		runDockerCommand(['rm', '-f', importContainerId]);
	}
}

function runImportFromStdin(archive: Buffer): void {
	runDockerComposeCommandRaw(
		['run', '--rm', '-T', containerName, '/app/pocket-id', 'import', '--yes', '--path', '-'],
		{ input: archive }
	);
}

function runExport(outputFile: string): void {
	const containerId = runDockerComposeCommand([
		'run',
		'-d',
		containerName,
		'/app/pocket-id',
		'export'
	]);

	try {
		// Wait until export finishes
		runDockerCommand(['wait', containerId]);
		runDockerCommand(['cp', `${containerId}:/app/pocket-id-export.zip`, outputFile]);
	} finally {
		runDockerCommand(['rm', '-f', containerId]);
	}

	expect(fs.existsSync(outputFile)).toBe(true);
}

function runExportToStdout(): Buffer {
	const res = runDockerComposeCommandRaw([
		'run',
		'--rm',
		'-T',
		containerName,
		'/app/pocket-id',
		'export',
		'--path',
		'-'
	]);
	fs.writeFileSync('export-stdout.txt', res);
	return res;
}

function unzipExport(zipFile: string, destDir: string): void {
	fs.rmSync(destDir, { recursive: true, force: true });
	const zip = new AdmZip(zipFile);
	zip.extractAllTo(destDir, true);
}

function unzipExportBuffer(zipBuffer: Buffer, destDir: string): void {
	fs.rmSync(destDir, { recursive: true, force: true });
	const zip = new AdmZip(zipBuffer);
	zip.extractAllTo(destDir, true);
}

function hashFile(filePath: string): string {
	const buffer = fs.readFileSync(filePath);
	return crypto.createHash('sha256').update(buffer).digest('hex');
}

function getAllFiles(dir: string, root = dir): string[] {
	return fs.readdirSync(dir).flatMap((entry) => {
		if (['.DS_Store', 'database.json'].includes(entry)) return [];

		const fullPath = path.join(dir, entry);
		const stat = fs.statSync(fullPath);
		return stat.isDirectory() ? getAllFiles(fullPath, root) : [path.relative(root, fullPath)];
	});
}

function hashAllFiles(dir: string): Record<string, string> {
	const files = getAllFiles(dir);
	const hashes: Record<string, string> = {};
	for (const relativePath of files) {
		const fullPath = path.join(dir, relativePath);
		hashes[relativePath] = hashFile(fullPath);
	}
	return hashes;
}

function runDockerCommand(args: string[], options?: ExecFileSyncOptions): string {
	return execFileSync('docker', args, {
		cwd: setupDir,
		stdio: 'pipe',
		maxBuffer: dockerCommandMaxBuffer,
		...options
	})
		.toString()
		.trim();
}

function runDockerComposeCommand(args: string[]): string {
	return runDockerComposeCommandRaw(args).toString().trim();
}

function runDockerComposeCommandRaw(args: string[], options?: ExecFileSyncOptions): Buffer {
	return execFileSync('docker', dockerComposeArgs(args), {
		cwd: setupDir,
		stdio: 'pipe',
		maxBuffer: dockerCommandMaxBuffer,
		...options
	}) as Buffer;
}

function dockerComposeArgs(args: string[]): string[] {
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
