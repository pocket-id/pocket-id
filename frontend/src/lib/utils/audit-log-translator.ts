import { m } from '#lib/paraglide/messages.js';

export const eventTypes: Record<string, string> = {
	SIGN_IN: m.audit_event_sign_in(),
	TOKEN_SIGN_IN: m.audit_event_token_sign_in(),
	REMOTE_SIGN_IN: m.audit_event_remote_sign_in(),
	CLIENT_AUTHORIZATION: m.audit_event_client_authorization(),
	NEW_CLIENT_AUTHORIZATION: m.audit_event_new_client_authorization(),
	ACCOUNT_CREATED: m.audit_event_account_created(),
	DEVICE_CODE_AUTHORIZATION: m.audit_event_device_code_authorization(),
	NEW_DEVICE_CODE_AUTHORIZATION: m.audit_event_new_device_code_authorization(),
	PASSKEY_ADDED: m.audit_event_passkey_added(),
	PASSKEY_REMOVED: m.audit_event_passkey_removed(),
	LOGIN_CODE_CREATED: m.audit_event_login_code_created(),
	EMAIL_CHANGED: m.audit_event_email_changed(),
	EMAIL_VERIFIED: m.audit_event_email_verified(),
	API_KEY_CREATED: m.audit_event_api_key_created(),
	API_KEY_RENEWED: m.audit_event_api_key_renewed(),
	API_KEY_REVOKED: m.audit_event_api_key_revoked(),
	CLIENT_SECRET_CREATED: m.audit_event_client_secret_created(),
	CLIENT_SECRET_DELETED: m.audit_event_client_secret_deleted(),
	SESSION_REVOKED: m.audit_event_session_revoked(),
	SIGNED_OUT: m.audit_event_signed_out(),
	CLIENT_AUTHORIZATION_REVOKED: m.audit_event_client_authorization_revoked(),
	ACCOUNT_DELETED: m.audit_event_account_deleted(),
	ACCOUNT_DISABLED: m.audit_event_account_disabled(),
	ACCOUNT_ENABLED: m.audit_event_account_enabled(),
	ADMIN_GRANTED: m.audit_event_admin_granted(),
	ADMIN_REVOKED: m.audit_event_admin_revoked(),
	CLIENT_CREATED: m.audit_event_client_created(),
	CLIENT_UPDATED: m.audit_event_client_updated(),
	CLIENT_DELETED: m.audit_event_client_deleted(),
	CLIENT_GROUP_ADDED: m.audit_event_client_group_added(),
	CLIENT_GROUP_REMOVED: m.audit_event_client_group_removed(),
	GROUP_CREATED: m.audit_event_group_created(),
	GROUP_DELETED: m.audit_event_group_deleted(),
	GROUP_MEMBER_ADDED: m.audit_event_group_member_added(),
	GROUP_MEMBER_REMOVED: m.audit_event_group_member_removed()
};

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
