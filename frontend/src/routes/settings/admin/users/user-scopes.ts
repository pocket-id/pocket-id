import UserService from '#lib/services/user-service.ts';

export type Scope = 'all' | 'admins' | 'disabled' | 'unverified';

export const scopeFilters: Record<Scope, Record<string, boolean[]>> = {
	all: {},
	admins: { isAdmin: [true] },
	disabled: { disabled: [true] },
	unverified: { emailVerified: [false] }
};

export async function countUsersByScope(): Promise<Record<Scope, number>> {
	const userService = new UserService();
	const [all, admins, disabled, unverified] = await Promise.all(
		Object.values(scopeFilters).map((filters) =>
			userService
				.list({ pagination: { page: 1, limit: 1 }, filters })
				.then((res) => res.pagination.totalItems)
		)
	);
	return { all, admins, disabled, unverified };
}
