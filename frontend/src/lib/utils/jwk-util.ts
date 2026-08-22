import { m } from '$lib/paraglide/messages';

export type Jwk = {
	kty?: unknown;
	kid?: unknown;
	alg?: unknown;
	use?: unknown;
	crv?: unknown;
	[key: string]: unknown;
};

// Members that only exist on private keys, which must never be sent to Pocket ID
const privateKeyMembers = ['d', 'p', 'q', 'dp', 'dq', 'qi', 'oth', 'k'];

// Members that a public key must have, by key type
// This is excluding symmetric (OKP) keys
const requiredMembers: Record<string, string[]> = {
	RSA: ['n', 'e'],
	EC: ['crv', 'x', 'y'],
	OKP: ['crv', 'x']
};

export type ParseJwkResult = { ok: true; keys: Jwk[] } | { ok: false; error: string };

/**
 * Parses pasted JWK input into a list of keys.
 * The input can either be a single JWK or a JWKS, in which case each key it contains is imported separately.
 */
export function parseJwkInput(input: string): ParseJwkResult {
	const trimmed = input.trim();
	if (!trimmed) {
		return { ok: false, error: m.paste_a_public_key_or_a_jwks() };
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(trimmed);
	} catch {
		return { ok: false, error: m.the_value_is_not_valid_json() };
	}

	if (!isJsonObject(parsed)) {
		return { ok: false, error: m.the_value_is_not_a_jwk_or_a_jwks() };
	}

	// A JWKS holds its keys in a "keys" array, everything else is treated as a single key
	if (!('keys' in parsed)) {
		const error = validateJwk(parsed);
		return error ? { ok: false, error } : { ok: true, keys: [parsed] };
	}

	if (!Array.isArray(parsed.keys) || parsed.keys.length === 0) {
		return { ok: false, error: m.the_jwks_does_not_contain_any_key() };
	}

	const keys: Jwk[] = [];
	for (const [index, key] of parsed.keys.entries()) {
		const number = index + 1;
		if (!isJsonObject(key)) {
			return { ok: false, error: m.jwks_key_is_not_a_jwk({ number }) };
		}

		const error = validateJwk(key);
		if (error) {
			return { ok: false, error: m.jwks_key_is_invalid({ number, error }) };
		}
		keys.push(key);
	}

	return { ok: true, keys };
}

/**
 * Validates that a JWK is a public key Pocket ID can verify signatures with.
 * Returns an error message, or null when the key is valid.
 */
export function validateJwk(key: Jwk): string | null {
	if (typeof key.kty !== 'string' || !key.kty) {
		return m.the_key_is_missing_the_property({ property: 'kty' });
	}

	const required = requiredMembers[key.kty];
	if (!required) {
		return m.keys_of_this_type_cannot_verify_signatures({ keyType: key.kty });
	}

	const privateMember = privateKeyMembers.find((member) => member in key);
	if (privateMember) {
		return m.the_key_contains_private_key_material({ property: privateMember });
	}

	const missing = required.find((member) => typeof key[member] !== 'string' || !key[member]);
	if (missing) {
		return m.the_key_is_missing_the_property({ property: missing });
	}

	// Pocket ID selects the key to verify an assertion with by its key ID, so it is always required
	if (typeof key.kid !== 'string' || !key.kid) {
		return m.the_key_is_missing_the_property({ property: 'kid' });
	}

	// A key restricted to encryption can never verify a signature
	if (typeof key.use === 'string' && key.use && key.use !== 'sig') {
		return m.the_key_is_not_meant_to_verify_signatures({ use: key.use });
	}

	return null;
}

/**
 * Returns a short description of a key, such as "RSA · RS256", to show next to its key ID.
 */
export function describeJwk(key: Jwk): string {
	return [key.kty, key.crv, key.alg]
		.filter((part): part is string => typeof part === 'string' && !!part)
		.join(' · ');
}

export function getJwkKeyId(key: Jwk): string {
	return typeof key.kid === 'string' ? key.kid : '';
}

function isJsonObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}
