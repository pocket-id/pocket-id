<script lang="ts">
	import DatePicker from '#lib/components/form/date-picker.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import type { ApiKey, ApiKeyResponse } from '#lib/types/api-key.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import ApiKeyReveal from './api-key-reveal.svelte';

	let { apiKey = $bindable(), onRenewed }: { apiKey: ApiKey | null; onRenewed: () => unknown } =
		$props();

	const apiKeyService = new ApiKeyService();

	const gracePeriods = [
		{ label: m.grace_period_none(), hours: 0 },
		{ label: m.one_hour(), hours: 1 },
		{ label: m.one_day(), hours: 24 },
		{ label: m.one_week(), hours: 24 * 7 },
		{ label: m.one_month(), hours: 24 * 30 }
	];

	let open = $state(false);
	let date = $state(new Date());
	let gracePeriodHours = $state(24);
	let renewed = $state<ApiKeyResponse | null>(null);

	// Renewing a key that still works rotates it, so the old key can stay valid for a grace period
	const rotating = $derived(!!apiKey && new Date(apiKey.expiresAt) > new Date());

	// The new key lasts as long as the old one did
	// It is rounded up to a midnight, because a picked day means the key stops working at the start of that day
	$effect(() => {
		if (!apiKey) return;
		const lifetime = new Date(apiKey.expiresAt).getTime() - new Date(apiKey.createdAt).getTime();
		const next = new Date(Date.now() + lifetime);
		next.setHours(24, 0, 0, 0);
		date = next;
		gracePeriodHours = 24;
		open = true;
	});

	async function renew() {
		try {
			renewed = await apiKeyService.renew(apiKey!.id, date, rotating ? gracePeriodHours : 0);
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
				{#if renewed}
					{rotating ? m.api_key_rotated() : m.api_key_renewed()}
				{:else}
					{(rotating ? m.rotate_name_question : m.renew_name_question)({
						name: apiKey?.name ?? ''
					})}
				{/if}
			</Dialog.Title>
			{#if !renewed}
				<Dialog.Description>
					{rotating ? m.rotate_api_key_consequence() : m.renew_api_key_consequence()}
				</Dialog.Description>
			{:else if renewed.apiKey.previousKeyExpiresAt}
				<Dialog.Description>
					{m.old_key_works_until({
						date: new Date(renewed.apiKey.previousKeyExpiresAt).toLocaleString(getLocale(), {
							dateStyle: 'medium',
							timeStyle: 'short'
						})
					})}
				</Dialog.Description>
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
			</Field.Field>
			{#if rotating}
				<Field.Field>
					<Field.Label for="renew-grace-period">{m.grace_period()}</Field.Label>
					<Select.Root
						type="single"
						value={String(gracePeriodHours)}
						onValueChange={(v) => v && (gracePeriodHours = Number(v))}
					>
						<Select.Trigger id="renew-grace-period" class="max-w-60">
							{gracePeriods.find((p) => p.hours === gracePeriodHours)?.label}
						</Select.Trigger>
						<Select.Content>
							{#each gracePeriods as period (period.hours)}
								<Select.Item value={String(period.hours)}>{period.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
					<Field.Description>{m.api_key_grace_period_helper()}</Field.Description>
				</Field.Field>
			{/if}
			<Dialog.Footer>
				<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button usePromiseLoading onclick={renew}>
					{rotating ? m.rotate_key() : m.renew_key()}
				</Button>
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
