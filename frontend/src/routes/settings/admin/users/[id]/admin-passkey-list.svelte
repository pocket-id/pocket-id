<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import PasskeyRow from '$lib/components/passkey-row.svelte';
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { Passkey } from '$lib/types/passkey.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideKeyRound } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let {
		userId,
		passkeys = $bindable()
	}: {
		userId: string;
		passkeys: Passkey[];
	} = $props();

	const userService = new UserService();

	async function refreshPasskeys() {
		passkeys = await userService.listUserPasskeys(userId);
	}

	function deletePasskey(passkey: Passkey) {
		openConfirmDialog({
			title: m.delete_passkey_name({ passkeyName: passkey.name }),
			message: m.are_you_sure_you_want_to_delete_this_passkey(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await userService.removeUserPasskey(userId, passkey.id);
						await refreshPasskeys();
						toast.success(m.passkey_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<Item.Group class="mt-3">
	{#each [...passkeys].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) as passkey}
		<PasskeyRow
			label={passkey.name}
			description={m.added_on() + ' ' + new Date(passkey.createdAt).toLocaleDateString()}
			icon={LucideKeyRound}
			showRenameAction={false}
			onDelete={() => deletePasskey(passkey)}
		/>
	{/each}
</Item.Group>
