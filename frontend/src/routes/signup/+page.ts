import UserService from '$lib/services/user-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const token = url.searchParams.get('token') || undefined;

	let requiredEmailDomain: string | null = null;
	if (token) {
		// Best-effort lookup of the token's public metadata to hint the required email domain.
		// Failures (e.g. an invalid or expired token) are ignored here; the signup request itself enforces validity.
		const userService = new UserService();
		try {
			const info = await userService.getSignupTokenInfo(token);
			requiredEmailDomain = info.emailDomain ?? null;
		} catch {
			requiredEmailDomain = null;
		}
	}

	return {
		token,
		requiredEmailDomain
	};
};
