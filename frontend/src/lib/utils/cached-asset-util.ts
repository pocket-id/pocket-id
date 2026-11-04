import { encodeClientIdParam } from './client-id-util';

type SkipCacheUntil = {
	[key: string]: number;
};

type CachedAsset = {
	getUrl: (...props: any[]) => string;
	bustCache: (...props: any[]) => void;
};

export const cachedApplicationLogo: CachedAsset = {
	getUrl: (light = true) => getCachedAssetUrl(applicationLogoUrl(light)),
	bustCache: (light = true) => bustAssetCache(applicationLogoUrl(light))
};

// The UI renders its own default logo, so the bundled logo is skipped to be able to tell whether a custom logo has been uploaded
function applicationLogoUrl(light: boolean) {
	const url = new URL('/api/application-assets/logo', window.location.origin);
	if (!light) url.searchParams.set('light', 'false');
	url.searchParams.set('default', 'false');
	return url;
}

export const cachedEmailLogo: CachedAsset = {
	getUrl: () => getCachedAssetUrl(new URL('/api/application-assets/email', window.location.origin)),
	bustCache: () => bustAssetCache(new URL('/api/application-assets/email', window.location.origin))
};

export const cachedDefaultProfilePicture: CachedAsset = {
	getUrl: () =>
		getCachedAssetUrl(
			new URL('/api/application-assets/default-profile-picture', window.location.origin)
		),
	bustCache: () =>
		bustAssetCache(
			new URL('/api/application-assets/default-profile-picture', window.location.origin)
		)
};

export const cachedBackgroundImage: CachedAsset = {
	getUrl: () =>
		getCachedAssetUrl(new URL('/api/application-assets/background', window.location.origin)),
	bustCache: () =>
		bustAssetCache(new URL('/api/application-assets/background', window.location.origin))
};

export const cachedTitleFont: CachedAsset = {
	getUrl: () => getCachedAssetUrl(new URL('/api/application-assets/font', window.location.origin)),
	bustCache: () => bustAssetCache(new URL('/api/application-assets/font', window.location.origin))
};

export const cachedProfilePicture: CachedAsset = {
	getUrl: (userId: string) => {
		const url = new URL(`/api/users/${userId}/profile-picture.png`, window.location.origin);
		return getCachedAssetUrl(url);
	},
	bustCache: (userId: string) => {
		const url = new URL(`/api/users/${userId}/profile-picture.png`, window.location.origin);
		bustAssetCache(url);
	}
};

export const cachedOidcClientLogo: CachedAsset = {
	getUrl: (clientId: string, light = true) => getCachedAssetUrl(oidcClientLogoUrl(clientId, light)),
	// Each variant falls back to the other one, so changing either logo can change what both URLs serve
	bustCache: (clientId: string) => {
		bustAssetCache(oidcClientLogoUrl(clientId, true));
		bustAssetCache(oidcClientLogoUrl(clientId, false));
	}
};

function oidcClientLogoUrl(clientId: string, light: boolean) {
	const url = new URL(
		`/api/oidc/clients/${encodeClientIdParam(clientId)}/logo`,
		window.location.origin
	);
	if (!light) url.searchParams.set('light', 'false');
	return url;
}

// Builds the URL of an authenticator icon
// Unlike the assets above these are static assets that only change with a new release, so they skip the cache busting helper and rely on the backend's cache headers
export function authenticatorIconUrl(aaguid: string, light = true) {
	const url = new URL(
		`/api/webauthn/authenticator-icons/${encodeURIComponent(aaguid)}`,
		window.location.origin
	);
	if (!light) url.searchParams.set('light', 'false');
	return url.pathname + (url.search ? `?${url.searchParams.toString()}` : '');
}

function getCachedAssetUrl(url: URL) {
	const baseKey = normalizeUrlForKey(url);
	const skipCacheUntil = getSkipCacheUntil(baseKey);
	const skipCache = skipCacheUntil > Date.now();

	const finalUrl = new URL(url.toString());
	if (skipCache) {
		finalUrl.searchParams.set('skip-cache', skipCacheUntil.toString());
	}

	return finalUrl.pathname + (finalUrl.search ? `?${finalUrl.searchParams.toString()}` : '');
}

function bustAssetCache(url: URL) {
	const key = normalizeUrlForKey(url);
	const expiresAt = Date.now() + 1000 * 60 * 15;

	const store: SkipCacheUntil = JSON.parse(localStorage?.getItem('skip-cache-until') ?? '{}');
	store[key] = expiresAt;
	localStorage?.setItem('skip-cache-until', JSON.stringify(store));
}

function getSkipCacheUntil(key: string): number {
	const store: SkipCacheUntil = JSON.parse(localStorage?.getItem('skip-cache-until') ?? '{}');
	return store[key] ?? 0;
}

// Removes transient params and normalizes query order before hashing
function normalizeUrlForKey(url: URL) {
	const u = new URL(url.toString());
	u.searchParams.delete('skip-cache');

	const sortedParams = new URLSearchParams(
		[...u.searchParams.entries()].sort(([a], [b]) => a.localeCompare(b))
	);
	const normalized = u.pathname + (sortedParams.toString() ? `?${sortedParams.toString()}` : '');
	return hashKey(normalized);
}

function hashKey(key: string): string {
	let hash = 0;
	for (let i = 0; i < key.length; i++) {
		const char = key.charCodeAt(i);
		hash = (hash << 5) - hash + char;
		hash |= 0;
	}
	return Math.abs(hash).toString(36);
}
