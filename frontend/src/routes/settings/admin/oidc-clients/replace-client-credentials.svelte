<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import OidcService from '$lib/services/oidc-service';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClient, OidcClientMetaData } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { toast } from 'svelte-sonner';
	import type { HTMLAttributes } from 'svelte/elements';
	import { TextCursorInput } from '@lucide/svelte';

	let {
		client,
		onRefresh,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		client?: OidcClient;
		onRefresh?: (newId?: string) => void;
	} = $props();

	let newClientIdInput = $state<OidcClientMetaData['id']>('');
	let expandUpdateClientIdentifiers = $state(false);

	async function handleUpdateClientId() {
		if (!client) return;
		try {
			const oidcService = new OidcService();
			await oidcService.updateClientId(client.id, newClientIdInput);	
			toast.success('Client ID updated successfully');
			onRefresh?.(newClientIdInput);			
		} catch (e) {
			axiosErrorToast(e);
		}
	}
	
</script>

<div {...restProps}>
	<Button class="mt-0" variant="secondary" size="sm" onclick={() => (expandUpdateClientIdentifiers = !expandUpdateClientIdentifiers)} type="button">
	<TextCursorInput class="mr-1 size-4" />
		{m.update()} {m.client_id()}
	</Button>

	{#if expandUpdateClientIdentifiers}
		<div class="mt-4 space-y-3 rounded-lg border p-4">
			<div class="flex flex-col sm:flex-row items-center justify-between gap-4">
				<Input
					id="newClientIdInput"
					bind:value={newClientIdInput}
					placeholder={client?.id ?? ''}
					class="flex-grow"
				/>
				<Button class="mt-0 whitespace-nowrap" variant="secondary" onclick={handleUpdateClientId}>{m.update()} {m.client_id()}</Button>
			</div>
		</div>
	{/if}
</div>
