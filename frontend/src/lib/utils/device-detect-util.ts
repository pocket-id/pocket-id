const LIMITED_DEVICE_PATTERNS = [
	// Smart TVs
	'SMART-TV',
	'SmartTV',
	'webOS',
	'Web0S',
	'webOS.TV',
	'NetCast',
	'Tizen',
	'BRAVIA',
	'Hisense',
	'VIDAA',
	'Vizio',
	'PhilipsTV',
	'Panasonic',
	'Vestel',
	'Sharp',
	'Funai',
	// Streaming / Set-top
	'Android TV',
	'GoogleTV',
	'AppleTV',
	'tvOS',
	'Roku',
	'CrKey',
	'AFT',
	'AmazonWebAppPlatform',
	// Automotive
	'Tesla',
	'CarPlay',
	'Android Auto',
	'Polestar',
	'Rivian',
	'MBUX',
	'Audi connect',
	'BMW Connected',
	'Volvo Cars',
	// Consoles
	'PlayStation',
	'Xbox',
	'Nintendo',
	'Valve Steam',
	// Other limited browsers
	'HbbTV',
	'Silk'
];

// All client-detection helpers default to "no redirect" during SSR (typeof window === 'undefined')
// so that the SvelteKit app is rendered server-side and the actual capability check runs in the
// browser. supportsOklch returns true on SSR for the same reason: we assume modern UI is fine
// until proven otherwise.

export function isLimitedDevice(): boolean {
	if (typeof window === 'undefined') return false;
	const ua = navigator.userAgent;
	return LIMITED_DEVICE_PATTERNS.some((pattern) => ua.includes(pattern));
}

export function supportsOklch(): boolean {
	if (typeof window === 'undefined') return true;
	return !!window.CSS?.supports?.('color', 'oklch(0 0 0)');
}

export function needsAlternativeLogin(): boolean {
	if (typeof window === 'undefined') return false;
	return isLimitedDevice() || !window.PublicKeyCredential;
}

export function getAlternativeLoginPath(queryString: string): string {
	const base = supportsOklch() ? '/login/alternative' : '/simple/qr/';
	return base + queryString;
}

// Navigates to the alternative-login page, picking goto() vs full reload depending on the target.
// /simple/* lives outside the SvelteKit SPA, so it must be a hard navigation.
export function navigateToAlternativeLogin(
	queryString: string,
	goto: (path: string) => Promise<void> | void
): void {
	const target = getAlternativeLoginPath(queryString);
	if (target.startsWith('/simple/')) {
		window.location.href = target;
	} else {
		goto(target);
	}
}
