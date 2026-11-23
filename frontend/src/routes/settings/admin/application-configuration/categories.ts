export const categories = [
	'general',
	'features',
	'signup',
	'passkeys',
	'email',
	'ldap',
	'oidc',
	'appearance'
] as const;

export type Category = (typeof categories)[number];
