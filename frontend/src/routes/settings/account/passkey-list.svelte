<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import { LucideKeyRound } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import RenamePasskeyModal from './rename-passkey-modal.svelte';
	import { m } from '$lib/paraglide/messages';
	import WebauthnService from '$lib/services/webauthn-service';
	import type { Passkey } from '$lib/types/passkey.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import GlassRowItem from '$lib/components/glass-row-item.svelte';

	let { passkeys = $bindable() }: { passkeys: Passkey[] } = $props();

	const webauthnService = new WebauthnService();

	let passkeyToRename: Passkey | null = $state(null);

	async function deletePasskey(item: any) {
		const passkey = item as Passkey;
		openConfirmDialog({
			title: m.delete_passkey_name({ passkeyName: passkey.name }),
			message: m.are_you_sure_you_want_to_delete_this_passkey(),
			confirm: {
				label: m.delete(),
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

	// Function to determine if a passkey was recently added (within the last 7 days)
	function isRecentlyAdded(date: string): boolean {
		const createdDate = new Date(date);
		const currentDate = new Date();
		const differenceInDays = Math.floor(
			(currentDate.getTime() - createdDate.getTime()) / (1000 * 3600 * 24)
		);
		return differenceInDays <= 7;
	}

	function handleRenamePasskey(item: any) {
		passkeyToRename = item as Passkey;
	}
</script>

<div class="space-y-3">
	{#each passkeys as passkey, i}
		<GlassRowItem
			item={passkey}
			icon={LucideKeyRound}
			onRename={handleRenamePasskey}
			onDelete={deletePasskey}
			showBadge={isRecentlyAdded(passkey.createdAt)}
			badgeText="New"
			dateLabel={m.added_on()}
		/>
	{/each}
</div>

<RenamePasskeyModal
	bind:passkey={passkeyToRename}
	callback={async () => (passkeys = await webauthnService.listCredentials())}
/>
