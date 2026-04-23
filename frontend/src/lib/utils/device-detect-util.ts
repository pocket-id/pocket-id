const LIMITED_DEVICE_PATTERNS = [
	// Smart TVs
	'SMART-TV',
	'SmartTV',
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

/**
 * Detects whether the current browser is a limited device (Smart TV, console, etc.)
 * that cannot use passkey/WebAuthn authentication.
 */
export function isLimitedDevice(): boolean {
	if (typeof window === 'undefined') return false;
	const ua = navigator.userAgent;
	return LIMITED_DEVICE_PATTERNS.some((pattern) => ua.includes(pattern));
}

/**
 * Returns true if the browser supports oklch colors (used by the SvelteKit UI).
 */
export function supportsOklch(): boolean {
	if (typeof window === 'undefined') return true;
	return !!window.CSS?.supports?.('color', 'oklch(0 0 0)');
}

/**
 * Returns true if the device needs an alternative login flow (no passkey input possible).
 */
export function needsAlternativeLogin(): boolean {
	if (typeof window === 'undefined') return false;
	return isLimitedDevice() || !window.PublicKeyCredential;
}

/**
 * Returns the redirect path for a device that needs alternative login.
 * `queryString` is appended as-is (e.g. `?redirect=%2Fauthorize%3F...`).
 */
export function getAlternativeLoginPath(queryString: string): string {
	const base = supportsOklch() ? '/login/alternative' : '/simple/qr/';
	return base + queryString;
}
