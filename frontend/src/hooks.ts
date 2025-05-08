import { paraglideMiddleware } from '$lib/paraglide/server';
import type { Handle, HandleServerError } from '@sveltejs/kit';
import { AxiosError } from 'axios';

// Handle to use the paraglide middleware
const paraglideHandle: Handle = ({ event, resolve }) => {
	return paraglideMiddleware(event.request, ({ locale }) => {
		return resolve(event, {
			transformPageChunk: ({ html }) => html.replace('%lang%', locale)
		});
	});
};

export const handle: Handle = paraglideHandle;

export const handleError: HandleServerError = async ({ error, message, status }) => {
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
