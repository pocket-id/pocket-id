import type { User } from '$lib/types/user.type';

export function mapUser(data: User): User {
	return {
		...data,
		birthDate: data.birthDate ? new Date(data.birthDate) : undefined
	};
}
