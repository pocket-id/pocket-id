import type { HandleClientError } from '@sveltejs/kit';
import { AxiosError } from 'axios';

export const handleError: HandleClientError = async ({ error, message, status }) => {
	if (error instanceof AxiosError) {
		message = error.response?.data.error || message;
		status = error.response?.status || status;
		console.error(
			`Axios error: ${error.request.path} - ${error.response?.data.error ?? error.message}`
		);
	} else {
		console.error(error);
	}

	return {
		message,
		status
	};
};
