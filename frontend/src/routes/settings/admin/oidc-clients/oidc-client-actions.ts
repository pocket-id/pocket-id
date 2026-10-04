import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import OIDCService from '#lib/services/oidc-service.ts';
import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
import type { OidcClient } from '#lib/types/oidc.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { LucideRefreshCcw, LucideTrash } from '@lucide/svelte';
import { toast } from 'svelte-sonner';

const oidcService = new OIDCService();

export function refreshClientAction<T extends OidcClient>(
	client: T,
	onRefreshed: () => unknown
): AdvancedTableAction<T> {
	return {
		label: m.refresh(),
		icon: LucideRefreshCcw,
		// Only metadata document clients have a remote document to refresh from
		hidden: client.clientType !== 'cimd',
		onClick: async (client) => {
			try {
				await oidcService.refreshClient(client.id);
				await onRefreshed();
				toast.success(m.oidc_client_metadata_refreshed_successfully());
			} catch (e) {
				axiosErrorToast(e);
			}
		}
	};
}

export function deleteClientAction<T extends OidcClient>(
	onDeleted: () => unknown
): AdvancedTableAction<T> {
	return {
		label: m.delete(),
		icon: LucideTrash,
		variant: 'danger',
		onClick: (client) =>
			openConfirmDialog({
				title: m.delete_name({ name: client.name }),
				message: m.are_you_sure_you_want_to_delete_this_oidc_client(),
				confirm: {
					label: m.delete(),
					destructive: true,
					action: async () => {
						try {
							await oidcService.removeClient(client.id);
							await onDeleted();
							toast.success(m.oidc_client_deleted_successfully());
						} catch (e) {
							axiosErrorToast(e);
						}
					}
				}
			})
	};
}
