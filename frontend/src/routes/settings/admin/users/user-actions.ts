import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import UserService from '#lib/services/user-service.ts';
import type { User } from '#lib/types/user.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { LucideBan } from '@lucide/svelte';
import { toast } from 'svelte-sonner';

const userService = new UserService();

// "Felix Wagner", "Felix Wagner and Hana Sato", "Felix Wagner, Hana Sato and 4 others"
function joinNames(users: User[]) {
	const names = users.map((u) => u.displayName);
	if (names.length === 1) return names[0];
	if (names.length <= 3)
		return m.names_and_last({ names: names.slice(0, -1).join(', '), last: names.at(-1)! });
	return m.names_and_others({ names: names.slice(0, 2).join(', '), count: names.length - 2 });
}

// Runs one request per person and returns how many succeeded, because the API has no bulk endpoints
async function forEachUser(users: User[], run: (user: User) => Promise<unknown>) {
	const results = await Promise.allSettled(users.map(run));
	const failed = results.filter((r) => r.status === 'rejected');
	if (failed.length) axiosErrorToast((failed[0] as PromiseRejectedResult).reason);
	return users.length - failed.length;
}

function setDisabled(user: User, disabled: boolean) {
	return userService.update(user.id, { ...user, disabled });
}

export function confirmDisableUsers(users: User[], onDone: () => unknown) {
	const one = users.length === 1;
	openConfirmDialog({
		title: one
			? m.disable_name_question({ name: users[0].displayName })
			: m.disable_count_users_question({ count: users.length }),
		message: one
			? m.disable_user_consequence({ name: users[0].firstName || users[0].displayName })
			: m.disable_users_consequence({ names: joinNames(users) }),
		confirm: {
			label: one
				? m.disable_name_action({ name: users[0].displayName })
				: m.disable_count_users({ count: users.length }),
			destructive: true,
			icon: LucideBan,
			action: async () => {
				const done = await forEachUser(users, (u) => setDisabled(u, true));
				if (!done) return;
				toast.success(
					one
						? m.name_disabled({ name: users[0].displayName })
						: m.count_users_disabled({ count: done })
				);
				await onDone();
			}
		}
	});
}

export async function enableUsers(users: User[], onDone: () => unknown) {
	const done = await forEachUser(users, (u) => setDisabled(u, false));
	if (!done) return;
	toast.success(
		users.length === 1
			? m.name_enabled({ name: users[0].displayName })
			: m.count_users_enabled({ count: done })
	);
	await onDone();
}

export function confirmDeleteUsers(users: User[], onDone: () => unknown) {
	const one = users.length === 1;
	openConfirmDialog({
		title: one
			? m.delete_name_question({ name: users[0].displayName })
			: m.delete_count_users_question({ count: users.length }),
		message: m.delete_users_consequence({ names: joinNames(users) }),
		confirm: {
			label: one
				? m.delete_name_action({ name: users[0].displayName })
				: m.delete_count_users({ count: users.length }),
			destructive: true,
			action: async () => {
				const done = await forEachUser(users, (u) => userService.remove(u.id));
				if (!done) return;
				toast.success(
					one
						? m.name_deleted({ name: users[0].displayName })
						: m.count_users_deleted({ count: done })
				);
				await onDone();
			}
		}
	});
}
