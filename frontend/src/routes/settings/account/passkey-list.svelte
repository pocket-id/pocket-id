<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import PasskeyRow from '#lib/components/passkey-row.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import WebauthnService from '#lib/services/webauthn-service.ts';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { toast } from 'svelte-sonner';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';

	let { passkeys = $bindable() }: { passkeys: Passkey[] } = $props();

	const webauthnService = new WebauthnService();

	let passkeyToRename: Passkey | null = $state(null);

	async function deletePasskey(passkey: Passkey) {
		openConfirmDialog({
			title: m.delete_passkey_name({ passkeyName: passkey.name }),
			message: m.are_you_sure_you_want_to_delete_this_passkey(),
			confirm: {
				label: m.delete_name_action({ name: passkey.name }),
				destructive: true,
				action: async () => {
					try {
						await webauthnService.removeCredential(passkey.id);
						passkeys = await webauthnService.listCredentials();
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
	{#each passkeys as passkey (passkey.id)}
		<PasskeyRow
			{passkey}
			onRename={() => (passkeyToRename = passkey)}
			onDelete={() => deletePasskey(passkey)}
		/>
	{/each}
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
