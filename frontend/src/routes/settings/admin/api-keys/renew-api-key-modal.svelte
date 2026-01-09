<script lang="ts">
	import DatePicker from '$lib/components/form/date-picker.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { ApiKey } from '$lib/types/api-key.type';

	let {
		apiKey = $bindable(null),
		onRenew
	}: {
		apiKey: ApiKey | null;
		onRenew: (date: Date) => Promise<void>;
	} = $props();

	let date = $state(new Date());

	$effect(() => {
		if (apiKey) {
			const lastExpirationDuration =
				new Date(apiKey.expiresAt).getTime() - new Date(apiKey.createdAt).getTime();
			date = new Date(Date.now() + lastExpirationDuration);
		}
	});

	function onOpenChange(open: boolean) {
		if (!open) {
			apiKey = null;
		}
	}
</script>

<Dialog.Root open={!!apiKey} {onOpenChange}>
	<Dialog.Content class="max-w-md" onOpenAutoFocus={(e) => e.preventDefault()}>
		<Dialog.Header>
			<Dialog.Title>{m.renew_api_key()}</Dialog.Title>
			<Dialog.Description>
				{m.renew_api_key_description()}
			</Dialog.Description>
		</Dialog.Header>
		<Field.Field>
			<Field.Label>{m.expiration()}</Field.Label>
			<DatePicker bind:value={date} />
		</Field.Field>

		<Dialog.Footer class="mt-3">
			<Button variant="outline" onclick={() => onOpenChange(false)}>{m.cancel()}</Button>
			<Button variant="default" usePromiseLoading onclick={() => onRenew(date)}>{m.renew()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
