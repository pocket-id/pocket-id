import type { HandleClientError } from '@sveltejs/kit/hooks';
import { isAxiosError } from 'axios';
import { getAxiosErrorMessage, getAxiosErrorRequestId } from '#lib/utils/error-util.ts';

export const handleError: HandleClientError = ({ kind, error }) => {
	// Preserve safe application and framework errors, including their status codes
	if (kind !== 'unknown') return error;

	// Retain API error messages and request IDs for failed backend requests
	if (isAxiosError(error)) {
		const message = getAxiosErrorMessage(error);
		const status = error.response?.status || 500;
		console.error(
			`Axios error: ${error.request?.path ?? 'unknown path'} - ${getAxiosErrorMessage(error, error.message)}`,
			{ requestId: getAxiosErrorRequestId(error) }
		);
		return { message, status };
	}

	// Keep SvelteKit's generic response for unexpected errors while logging the cause
	console.error(error);
};
