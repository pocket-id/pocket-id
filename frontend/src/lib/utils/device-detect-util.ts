const LIMITED_DEVICE_PATTERNS = [
	// Smart TVs
	'SMART-TV',
	'SmartTV',
	'Web0S',
	'webOS.TV',
	'NetCast',
	'Tizen',
	'BRAVIA',
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
	// Consoles
	'PlayStation',
	'Xbox'
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
