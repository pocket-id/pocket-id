export type Passkey = {
	id: string;
	name: string;
	createdAt: string;
	aaguid: string;
	hasIcon: boolean;
	attestationVerified: boolean;
	verifiedAuthenticatorName?: string;
};
