import { expect, test } from '@playwright/test';
import AdmZip from 'adm-zip';
import { execSync } from 'child_process';
import crypto from 'crypto';
import { users } from 'data';
import fs from 'fs';
import path from 'path';
import { cleanupBackend } from 'utils/cleanup.util';
import { pathFromRoot } from 'utils/fs.util';

const containerName = 'pocket-id';
const setupDir = pathFromRoot('setup');
const tmpDir = pathFromRoot('.tmp');
const exampleExportPath = pathFromRoot('resources/export');

test('Export', async ({ baseURL }) => {
	// Reset the backend but with LDAP setup because the example export has no LDAP data
	await cleanupBackend({ skipLdapSetup: true });

	// Export the data from the seeded container
	const exportPath = path.join(tmpDir, 'export.zip');
	const extractPath = path.join(tmpDir, 'export-extracted');

	// Fetch the profile pictures because they get generated on demand
	await Promise.all([
		fetch(`${baseURL}/api/users/${users.craig.id}/profile-picture.png`),
		fetch(`${baseURL}/api/users/${users.tim.id}/profile-picture.png`)
	]);

	runExport(exportPath);
	unzipExport(exportPath, extractPath);

	compareExports(exampleExportPath, extractPath);
});

test('Import', async () => {
	// Reset the backend without seeding
	await cleanupBackend({ skipSeed: true });

	// Run the import with the example export data
	const exampleExportArchivePath = path.join(tmpDir, 'example-export.zip');
	archiveExampleExport(exampleExportArchivePath);

	// runDockerCommand(`docker compose stop pocket-id`);
	runImport(exampleExportArchivePath);
	// runDockerCommand(`docker compose up -d pocket-id`);

	// Export again from the imported instance
	const exportPath = path.join(tmpDir, 'export.zip');
	const exportExtracted = path.join(tmpDir, 'export-extracted');
	runExport(exportPath);
	unzipExport(exportPath, exportExtracted);

	compareExports(exampleExportPath, exportExtracted);
});

function compareExports(dir1: string, dir2: string): void {
	const hashes1 = hashAllFiles(dir1);
	const hashes2 = hashAllFiles(dir2);

	const files1 = Object.keys(hashes1).sort();
	const files2 = Object.keys(hashes2).sort();
	expect(files1).toEqual(files2);

	for (const file of files1) {
		expect(hashes1[file], `${file} hash should match`).toEqual(hashes2[file]);
	}

	// Compare database.json contents
	const expectedData = loadJSON(path.join(dir1, 'database.json'));
	const actualData = loadJSON(path.join(dir2, 'database.json'));

	// Check special fields
	validateSpecialFields(actualData);

	// Normalize and compare
	const normalized1 = normalizeJSON(expectedData);
	const normalized2 = normalizeJSON(actualData);
	expect(normalized1).toEqual(normalized2);
}

function archiveExampleExport(outputPath: string) {
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

	fs.writeFileSync(outputPath, zip.toBuffer());
}

// Helper to load JSON files
function loadJSON(path: string) {
	return JSON.parse(fs.readFileSync(path, 'utf-8'));
}

function normalizeJSON(obj: any): any {
	if (Array.isArray(obj)) {
		// Sort arrays to make order irrelevant
		return obj
			.map(normalizeJSON)
			.sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b)));
	} else if (obj && typeof obj === 'object') {
		const ignoredKeys = ['id', 'created_at', 'expires_at', 'credentials'];

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
					isUnixTimestamp(value),
					`Expected '${key}' = ${value} to be a valid UNIX timestamp`
				).toBe(true);
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

function isUnixTimestamp(value: any): boolean {
	if (typeof value !== 'number') return false;
	// Rough range: after 2000-01-01 and before 2100-01-01
	return value > 946684800 && value < 4102444800;
}

function runImport(pathToFile: string) {
	const importContainerId = runDockerCommand(
		`docker compose run -d -v ${pathToFile}:/app/pocket-id-export.zip ${containerName} /app/pocket-id import --yes`
	);
	try {
		runDockerCommand(`docker wait ${importContainerId}`);
	} finally {
		runDockerCommand(`docker rm -f ${importContainerId}`);
	}
}

function runExport(outputFile: string): void {
	const containerId = runDockerCommand(
		`docker compose run -d ${containerName} /app/pocket-id export`
	);
	try {
		// Wait until export finishes
		runDockerCommand(`docker wait ${containerId}`);
		runDockerCommand(`docker cp ${containerId}:/app/pocket-id-export.zip ${outputFile}`);
	} finally {
		runDockerCommand(`docker rm -f ${containerId}`);
	}
	expect(fs.existsSync(outputFile)).toBe(true);
}

function unzipExport(zipFile: string, destDir: string): void {
	fs.rmSync(destDir, { recursive: true, force: true });
	const zip = new AdmZip(zipFile);
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

function runDockerCommand(cmd: string): string {
	return execSync(cmd, { cwd: setupDir, stdio: 'pipe' }).toString().trim();
}
