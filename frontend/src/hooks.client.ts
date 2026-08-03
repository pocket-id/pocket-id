import type { HandleClientError } from '@sveltejs/kit';
import { isAxiosError } from 'axios';
import { getAxiosErrorMessage, getAxiosErrorRequestId } from '$lib/utils/error-util';

export const handleError: HandleClientError = async ({ error, message, status }) => {
	if (isAxiosError(error)) {
		message = getAxiosErrorMessage(error, message);
		status = error.response?.status || status;
		console.error(
			`Axios error: ${error.request?.path ?? 'unknown path'} - ${getAxiosErrorMessage(error, error.message)}`,
			{ requestId: getAxiosErrorRequestId(error) }
		);
	} else {
		console.error(error);
	}

	return {
		message,
		status
	};
};
