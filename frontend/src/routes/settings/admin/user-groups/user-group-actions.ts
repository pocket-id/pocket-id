import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import UserGroupService from '#lib/services/user-group-service.ts';
import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
import type { UserGroup } from '#lib/types/user-group.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { LucideTrash } from '@lucide/svelte';
import { toast } from 'svelte-sonner';

const userGroupService = new UserGroupService();

export function deleteUserGroupAction<T extends Pick<UserGroup, 'id' | 'name'>>(
	onDeleted: () => unknown
): AdvancedTableAction<T> {
	return {
		label: m.delete(),
		icon: LucideTrash,
		variant: 'danger',
		onClick: (userGroup) =>
			openConfirmDialog({
				title: m.delete_name({ name: userGroup.name }),
				message: m.are_you_sure_you_want_to_delete_this_user_group(),
				confirm: {
					label: m.delete(),
					destructive: true,
					action: async () => {
						try {
							await userGroupService.remove(userGroup.id);
							await onDeleted();
							toast.success(m.user_group_deleted_successfully());
						} catch (e) {
							axiosErrorToast(e);
						}
					}
				}
			})
	};
}
