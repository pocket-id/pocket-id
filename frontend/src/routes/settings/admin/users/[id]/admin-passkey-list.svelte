<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import PasskeyRow from '#lib/components/passkey-row.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import UserService from '#lib/services/user-service.ts';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
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

<div class="mt-3 divide-y overflow-hidden rounded-md border">
	{#each [...passkeys].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) as passkey (passkey.id)}
		<PasskeyRow {passkey} onDelete={() => deletePasskey(passkey)} />
	{/each}
</div>
