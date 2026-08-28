import { m } from '$lib/paraglide/messages';

export type Jwk = {
	kty?: unknown;
	kid?: unknown;
	alg?: unknown;
	use?: unknown;
	crv?: unknown;
	[key: string]: unknown;
};

export type ParseJwkResult = { ok: true; keys: Jwk[] } | { ok: false; error: string };

/**
 * Parses pasted JWK input into a list of keys.
 * The input can either be a single JWK or a JWKS, in which case each key it contains is imported separately.
 * Full validation is performed by the backend.
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
		return { ok: true, keys: [parsed] };
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
		keys.push(key);
	}

	return { ok: true, keys };
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
