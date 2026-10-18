import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
import { m } from '#lib/paraglide/messages.js';
import ApisService from '#lib/services/apis-service.ts';
import type { Api } from '#lib/types/api.type.ts';
import { axiosErrorToast } from '#lib/utils/error-util.ts';
import { toast } from 'svelte-sonner';

const apisService = new ApisService();

// The consequence names the clients that lose access, so it is looked up before the dialog opens
export async function confirmDeleteApi(api: Api, onDeleted: () => unknown) {
	let clients: string[];
	try {
		const res = await apisService.listClients(api.id, { pagination: { page: 1, limit: 100 } });
		clients = res.data
			.filter((entry) => entry.userDelegatedAccess || entry.clientAccess)
			.map((entry) => entry.client.name);
	} catch (e) {
		axiosErrorToast(e);
		return;
	}

	let message: string;
	if (clients.length === 1) message = m.delete_api_consequence_one_client({ client: clients[0] });
	else if (clients.length > 1)
		message = m.delete_api_consequence_clients({
			clients:
				clients.length <= 3
					? m.names_and_last({ names: clients.slice(0, -1).join(', '), last: clients.at(-1)! })
					: m.names_and_others({ names: clients.slice(0, 2).join(', '), count: clients.length - 2 })
		});
	else if (api.allowCimdClients) message = m.delete_api_consequence_cimd();
	else message = m.delete_api_consequence_no_clients();

	openConfirmDialog({
		title: m.delete_name_question({ name: api.name }),
		message,
		confirm: {
			label: m.delete_name_action({ name: api.name }),
			destructive: true,
			action: async () => {
				try {
					await apisService.remove(api.id);
					toast.success(m.name_deleted({ name: api.name }));
					await onDeleted();
				} catch (e) {
					axiosErrorToast(e);
				}
			}
		}
	});
}
