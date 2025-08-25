import { m } from '$lib/paraglide/messages';

export const eventTypes: Record<string, string> = {
	SIGN_IN: m.sign_in(),
	TOKEN_SIGN_IN: m.token_sign_in(),
	CLIENT_AUTHORIZATION: m.client_authorization(),
	NEW_CLIENT_AUTHORIZATION: m.new_client_authorization(),
	ACCOUNT_CREATED: m.account_created()
}

/**
 * Translates an audit log event type using paraglide messages.
 * Falls back to a formatted string if no specific translation is found.
 * @param event The event type string from the backend (e.g., "CLIENT_AUTHORIZATION").
 * @returns The translated string.
 */
export function translateAuditLogEvent(event: string): string {
	if (event in eventTypes) {
		return eventTypes[event];
	}

	// If no specific translation is found, provide a readable fallback.
	// This converts "SOME_EVENT" to "Some Event".
	const words = event.split('_');
	const capitalizedWords = words.map((word) => {
		return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
	});
	return capitalizedWords.join(' ');
}
