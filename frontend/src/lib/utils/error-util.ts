import { m } from '$lib/paraglide/messages';
import { WebAuthnError } from '@simplewebauthn/browser';
import { AxiosError } from 'axios';
import { toast } from 'svelte-sonner';

export function getAxiosErrorMessage(
	e: unknown,
	defaultMessage: string = m.an_unknown_error_occurred()
) {
	let message = defaultMessage;
	if (e instanceof AxiosError) {
		message = e.response?.data.error || message;
	}
	return message;
}

export function axiosErrorToast(e: unknown, defaultMessage: string = m.an_unknown_error_occurred()) {
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
			m.authenticator_does_not_support_any_of_the_requested_algorithms()
	};

	let message = m.an_unknown_error_occurred();
	if (e instanceof WebAuthnError && e.code in errors) {
		message = errors[e.code as keyof typeof errors];
	} else if (e instanceof WebAuthnError && e?.message.includes('timed out')) {
		message = m.authenticator_timed_out();
	} else if (e instanceof AxiosError && e.response?.data.error) {
		message = e.response?.data.error;
	} else {
		console.error(e);
	}
	return message;
}
