import { version as currentVersion } from '$app/environment';
import axios from 'axios';

const VERSION_CACHE_KEY = 'version_cache';
const CACHE_DURATION = 2 * 60 * 60 * 1000; // 2 hours

async function getNewestVersion() {
	const cachedData = await getVersionFromCache();

	// If we have valid cached data, return it
	if (cachedData) {
		return cachedData;
	}

	// Otherwise fetch from API
	try {
		const response = await axios
			.get('https://api.github.com/repos/pocket-id/pocket-id/releases/latest', {
				timeout: 2000
			})
			.then((res) => res.data);
		console.log('Fetched newest version:', response);
		const newestVersion = response.tag_name.replace('v', '');

		// Cache the result
		cacheVersion(newestVersion);

		return newestVersion;
	} catch (error) {
		console.error('Failed to fetch newest version:', error);
		// If fetch fails but we have an expired cache, return that as fallback
		const cache = getCacheObject();
		return cache?.newestVersion || currentVersion;
	}
}

function getCurrentVersion() {
	return currentVersion;
}

async function isUpToDate() {
	const newestVersion = await getNewestVersion();
	const currentVersion = getCurrentVersion();

	// If the current version changed, invalidate the cache
	const cache = getCacheObject();
	if (cache?.lastCurrentVersion && currentVersion !== cache.lastCurrentVersion) {
		invalidateCache();
	}

	return newestVersion === currentVersion;
}

// Helper methods for caching
function getCacheObject() {
	const cacheJson = localStorage.getItem(VERSION_CACHE_KEY);
	if (!cacheJson) return null;

	try {
		return JSON.parse(cacheJson);
	} catch (e) {
		console.error('Failed to parse cache:', e);
		return null;
	}
}

async function getVersionFromCache() {
	const cache = getCacheObject();

	if (!cache || !cache.newestVersion || !cache.timestamp) {
		return null;
	}

	const now = Date.now();

	// Check if cache is still valid
	if (now - cache.timestamp > CACHE_DURATION) {
		invalidateCache();
		return null;
	}

	// Check if current version matches what it was when we cached
	if (cache.lastCurrentVersion && cache.lastCurrentVersion !== currentVersion) {
		invalidateCache();
		return null;
	}

	return cache.newestVersion;
}

async function cacheVersion(version: string) {
	const cacheObject = {
		newestVersion: version,
		timestamp: Date.now(),
		lastCurrentVersion: currentVersion
	};

	localStorage.setItem(VERSION_CACHE_KEY, JSON.stringify(cacheObject));
}

async function invalidateCache() {
	localStorage.removeItem(VERSION_CACHE_KEY);
}

export default {
	getNewestVersion,
	getCurrentVersion,
	isUpToDate
};
