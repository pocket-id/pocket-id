<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import PasskeyRow from '#lib/components/passkey-row.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { axiosErrorToast, getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import PlusIcon from '~icons/solar/add-circle-bold';
	import { startRegistration } from '@simplewebauthn/browser';
	import { toast } from 'svelte-sonner';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';

	let { passkeys = $bindable() }: { passkeys: Passkey[] } = $props();

	const webauthnService = new WebAuthnService();

	let passkeyToRename: Passkey | null = $state(null);

	async function refresh() {
		passkeys = await webauthnService.listCredentials();
	}

	export async function createPasskey() {
		try {
			const opts = await webauthnService.getRegistrationOptions();
			const attResp = await startRegistration({ optionsJSON: opts });
			const passkey = await webauthnService.finishRegistration(attResp);

			await refresh();
			passkeyToRename = passkey;
		} catch (e) {
			toast.error(getWebauthnErrorMessage(e));
		}
	}

	function deletePasskey(passkey: Passkey) {
		openConfirmDialog({
			title: m.delete_name_question({ name: passkey.name }),
			message:
				passkeys.length === 1
					? m.delete_last_own_passkey_consequence()
					: m.delete_own_passkey_consequence(),
			confirm: {
				label: m.delete_name_action({ name: passkey.name }),
				destructive: true,
				action: async () => {
					try {
						await webauthnService.removeCredential(passkey.id);
						await refresh();
						toast.success(m.passkey_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<div class="flex flex-col items-start gap-2">
	{#if passkeys.length > 0}
		<!-- Open rows with hairlines take the setting rows' padding, tile size and gap, and the list is pulled up like the Preferences and Developers rows so the first name starts at the panel top -->
		<div
			role="list"
			aria-label={m.passkeys()}
			class="-mt-4 self-stretch *:*:gap-4 *:*:px-0 *:*:py-4 *:shadow-[inset_0_-1px_0_var(--border)] *:*:*:first:size-9 *:last:shadow-none"
		>
			{#each passkeys as passkey (passkey.id)}
				<div role="listitem">
					<PasskeyRow
						{passkey}
						onRename={() => (passkeyToRename = passkey)}
						onDelete={() => deletePasskey(passkey)}
					/>
				</div>
			{/each}
		</div>
	{/if}
	<Button variant="secondary" size="sm" usePromiseLoading onclick={createPasskey}>
		<PlusIcon />
		{m.add_passkey()}
	</Button>
</div>

<RenamePasskeyModal bind:passkey={passkeyToRename} callback={refresh} />
