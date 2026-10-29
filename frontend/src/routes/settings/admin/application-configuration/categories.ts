export const categories = [
	'general',
	'features',
	'signup',
	'passkeys',
	'email',
	'ldap',
	'oidc',
	'branding'
] as const;

export type Category = (typeof categories)[number];
