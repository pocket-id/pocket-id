<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import PasskeyRow from '#lib/components/passkey-row.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import WebAuthnService from '#lib/services/webauthn-service.ts';
	import type { Passkey } from '#lib/types/passkey.type.ts';
	import { axiosErrorToast, getWebauthnErrorMessage } from '#lib/utils/error-util.ts';
	import { LucidePlus } from '@lucide/svelte';
	import { startRegistration } from '@simplewebauthn/browser';
	import { toast } from 'svelte-sonner';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';

	let { passkeys = $bindable() }: { passkeys: Passkey[] } = $props();

	const webauthnService = new WebAuthnService();

	let passkeyToRename: Passkey | null = $state(null);

	async function refresh() {
		passkeys = await webauthnService.listCredentials();
	}

	async function createPasskey() {
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

<SettingsSection title={m.passkeys()} description={m.passkeys_section_description()}>
	{#snippet action()}
		{#if passkeys.length > 0}
			<Button variant="secondary" size="sm" usePromiseLoading onclick={createPasskey}>
				<LucidePlus />
				{m.add_passkey()}
			</Button>
		{/if}
	{/snippet}

	<div class="flex flex-col gap-3 pt-2">
		{#if passkeys.length > 0}
			<!-- Collections inside a section are open rows with hairlines, so the shared row loses its own inset -->
			<div
				role="list"
				aria-label={m.passkeys()}
				class="*:*:min-h-[68px] *:*:px-0 *:shadow-[inset_0_-1px_0_var(--border)] *:last:shadow-none"
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
		{:else}
			<div class="flex min-h-11 flex-wrap items-center justify-between gap-2">
				<p class="text-control text-fg-muted">{m.no_passkeys_yet()}</p>
				<Button variant="ghost" size="sm" usePromiseLoading onclick={createPasskey}>
					<LucidePlus />
					{m.add_passkey()}
				</Button>
			</div>
		{/if}
	</div>
</SettingsSection>

<RenamePasskeyModal bind:passkey={passkeyToRename} callback={refresh} />
