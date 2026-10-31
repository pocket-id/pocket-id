import fs from 'node:fs';
import playwrightConfig from '../playwright.config';
import { pathFromRoot } from './fs.util';

const authFile = pathFromRoot('.tmp/auth/user.json');

// The access token saved by auth.setup.ts, so the reset endpoint can re-track the session the reset just wiped
function savedAccessToken(): string | null {
	try {
		const state = JSON.parse(fs.readFileSync(authFile, 'utf-8'));
		const cookie = state.cookies?.find(
			(c: { name: string; value: string }) =>
				c.name === '__Host-access_token' || c.name === 'access_token'
		);
		return cookie?.value ?? null;
	} catch {
		return null;
	}
}

export async function cleanupBackend({ skipSeed = false, skipLdapSetup = false } = {}) {
	const url = new URL('/api/test/reset', playwrightConfig.use!.baseURL);

	if (process.env.SKIP_LDAP_TESTS === 'true' || skipSeed || skipLdapSetup) {
		url.searchParams.append('skip-ldap', 'true');
	}

	if (skipSeed) {
		url.searchParams.append('skip-seed', 'true');
	}

	const accessToken = savedAccessToken();
	const response = await fetch(url, {
		method: 'POST',
		headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
	});

	if (!response.ok) {
		throw new Error(`Failed to reset backend: ${response.status} ${response.statusText}`);
	}
}

export async function cleanupScimServiceProvider() {
	if (!process.env.SCIM_SERVICE_PROVIDER_URL) return;
	await fetch(`${process.env.SCIM_SERVICE_PROVIDER_URL}/reset`, { method: 'POST' });
}
