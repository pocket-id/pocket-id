export interface IconFallbackState {
	triedUrls: Set<string>;
	clear(): void;
}

export function createIconFallbackState(): IconFallbackState {
	const triedUrls = new Set<string>();

	return {
		triedUrls,
		clear() {
			triedUrls.clear();
		}
	};
}

export function resolveIconUrl(iconRef: string | undefined): string | undefined {
	if (!iconRef) return undefined;

	if (iconRef.startsWith('http') || iconRef.startsWith('/')) {
		return iconRef;
	}

	if (iconRef.startsWith('sh-')) {
		const iconName = iconRef
			.replace('sh-', '')
			.replace('.svg', '')
			.replace('.png', '')
			.replace('.webp', '');

		// If specific extension is requested, return that
		if (iconRef.endsWith('.svg')) {
			return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/svg/${iconName}.svg`;
		} else if (iconRef.endsWith('.png')) {
			return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/png/${iconName}.png`;
		} else if (iconRef.endsWith('.webp')) {
			return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/webp/${iconName}.webp`;
		} else {
			// Default to SVG for selfhst
			return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/svg/${iconName}.svg`;
		}
	}

	if (iconRef.endsWith('.svg')) {
		const iconName = iconRef.replace('.svg', '');
		return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/${iconName}.svg`;
	}

	if (iconRef.endsWith('.webp')) {
		const iconName = iconRef.replace('.webp', '');
		return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/webp/${iconName}.webp`;
	}

	const iconName = iconRef.replace('.png', '');
	return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/${iconName}.png`;
}

export function getNextFallbackUrl(
	currentSrc: string,
	icon: string | undefined,
	fallbackState: IconFallbackState
): string {
	// If we've already tried this URL, don't try again
	if (fallbackState.triedUrls.has(currentSrc)) {
		return '/img/logo.png';
	}

	fallbackState.triedUrls.add(currentSrc);

	if (!icon || !icon.startsWith('sh-')) {
		return '/img/logo.png';
	}

	const iconName = icon
		.replace('sh-', '')
		.replace('.svg', '')
		.replace('.png', '')
		.replace('.webp', '');

	// Try different formats in order: svg -> png -> webp -> dashboard-icons
	if (currentSrc.includes('selfhst/icons') && currentSrc.includes('/svg/')) {
		return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/png/${iconName}.png`;
	} else if (currentSrc.includes('selfhst/icons') && currentSrc.includes('/png/')) {
		return `https://cdn.jsdelivr.net/gh/selfhst/icons@main/webp/${iconName}.webp`;
	} else if (currentSrc.includes('selfhst/icons') && currentSrc.includes('/webp/')) {
		return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/${iconName}.svg`;
	} else if (currentSrc.includes('dashboard-icons/svg/')) {
		return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/${iconName}.png`;
	} else if (currentSrc.includes('dashboard-icons/png/')) {
		return `https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/webp/${iconName}.webp`;
	} else {
		// All fallbacks failed, use logo
		return '/img/logo.png';
	}
}
