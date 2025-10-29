type SkipCacheUntil = {
	[key: string]: number;
};

type CachableImage = {
	getUrl: (...props: any[]) => string;
	bustCache: (...props: any[]) => void;
};

export const cachedApplicationLogo: CachableImage = {
	getUrl: (light = true) => {
		const url = new URL('/api/application-images/logo', window.location.origin);
		if (!light) url.searchParams.set('light', 'false');
		return getCachedImageUrl(url);
	},
	bustCache: (light = true) => {
		const url = new URL('/api/application-images/logo', window.location.origin);
		if (!light) url.searchParams.set('light', 'false');
		bustImageCache(url);
	}
};

export const cachedDefaultProfilePicture: CachableImage = {
	getUrl: () =>
		getCachedImageUrl(new URL('/api/application-images/default-profile-picture', window.location.origin)),
	bustCache: () =>
		bustImageCache(new URL('/api/application-images/default-profile-picture', window.location.origin))
};

export const cachedBackgroundImage: CachableImage = {
	getUrl: () =>
		getCachedImageUrl(new URL('/api/application-images/background', window.location.origin)),
	bustCache: () =>
		bustImageCache(new URL('/api/application-images/background', window.location.origin))
};

export const cachedProfilePicture: CachableImage = {
	getUrl: (userId: string) => {
		const url = new URL(`/api/users/${userId}/profile-picture.png`, window.location.origin);
		return getCachedImageUrl(url);
	},
	bustCache: (userId: string) => {
		const url = new URL(`/api/users/${userId}/profile-picture.png`, window.location.origin);
		bustImageCache(url);
	}
};

export const cachedOidcClientLogo: CachableImage = {
	getUrl: (clientId: string, light = true) => {
		const url = new URL(`/api/oidc/clients/${clientId}/logo`, window.location.origin);
		if (!light) url.searchParams.set('light', 'false');
		return getCachedImageUrl(url);
	},
	bustCache: (clientId: string, light = true) => {
		const url = new URL(`/api/oidc/clients/${clientId}/logo`, window.location.origin);
		if (!light) url.searchParams.set('light', 'false');
		bustImageCache(url);
	}
};

function getCachedImageUrl(url: URL) {
	const baseKey = normalizeUrlForKey(url);
	const skipCacheUntil = getSkipCacheUntil(baseKey);
	const skipCache = skipCacheUntil > Date.now();

	const finalUrl = new URL(url.toString());
	if (skipCache) {
		finalUrl.searchParams.set('skip-cache', skipCacheUntil.toString());
	}

	return finalUrl.pathname + (finalUrl.search ? `?${finalUrl.searchParams.toString()}` : '');
}

function bustImageCache(url: URL) {
	const key = normalizeUrlForKey(url);
	const expiresAt = Date.now() + 1000 * 60 * 15;

	const store: SkipCacheUntil = JSON.parse(localStorage.getItem('skip-cache-until') ?? '{}');
	store[key] = expiresAt;
	localStorage.setItem('skip-cache-until', JSON.stringify(store));
}

function getSkipCacheUntil(key: string): number {
	const store: SkipCacheUntil = JSON.parse(localStorage.getItem('skip-cache-until') ?? '{}');
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
