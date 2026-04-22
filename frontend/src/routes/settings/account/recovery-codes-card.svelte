<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import { Button } from '$lib/components/ui/button';
	import * as Item from '$lib/components/ui/item/index.js';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { RecoveryCodeStatus } from '$lib/types/recovery-code.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideLifeBuoy } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import RecoveryCodesModal from './recovery-codes-modal.svelte';

	let {
		status = $bindable()
	}: {
		status: RecoveryCodeStatus;
	} = $props();

	const userService = new UserService();

	let showCodes: boolean = $state(false);
	let generatedCodes: string[] = $state([]);
	let isWorking = $state(false);

	const hasCodes = $derived(status.total > 0);

	async function generate() {
		isWorking = true;
		try {
			const batch = await userService.generateRecoveryCodes();
			generatedCodes = batch.codes;
			showCodes = true;
			status = { total: batch.codes.length, unused: batch.codes.length };
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isWorking = false;
		}
	}

	function confirmGenerate() {
		if (!hasCodes) {
			generate();
			return;
		}
		openConfirmDialog({
			title: m.regenerate_recovery_codes_title(),
			message: m.regenerate_recovery_codes_description(),
			confirm: {
				label: m.regenerate(),
				destructive: true,
				action: generate
			}
		});
	}

	function confirmRevoke() {
		openConfirmDialog({
			title: m.revoke_recovery_codes_title(),
			message: m.revoke_recovery_codes_description(),
			confirm: {
				label: m.revoke(),
				destructive: true,
				action: async () => {
					isWorking = true;
					try {
						await userService.revokeRecoveryCodes();
						status = { total: 0, unused: 0 };
						toast.success(m.recovery_codes_revoked());
					} catch (e) {
						axiosErrorToast(e);
					} finally {
						isWorking = false;
					}
				}
			}
		});
	}
</script>

<Item.Root variant="card" class="border-border">
	<Item.Media class="text-primary/80">
		<LucideLifeBuoy class="size-5" />
	</Item.Media>
	<Item.Content class="min-w-52">
		<Item.Title>{m.recovery_codes_title()}</Item.Title>
		<Item.Description>
			{#if hasCodes}
				{m.recovery_codes_status({ unused: status.unused, total: status.total })}
				<br />
				{m.recovery_codes_description()}
			{:else}
				{m.recovery_codes_description()}
			{/if}
		</Item.Description>
	</Item.Content>
	<Item.Actions class="w-full flex-wrap gap-2 sm:w-auto">
		{#if hasCodes}
			<Button
				variant="outline"
				class="w-full sm:w-auto"
				disabled={isWorking}
				onclick={confirmRevoke}
			>
				{m.revoke()}
			</Button>
			<Button
				variant="outline"
				class="w-full sm:w-auto"
				disabled={isWorking}
				onclick={confirmGenerate}
			>
				{m.regenerate()}
			</Button>
		{:else}
			<Button class="w-full sm:w-auto" disabled={isWorking} onclick={confirmGenerate}>
				{m.generate()}
			</Button>
		{/if}
	</Item.Actions>
</Item.Root>

<RecoveryCodesModal bind:show={showCodes} codes={generatedCodes} />
