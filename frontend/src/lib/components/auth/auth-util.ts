import { WebAuthnError } from '@simplewebauthn/browser';
import { page } from '$app/state';
import { m } from '#lib/paraglide/messages.js';
import { cssMs } from '#lib/utils/style.ts';

// The one filled button of an auth step, full width and taller in the desktop card
export const authPrimaryClass = 'h-(--h-xl) w-full text-heading lg:h-13';

// Every other path on an auth step is a quiet text action with a full-height hit area
export const authQuietClass =
	'h-(--h-xl) px-3 text-body font-normal text-fg-muted hover:text-fg aria-busy:text-fg-muted';

// Other ways to sign in carries the current page, so signing in there returns to the app that asked
export function alternativeSignInHref() {
	return page.url.pathname === '/login'
		? `/login/alternative${page.url.search}`
		: `/login/alternative?redirect=${encodeURIComponent(page.url.pathname + page.url.search)}`;
}

// Closing the browser's passkey prompt is the person's choice, so it reads as information rather than an error
export function isPasskeyPromptClosed(e: unknown) {
	return (
		e instanceof WebAuthnError &&
		(e.code === 'ERROR_CEREMONY_ABORTED' ||
			(e.cause as Error | undefined)?.name === 'NotAllowedError')
	);
}

export function takingYouMessage(redirect?: string | null) {
	return !redirect || redirect.startsWith('/settings')
		? m.auth_taking_you_to_your_account()
		: m.auth_taking_you_back();
}

// The success tile shows for a moment before the page moves on
export function successHold() {
	return new Promise((resolve) => setTimeout(resolve, cssMs('--t-success-hold')));
}

// The address a login code was just emailed to, so the code page can say where to look and resend it
export const emailedLoginCode = { sentTo: '' };
