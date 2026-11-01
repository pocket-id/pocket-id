import { m } from '#lib/paraglide/messages.js';
import type { Session } from '#lib/types/session.type.ts';

export function sessionDeviceName(session: Session) {
	if (session.browser && session.os)
		return m.browser_on_os({ browser: session.browser, os: session.os });
	return session.browser || session.os || m.unknown_device();
}

export function sessionLocation(session: Session) {
	return [session.city, session.country].filter(Boolean).join(', ') || m.unknown();
}

export function sessionSignInMethod(session: Session) {
	switch (session.authenticationMethod) {
		case 'phr':
			return m.passkey();
		case 'otp':
			return m.sign_in_method_login_code();
		default:
			return m.sign_in_method_account_creation();
	}
}
