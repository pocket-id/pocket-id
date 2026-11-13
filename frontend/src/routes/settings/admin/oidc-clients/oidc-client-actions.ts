import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import OIDCService from '#lib/services/oidc-service.ts';
import type { AdvancedTableAction } from '#lib/types/advanced-table.type.ts';
import type { OidcClient } from '#lib/types/oidc.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import EnableIcon from '~icons/solar/check-circle-bold';
import CopyIcon from '~icons/solar/copy-bold';
import BanIcon from '~icons/solar/forbidden-circle-bold';
import RefreshCwIcon from '~icons/solar/refresh-bold';
import Trash2Icon from '~icons/solar/trash-bin-minimalistic-bold';
import { toast } from 'svelte-sonner';

const oidcService = new OIDCService();

export const typeLabel = (client: OidcClient) => (client.isPublic ? m.public() : m.confidential());

export function copyClientIdAction<T extends OidcClient>(): AdvancedTableAction<T> {
	return {
		label: m.copy_client_id(),
		icon: CopyIcon,
		onClick: async (client) => {
			try {
				await navigator.clipboard.writeText(client.id);
				toast.success(m.name_client_id_copied({ name: client.name }));
			} catch {
				toast.error(m.could_not_copy_the_client_id());
			}
		}
	};
}

export function refreshClientAction<T extends OidcClient>(
	client: T,
	onRefreshed: () => unknown
): AdvancedTableAction<T> {
	return {
		label: m.refresh_metadata(),
		icon: RefreshCwIcon,
		// Only metadata document clients have a remote document to refresh from
		hidden: client.clientType !== 'cimd',
		onClick: async (client) => {
			try {
				await oidcService.refreshClient(client.id);
				await onRefreshed();
				toast.success(m.metadata_document_for_name_refreshed({ name: client.name }));
			} catch (e) {
				axiosErrorToast(e);
			}
		}
	};
}

export async function setClientDisabled<T extends OidcClient>(
	client: T,
	disabled: boolean,
	onDone: (disabled: boolean) => unknown
) {
	try {
		await oidcService.updateClient(client.id, { disabled });
		await onDone(disabled);
		toast.success(
			disabled ? m.name_disabled({ name: client.name }) : m.name_enabled({ name: client.name })
		);
	} catch (e) {
		axiosErrorToast(e);
	}
}

export function toggleClientDisabledAction<T extends OidcClient>(
	client: T,
	onDone: (disabled: boolean) => unknown,
	disableLabel = m.disable_ellipsis()
): AdvancedTableAction<T> {
	if (client.disabled) {
		return {
			label: m.enable(),
			icon: EnableIcon,
			onClick: (client) => setClientDisabled(client, false, onDone)
		};
	}
	return {
		label: disableLabel,
		icon: BanIcon,
		onClick: (client) =>
			openConfirmDialog({
				title: m.disable_name_question({ name: client.name }),
				message: m.disable_oidc_client_consequence({ name: client.name }),
				confirm: {
					label: m.disable_name_action({ name: client.name }),
					destructive: true,
					icon: BanIcon,
					action: () => setClientDisabled(client, true, onDone)
				}
			})
	};
}

export function deleteClientAction<T extends OidcClient>(
	onDeleted: () => unknown,
	label = m.delete_ellipsis()
): AdvancedTableAction<T> {
	return {
		label,
		icon: Trash2Icon,
		variant: 'danger',
		onClick: (client) =>
			openConfirmDialog({
				title: m.delete_name_question({ name: client.name }),
				message: m.delete_oidc_client_consequence({ name: client.name }),
				confirm: {
					label: m.delete_name_action({ name: client.name }),
					destructive: true,
					action: async () => {
						try {
							await oidcService.removeClient(client.id);
							await onDeleted();
							toast.success(m.name_deleted({ name: client.name }));
						} catch (e) {
							axiosErrorToast(e);
						}
					}
				}
			})
	};
}
