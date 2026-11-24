import { error } from '@sveltejs/kit';

/**
 * Encodes a value as exactly one path segment of a URL
 * The server and proxies in front of it decode the path before routing, so a value that would still contain a slash or be a dot segment after decoding is refused instead of encoded
 * No resource has such an ID, so it is answered as not found
 */
export function pathSegment(value: string): string {
	if (value === '' || value === '.' || value === '..' || /[/\\]/.test(value)) {
		error(404, 'Not found');
	}
	return encodeURIComponent(value);
}
