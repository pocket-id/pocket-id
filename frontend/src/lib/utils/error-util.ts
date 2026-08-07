import { m } from '$lib/paraglide/messages';
import { WebAuthnError } from '@simplewebauthn/browser';
import { isAxiosError } from 'axios';
import { toast } from 'svelte-sonner';

interface ApiErrorResponse {
	error?: unknown;
	code?: string;
	request_id?: string;
}

const codeMessages: Record<string, () => string> = {
	internal_error: () => m.an_unknown_error_occurred(),
	request_timeout: () => m.please_try_again(),
	rate_limited: () => m.please_try_again(),
	not_signed_in: () => m.please_try_to_sign_in_again(),
	reauthentication_required: () => m.please_try_to_sign_in_again(),
	invalid_webauthn_session: () => m.passkey_request_expired(),
	invalid_webauthn_response: () => m.passkey_response_invalid(),
	webauthn_authentication_failed: () => m.passkey_verification_failed(),
	passkey_user_verification_required: () => m.passkey_user_verification_required(),
	synced_passkey_not_allowed: () => m.synced_passkeys_not_allowed(),
	device_login_expired: () => m.device_login_request_expired()
};

function getApiErrorResponse(e: unknown): ApiErrorResponse | undefined {
	if (!isAxiosError(e)) {
		return undefined;
	}

	const data = e.response?.data;
	if (!data || typeof data !== 'object') {
		return undefined;
	}

	return data as ApiErrorResponse;
}

export function getAxiosErrorMessage(
	e: unknown,
	defaultMessage: string = m.an_unknown_error_occurred()
) {
	const response = getApiErrorResponse(e);
	if (!response) {
		return defaultMessage;
	}

	const codeMessage = response.code ? codeMessages[response.code]?.() : undefined;
	return codeMessage || (typeof response.error === 'string' ? response.error : defaultMessage);
}

export function getAxiosErrorRequestId(e: unknown): string | undefined {
	if (!isAxiosError(e)) {
		return undefined;
	}

	const bodyRequestId = getApiErrorResponse(e)?.request_id;
	if (bodyRequestId) {
		return bodyRequestId;
	}

	const headerRequestId = e.response?.headers?.['x-request-id'];
	return typeof headerRequestId === 'string' ? headerRequestId : undefined;
}

export function axiosErrorToast(
	e: unknown,
	defaultMessage: string = m.an_unknown_error_occurred()
) {
	const message = getAxiosErrorMessage(e, defaultMessage);
	toast.error(message);
}

export function getWebauthnErrorMessage(e: unknown) {
	const errors = {
		ERROR_CEREMONY_ABORTED: m.authentication_process_was_aborted(),
		ERROR_AUTHENTICATOR_GENERAL_ERROR: m.error_occurred_with_authenticator(),
		ERROR_AUTHENTICATOR_MISSING_DISCOVERABLE_CREDENTIAL_SUPPORT:
			m.authenticator_does_not_support_discoverable_credentials(),
		ERROR_AUTHENTICATOR_MISSING_RESIDENT_KEY_SUPPORT:
			m.authenticator_does_not_support_resident_keys(),
		ERROR_AUTHENTICATOR_PREVIOUSLY_REGISTERED: m.passkey_was_previously_registered(),
		ERROR_AUTHENTICATOR_NO_SUPPORTED_PUBKEYCREDPARAMS_ALG:
			m.authenticator_does_not_support_any_of_the_requested_algorithms(),
		ERROR_INVALID_DOMAIN: `${m.webauthn_error_invalid_domain()} ${m.contact_administrator_to_fix()}`,
		ERROR_INVALID_RP_ID: `${m.webauthn_error_invalid_rp_id()} ${m.contact_administrator_to_fix()}`,
		NotSupportedError: m.webauthn_not_supported_by_browser(),
		NotAllowedError: m.webauthn_operation_not_allowed_or_timed_out()
	};

	const response = getApiErrorResponse(e);
	let message: string = m.an_unknown_error_occurred();
	if (e instanceof WebAuthnError && e.code in errors) {
		message = errors[e.code as keyof typeof errors];
	} else if (e instanceof WebAuthnError && e.cause instanceof Error && e.cause.name in errors) {
		message = errors[e.cause.name as keyof typeof errors];
	} else if (isAxiosError(e) && (response?.code || response?.error)) {
		message = getAxiosErrorMessage(e);
	} else {
		console.error(e);
	}
	return message;
}
