import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import UserService from '#lib/services/user-service.ts';
import userStore from '#lib/stores/user-store.ts';
import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
import type { User } from '#lib/types/user.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { LucideLink, LucideTrash } from '@lucide/svelte';
import { toast } from 'svelte-sonner';
import { get } from 'svelte/store';

const userService = new UserService();

export function loginCodeAction(onClick: (user: User) => void): AdvancedTableAction<User> {
	return { label: m.login_code(), icon: LucideLink, onClick };
}

export function deleteUserAction(user: User, onDeleted: () => unknown): AdvancedTableAction<User> {
	return {
		label: m.delete(),
		icon: LucideTrash,
		variant: 'danger',
		onClick: (user) =>
			openConfirmDialog({
				title: m.delete_firstname_lastname({
					firstName: user.firstName,
					lastName: user.lastName ?? ''
				}),
				message: m.are_you_sure_you_want_to_delete_this_user(),
				confirm: {
					label: m.delete(),
					destructive: true,
					action: async () => {
						try {
							await userService.remove(user.id);
							await onDeleted();
							toast.success(m.user_deleted_successfully());
						} catch (e) {
							axiosErrorToast(e);
						}
					}
				}
			}),
		hidden: !!user.ldapId && !user.disabled,
		disabled: user.id === get(userStore)?.id
	};
}
