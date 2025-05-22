import type { CDPSession, Page } from '@playwright/test';

// The existing passkeys are already stored in the database
const passkeys = {
	tim: {
		credentialId: 'test-credential-tim',
		userHandle: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		privateKey:
			'MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg3rNKkGApsEA1TpGiphKh6axTq3Vh6wBghLLea/YkIp+hRANCAATBw6jkpXXr0pHrtAQetxiR5cTcILG/YGDCdKrhVhNDHIu12YrF6B7Frwl3AUqEpdrYEwj3Fo3XkGgvrBIJEUmG'
	},
	craig: {
		credentialId: 'test-credential-craig',
		userHandle: '1cd19686-f9a6-43f4-a41f-14a0bf5b4036',
		privateKey:
			'MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgL1UaeWG1KYpN+HcxQvXEJysiQjT9Fn7Zif3i5cY+s+yhRANCAASPioDQ+tnODwKjULbufJRvOunwTCOvt46UYjYt+vOZsvmc+FlEB0neERqqscxKckGF8yq1AYrANiloshAUAouH'
	},
	timNew: {
		credentialId: 'new-test-credential-tim',
		userHandle: 'f4b89dc2-62fb-46bf-9f5f-c34f4eafe93e',
		privateKey:
			'MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgFl2lIlRyc2G7O9D8WWrw2N8D7NTlhgWcKFY7jYxrfcmhRANCAASmvbCFrXshUvW7avTIysV9UymbhmUwGb7AonUMQPgqK2Jur7PWp9V0AIe5YMuXYH1oxsqY5CoAbdY2YsPmhYoX'
	}
};

async function init(page: Page) {
	const client = await page.context().newCDPSession(page);
	await client.send('WebAuthn.enable');
	const authenticatorId = await addVirtualAuthenticator(client);

	return {
		addPasskey: async (passkey?: keyof typeof passkeys) => {
			await addPasskey(authenticatorId, client, passkey);
		}
	};
}

async function addVirtualAuthenticator(client: CDPSession): Promise<string> {
	const result = await client.send('WebAuthn.addVirtualAuthenticator', {
		// config authenticator
		options: {
			protocol: 'ctap2',
			transport: 'internal',
			hasResidentKey: true,
			hasUserVerification: true,
			isUserVerified: true
		}
	});
	return result.authenticatorId;
}

async function addPasskey(
	authenticatorId: string,
	client: CDPSession,
	passkeyName: keyof typeof passkeys = 'tim'
): Promise<void> {
	const passkey = passkeys[passkeyName];
	await client.send('WebAuthn.addCredential', {
		authenticatorId,
		credential: {
			credentialId: btoa(passkey.credentialId),
			isResidentCredential: true,
			rpId: 'localhost',
			privateKey: passkey.privateKey,
			userHandle: btoa(passkey.userHandle),
			signCount: Math.round((new Date().getTime() - 1704444610871) / 1000 / 2)
		}
	});
}

export default { init };
