import { mode } from 'mode-watcher';
import { cachedApplicationLogo } from '$lib/utils/cached-image-util';

export function getLogoUrl() {
	const isLightMode = mode.current === 'light';
	return cachedApplicationLogo.getUrl(isLightMode);
}
