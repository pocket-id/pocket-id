import type { User } from '$lib/types/user.type';

// Returns the path to redirect to based on the current path and user authentication status
// If no redirect is needed, it returns null
export function getAuthRedirectPath(url: URL, user: User | null) {
	const path = url.pathname;
	const isSignedIn = !!user;
	const isAdmin = user?.isAdmin;

	const isUnauthenticatedOnlyPath =
		path == '/login' ||
		(path.startsWith('/login/') && path != '/login/alternative/code') ||
		path == '/lc' ||
		path == '/signup' ||
		path == '/signup/setup' ||
		path == '/setup' ||
		path.startsWith('/st/');

	const isPublicPath =
		path.startsWith('/lc/') ||
		['/authorize', '/login/alternative/code', '/device', '/health', '/healthz'].includes(path);

	const isAdminPath = path == '/settings/admin' || path.startsWith('/settings/admin/');

	if (!isUnauthenticatedOnlyPath && !isPublicPath && !isSignedIn) {
		const redirect = url.pathname + url.search;
		return `/login?redirect=${encodeURIComponent(redirect)}`;
	}

	if (isUnauthenticatedOnlyPath && isSignedIn) {
		return '/settings';
	}

	if (isAdminPath && !isAdmin) {
		return '/settings';
	}

	return null;
}
