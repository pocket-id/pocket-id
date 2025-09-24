type SkipCacheUntil = {
	[key: string]: number;
};

type CachableImage = {
	getUrl: (...props: any[]) => string;
	bustCache: (...props: any[]) => void;
};

export const cachedApplicationLogo: CachableImage = {
	getUrl: (light = true) => {
		let url = '/api/application-images/logo';
		if (!light) {
			url += '?light=false';
		}
		return getCachedImageUrl(url);
	},
	bustCache: (light = true) => {
		let url = '/api/application-images/logo';
		if (!light) {
			url += '?light=false';
		}
		bustImageCache(url);
	}
};

export const cachedBackgroundImage: CachableImage = {
	getUrl: () => getCachedImageUrl('/api/application-images/background'),
	bustCache: () => bustImageCache('/api/application-images/background')
};

export const cachedProfilePicture: CachableImage = {
	getUrl: (userId: string) => {
		const url = `/api/users/${userId}/profile-picture.png`;
		return getCachedImageUrl(url);
	},
	bustCache: (userId: string) => {
		const url = `/api/users/${userId}/profile-picture.png`;
		bustImageCache(url);
	}
};

export const cachedOidcClientLogo: CachableImage = {
	getUrl: (clientId: string) => {
		const url = `/api/oidc/clients/${clientId}/logo`;
		return getCachedImageUrl(url);
	},
	bustCache: (clientId: string) => {
		const url = `/api/oidc/clients/${clientId}/logo`;
		bustImageCache(url);
	}
};

function getCachedImageUrl(url: string) {
	const skipCacheUntil = getSkipCacheUntil(url);
	const skipCache = skipCacheUntil > Date.now();
	if (skipCache) {
		const skipCacheParam = new URLSearchParams();
		skipCacheParam.append('skip-cache', skipCacheUntil.toString());
		url += '?' + skipCacheParam.toString();
	}

	return url.toString();
}

function bustImageCache(url: string) {
	const skipCacheUntil: SkipCacheUntil = JSON.parse(
		localStorage.getItem('skip-cache-until') ?? '{}'
	);
	skipCacheUntil[hashKey(url)] = Date.now() + 1000 * 60 * 15; // 15 minutes
	localStorage.setItem('skip-cache-until', JSON.stringify(skipCacheUntil));
}

function getSkipCacheUntil(url: string) {
	const skipCacheUntil: SkipCacheUntil = JSON.parse(
		localStorage.getItem('skip-cache-until') ?? '{}'
	);
	return skipCacheUntil[hashKey(url)] ?? 0;
}

function hashKey(key: string): string {
	let hash = 0;
	for (let i = 0; i < key.length; i++) {
		const char = key.charCodeAt(i);
		hash = (hash << 5) - hash + char;
		hash = hash & hash;
	}
	return Math.abs(hash).toString(36);
}
