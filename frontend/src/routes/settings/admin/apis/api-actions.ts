import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import ApisService from '#lib/services/apis-service.ts';
import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
import type { Api } from '#lib/types/api.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { LucideTrash } from '@lucide/svelte';
import { toast } from 'svelte-sonner';

const apisService = new ApisService();

export function deleteApiAction(onDeleted: () => unknown): AdvancedTableAction<Api> {
	return {
		label: m.delete(),
		icon: LucideTrash,
		variant: 'danger',
		onClick: (api) =>
			openConfirmDialog({
				title: m.delete_name({ name: api.name }),
				message: m.are_you_sure_you_want_to_delete_this_api(),
				confirm: {
					label: m.delete(),
					destructive: true,
					action: async () => {
						try {
							await apisService.remove(api.id);
							await onDeleted();
							toast.success(m.api_deleted_successfully());
						} catch (e) {
							axiosErrorToast(e);
						}
					}
				}
			})
	};
}
