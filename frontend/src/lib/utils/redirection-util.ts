import type { User } from '$lib/types/user.type';

// Returns the path to redirect to based on the current path and user authentication status
// If no redirect is needed, it returns null
export function getAuthRedirectPath(path: string, user: User | null) {
	const isSignedIn = !!user;
	const isAdmin = user?.isAdmin;

	const isUnauthenticatedOnlyPath =
		path == '/login' ||
		path.startsWith('/login/') ||
		path == '/lc' ||
		path.startsWith('/lc/') ||
		path == '/signup' ||
		path == '/signup/setup' ||
		path == '/setup' ||
		path.startsWith('/st/');
	const isPublicPath = ['/authorize', '/device', '/health', '/healthz'].includes(path);
	const isAdminPath = path == '/settings/admin' || path.startsWith('/settings/admin/');

	if (!isUnauthenticatedOnlyPath && !isPublicPath && !isSignedIn) {
		return '/login';
	}

	if (isUnauthenticatedOnlyPath && isSignedIn) {
		return '/settings';
	}

	if (isAdminPath && !isAdmin) {
		return '/settings';
	}
}
