<script lang="ts">
	import DatePicker from '#lib/components/form/date-picker.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import type { ApiKey, ApiKeyResponse } from '#lib/types/api-key.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import ApiKeyReveal from './api-key-reveal.svelte';

	let { apiKey = $bindable(), onRenewed }: { apiKey: ApiKey | null; onRenewed: () => unknown } =
		$props();

	const apiKeyService = new ApiKeyService();

	let open = $state(false);
	let date = $state(new Date());
	let renewed = $state<ApiKeyResponse | null>(null);

	// The new key lasts as long as the old one did
	// It is rounded up to a midnight, because a picked day means the key stops working at the start of that day
	$effect(() => {
		if (!apiKey) return;
		const lifetime = new Date(apiKey.expiresAt).getTime() - new Date(apiKey.createdAt).getTime();
		const next = new Date(Date.now() + lifetime);
		next.setHours(24, 0, 0, 0);
		date = next;
		open = true;
	});

	async function renew() {
		try {
			renewed = await apiKeyService.renew(apiKey!.id, date);
			await onRenewed();
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function reset() {
		apiKey = null;
		renewed = null;
	}
</script>

<Dialog.Root bind:open onOpenChangeComplete={(o) => !o && reset()}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>
				{renewed ? m.api_key_renewed() : m.renew_name_question({ name: apiKey?.name ?? '' })}
			</Dialog.Title>
			{#if !renewed}
				<Dialog.Description>{m.renew_api_key_consequence()}</Dialog.Description>
			{/if}
		</Dialog.Header>
		{#if renewed}
			<ApiKeyReveal response={renewed} />
			<Dialog.Footer>
				<Button onclick={() => (open = false)}>{m.done()}</Button>
			</Dialog.Footer>
		{:else}
			<Field.Field>
				<Field.Label for="renew-expires-on">{m.expires_on()}</Field.Label>
				<div class="max-w-60"><DatePicker id="renew-expires-on" bind:value={date} /></div>
				<Field.Description>{m.api_key_renew_expires_on_helper()}</Field.Description>
			</Field.Field>
			<Dialog.Footer>
				<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button usePromiseLoading onclick={renew}>{m.renew_key()}</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
