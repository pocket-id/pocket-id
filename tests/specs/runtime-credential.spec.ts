import { generateKeyPairSync, sign } from 'node:crypto';
import test, { expect, request } from '@playwright/test';
import { cleanupBackend } from '../utils/cleanup.util';

test.beforeEach(async () => await cleanupBackend());

// FCA14 exercises headless bootstrap, proof login, administrator parity, path safeguards, and future-authentication-only revocation
test('Agent runtime completes bootstrap, authenticates as an admin, and survives credential revocation only in its existing session', async ({
	page
}) => {
	const createdUserResponse = await page.request.post('/api/users', {
		data: {
			username: 'vex-runtime',
			email: 'vex-runtime@test.com',
			emailVerified: true,
			firstName: 'Vex',
			lastName: 'Runtime',
			displayName: 'Vex Runtime',
			isAdmin: true,
			disabled: false,
			isAgent: true
		}
	});
	expect(createdUserResponse.ok()).toBe(true);
	const createdUser = (await createdUserResponse.json()) as {
		id: string;
		username: string;
		isAgent: boolean;
	};
	expect(createdUser.isAgent).toBe(true);

	const tokenResponse = await page.request.post(
		`/api/users/${createdUser.id}/one-time-access-token`,
		{ data: {} }
	);
	expect(tokenResponse.ok()).toBe(true);
	const { token } = (await tokenResponse.json()) as { token: string };

	const runtime = await request.newContext({
		baseURL: test.info().project.use.baseURL,
		storageState: { cookies: [], origins: [] }
	});
	const { publicKey, privateKey } = generateKeyPairSync('ed25519');
	const publicJwk = publicKey.export({ format: 'jwk' });
	expect(publicJwk.x).toBeTruthy();

	try {
		const unauthorizedList = await runtime.get(`/api/users/${createdUser.id}/runtime-credentials`);
		expect(unauthorizedList.status()).toBe(401);
		const unauthorizedSelectorChange = await runtime.put(`/api/users/${createdUser.id}`, {
			data: { ...createdUser, isAgent: false }
		});
		expect(unauthorizedSelectorChange.status()).toBe(401);

		const registrationStart = await runtime.post('/api/runtime-credentials/register/start', {
			data: {
				token,
				name: 'Vex OpenClaw test runtime',
				algorithm: 'Ed25519',
				publicKey: publicJwk.x
			}
		});
		expect(registrationStart.ok()).toBe(true);
		const registrationChallenge = (await registrationStart.json()) as {
			sessionId: string;
			challenge: string;
		};

		const registrationFinish = await runtime.post('/api/runtime-credentials/register/finish', {
			data: {
				sessionId: registrationChallenge.sessionId,
				signature: sign(
					null,
					Buffer.from(registrationChallenge.challenge, 'base64url'),
					privateKey
				).toString('base64url')
			}
		});
		expect(registrationFinish.ok()).toBe(true);
		const registered = (await registrationFinish.json()) as {
			user: { id: string; isAdmin: boolean; isAgent: boolean };
			credential: { id: string; name: string; algorithm: string };
		};
		expect(registered.user).toMatchObject({
			id: createdUser.id,
			isAdmin: true,
			isAgent: true
		});
		expect(registered.credential).toMatchObject({
			name: 'Vex OpenClaw test runtime',
			algorithm: 'Ed25519'
		});

		const replay = await runtime.post('/api/runtime-credentials/register/finish', {
			data: { sessionId: registrationChallenge.sessionId, signature: 'invalid' }
		});
		expect(replay.ok()).toBe(false);

		const adminFunction = await runtime.get('/api/users');
		expect(adminFunction.ok()).toBe(true);
		const adminCredentialList = await page.request.get(
			`/api/users/${createdUser.id}/runtime-credentials`
		);
		expect(adminCredentialList.ok()).toBe(true);
		expect(await adminCredentialList.json()).toEqual([
			expect.objectContaining({
				id: registered.credential.id,
				name: 'Vex OpenClaw test runtime',
				algorithm: 'Ed25519',
				revokedAt: null
			})
		]);

		await runtime.dispose();
		const repeatRuntime = await request.newContext({
			baseURL: test.info().project.use.baseURL,
			storageState: { cookies: [], origins: [] }
		});
		try {
			const loginStart = await repeatRuntime.post('/api/runtime-credentials/login/start', {
				data: {
					username: createdUser.username,
					credentialId: registered.credential.id
				}
			});
			expect(loginStart.ok()).toBe(true);
			const loginChallenge = (await loginStart.json()) as {
				sessionId: string;
				challenge: string;
			};
			const loginFinish = await repeatRuntime.post('/api/runtime-credentials/login/finish', {
				data: {
					sessionId: loginChallenge.sessionId,
					signature: sign(
						null,
						Buffer.from(loginChallenge.challenge, 'base64url'),
						privateKey
					).toString('base64url')
				}
			});
			expect(loginFinish.ok()).toBe(true);

			const revoke = await page.request.delete(
				`/api/users/${createdUser.id}/runtime-credentials/${registered.credential.id}`
			);
			expect(revoke.ok()).toBe(true);

			const existingSession = await repeatRuntime.get('/api/users/me');
			expect(existingSession.ok()).toBe(true);
			const deniedFutureLogin = await request.newContext({
				baseURL: test.info().project.use.baseURL,
				storageState: { cookies: [], origins: [] }
			});
			try {
				const deniedStart = await deniedFutureLogin.post('/api/runtime-credentials/login/start', {
					data: {
						username: createdUser.username,
						credentialId: registered.credential.id
					}
				});
				expect(deniedStart.ok()).toBe(false);
			} finally {
				await deniedFutureLogin.dispose();
			}
		} finally {
			await repeatRuntime.dispose();
		}
	} finally {
		await runtime.dispose().catch(() => undefined);
	}
});
