import { goto } from '$app/navigation';
import AppConfigService from '$lib/services/app-config-service';
import UserService from '$lib/services/user-service';
import type { User } from '$lib/types/user.type';
import type { LayoutLoad } from './$types';

export const ssr = false;

export const load: LayoutLoad = async ({ url }) => {
	const userService = new UserService();
	const appConfigService = new AppConfigService();

	const userPromise = userService.getCurrent().catch(() => null);

	const appConfigPromise = appConfigService.list().catch((e) => {
		console.error(
			`Failed to get application configuration: ${e.response?.data.error || e.message}`
		);
		return null;
	});

	const [user, appConfig] = await Promise.all([userPromise, appConfigPromise]);

	const redirectPath = await getRedirectPath(url.pathname, user);
	if (redirectPath) {
		goto(redirectPath);
	}

	return {
		user,
		appConfig
	};
};

const getRedirectPath = async (path: string, user: User | null) => {
	const isSignedIn = !!user;
	const isAdmin = user?.isAdmin;

	const isUnauthenticatedOnlyPath =
		path == '/login' || path.startsWith('/login/') || path == '/lc' || path.startsWith('/lc/');
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
};
