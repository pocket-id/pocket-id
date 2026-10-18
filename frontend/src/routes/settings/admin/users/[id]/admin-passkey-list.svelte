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
		firstName,
		passkeys = $bindable()
	}: {
		userId: string;
		firstName: string;
		passkeys: Passkey[];
	} = $props();

	const userService = new UserService();

	const sorted = $derived(
		[...passkeys].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
	);

	function deletePasskey(passkey: Passkey) {
		openConfirmDialog({
			title: m.delete_name_question({ name: passkey.name }),
			message:
				passkeys.length === 1
					? m.delete_last_user_passkey_consequence({ name: firstName })
					: m.delete_user_passkey_consequence({ name: firstName }),
			confirm: {
				label: m.delete_name_action({ name: passkey.name }),
				destructive: true,
				action: async () => {
					try {
						await userService.removeUserPasskey(userId, passkey.id);
						passkeys = await userService.listUserPasskeys(userId);
						toast.success(m.passkey_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<div class="border-t">
	{#each sorted as passkey (passkey.id)}
		<div class="border-b">
			<PasskeyRow {passkey} onDelete={() => deletePasskey(passkey)} />
		</div>
	{:else}
		<p class="text-control text-fg-muted flex min-h-(--row-2line) items-center border-b px-4">
			{m.user_has_no_passkeys_yet_give_login_code({ name: firstName })}
		</p>
	{/each}
</div>
