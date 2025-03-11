import { env } from '$env/dynamic/private';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import type { Handle, HandleServerError } from '@sveltejs/kit';
import { AxiosError } from 'axios';
import { decodeJwt } from 'jose';

// Workaround so that we can also import this environment variable into client-side code
// If we would directly import $env/dynamic/private into the api-service.ts file, it would throw an error
// this is still secure as process will just be undefined in the browser
process.env.INTERNAL_BACKEND_URL = env.INTERNAL_BACKEND_URL ?? 'http://localhost:8080';

export const handle: Handle = async ({ event, resolve }) => {
	const { isSignedIn, isAdmin } = verifyJwt(event.cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const isUnauthenticatedOnlyPath = event.url.pathname.startsWith('/login') || event.url.pathname.startsWith('/lc')
	const isPublicPath = ['/authorize', '/health'].includes(event.url.pathname);
	const isAdminPath = event.url.pathname.startsWith('/settings/admin');

	if (!isUnauthenticatedOnlyPath && !isPublicPath) {
		if (!isSignedIn) {
			return new Response(null, {
				status: 302,
				headers: { location: '/login' }
			});
		}
	}

	if (isUnauthenticatedOnlyPath && isSignedIn) {
		return new Response(null, {
			status: 302,
			headers: { location: '/settings' }
		});
	}

	if (isAdminPath && !isAdmin) {
		return new Response(null, {
			status: 302,
			headers: { location: '/settings' }
		});
	}

	const response = await resolve(event);
	return response;
};

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

function verifyJwt(accessToken: string | undefined) {
	let isSignedIn = false;
	let isAdmin = false;

	if (accessToken) {
		const jwtPayload = decodeJwt<{ isAdmin: boolean }>(accessToken);
		if (jwtPayload?.exp && jwtPayload.exp * 1000 > Date.now()) {
			isSignedIn = true;
			isAdmin = jwtPayload?.isAdmin || false;
		}
	}

	return { isSignedIn, isAdmin };
}
