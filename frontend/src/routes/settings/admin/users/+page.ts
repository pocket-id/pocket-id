import UserService from '$lib/services/user-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();

	const usersRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'firstName',
			direction: 'asc'
		}
	};

	const signupTokensRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'createdAt',
			direction: 'desc'
		}
	};

	const [users, signupTokens] = await Promise.all([
		userService.list(usersRequestOptions),
		userService.listSignupTokens(signupTokensRequestOptions)
	]);

	return {
		users,
		usersRequestOptions,
		signupTokens,
		signupTokensRequestOptions
	};
};
