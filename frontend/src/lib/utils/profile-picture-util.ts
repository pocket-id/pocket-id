import { browser } from '$app/environment';

type SkipCacheUntil = {
	[key: string]: number;
};

export function getProfilePictureUrl(userId?: string) {
	if (!userId) return '';

	let url = `/api/users/${userId}/profile-picture.png`;

	if (browser) {
		const skipCacheUntil = getSkipCacheUntil(userId);
		const skipCache = skipCacheUntil > Date.now();
		if (skipCache) {
			const skipCacheParam = new URLSearchParams();
			skipCacheParam.append('skip-cache', skipCacheUntil.toString());
			url += '?' + skipCacheParam.toString();
		}
	}

	return url.toString();
}

function getSkipCacheUntil(userId: string) {
	const skipCacheUntil: SkipCacheUntil = JSON.parse(
		localStorage.getItem('skip-cache-until') ?? '{}'
	);
	return skipCacheUntil[userId] ?? 0;
}

export function bustProfilePictureCache(userId: string) {
	const skipCacheUntil: SkipCacheUntil = JSON.parse(
		localStorage.getItem('skip-cache-until') ?? '{}'
	);
	skipCacheUntil[userId] = Date.now() + 1000 * 60 * 15; // 15 minutes
	localStorage.setItem('skip-cache-until', JSON.stringify(skipCacheUntil));
}
