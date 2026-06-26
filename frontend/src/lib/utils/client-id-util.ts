// Raw pocket-id client IDs match this pattern and need no encoding.
const RAW_CLIENT_ID = /^[a-zA-Z0-9._-]+$/;

/**
 * Encodes a client ID for use as a path segment.
 *
 * CIMD client IDs are full https URLs containing slashes and colons, which
 * cannot be carried in a single path segment. Such IDs are encoded as
 * `~<base64url>`; the backend decodes them. Plain client IDs are unchanged.
 */
export function encodeClientIdParam(id: string): string {
	if (RAW_CLIENT_ID.test(id)) {
		return id;
	}
	const base64 = btoa(String.fromCharCode(...new TextEncoder().encode(id)));
	const base64url = base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
	return '~' + base64url;
}

/**
 * Reverses {@link encodeClientIdParam}. Decodes a `~<base64url>` value back to the
 * real client ID; returns plain values unchanged.
 */
export function decodeClientIdParam(param: string): string {
	if (!param.startsWith('~')) {
		return param;
	}
	const base64 = param.slice(1).replace(/-/g, '+').replace(/_/g, '/');
	const binary = atob(base64);
	const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
	return new TextDecoder().decode(bytes);
}
