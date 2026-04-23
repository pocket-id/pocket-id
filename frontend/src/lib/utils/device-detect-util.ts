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
 * Returns true if the browser supports modern CSS (oklch).
 */
export function supportsModernCSS(): boolean {
	if (typeof window === 'undefined') return true;
	return !!window.CSS?.supports?.('color', 'oklch(0 0 0)');
}

/**
 * Checks if the device needs an alternative login flow (no passkey input possible)
 * and returns the appropriate redirect path, or null if no redirect is needed.
 */
export function getAlternativeLoginRedirect(basePath: string): string | null {
	if (!isLimitedDevice() && window.PublicKeyCredential) return null;
	if (supportsModernCSS()) {
		return '/login/alternative' + (basePath ? `?redirect=${encodeURIComponent(basePath)}` : '');
	}
	return '/simple/qr/' + (basePath ? `?redirect=${encodeURIComponent(basePath)}` : '');
}
