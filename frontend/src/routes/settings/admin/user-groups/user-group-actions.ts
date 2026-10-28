import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import UserGroupService from '#lib/services/user-group-service.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { toast } from 'svelte-sonner';

const userGroupService = new UserGroupService();

export function confirmDeleteGroup(
	group: { id: string; friendlyName: string; memberCount: number },
	onDeleted: () => unknown
) {
	openConfirmDialog({
		title: m.delete_name_question({ name: group.friendlyName }),
		message:
			group.memberCount === 0
				? m.delete_empty_group_consequence()
				: m.delete_group_consequence({ count: group.memberCount }),
		confirm: {
			label: m.delete_name_action({ name: group.friendlyName }),
			destructive: true,
			action: async () => {
				try {
					await userGroupService.remove(group.id);
					toast.success(m.name_deleted({ name: group.friendlyName }));
					await onDeleted();
				} catch (e) {
					axiosErrorToast(e);
				}
			}
		}
	});
}
