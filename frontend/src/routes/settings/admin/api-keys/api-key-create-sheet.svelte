<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import type { ApiKeyResponse } from '#lib/types/api-key.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { emptyToUndefined } from '#lib/utils/zod-util.ts';
	import { z } from 'zod/v4';
	import ApiKeyReveal from './api-key-reveal.svelte';

	let { open = $bindable(), onCreated }: { open: boolean; onCreated: () => unknown } = $props();

	const apiKeyService = new ApiKeyService();
	const formId = $props.id();

	let isLoading = $state(false);
	let created = $state<ApiKeyResponse | null>(null);

	function inThirtyDays() {
		const date = new Date();
		date.setDate(date.getDate() + 30);
		date.setHours(0, 0, 0, 0);
		return date;
	}

	const formSchema = z.object({
		name: z.string().min(3).max(50),
		description: emptyToUndefined(z.string()),
		expiresAt: z.date().min(new Date(), m.expiration_date_must_be_in_the_future())
	});
	const formStore = createForm(formSchema, {
		name: '',
		description: '',
		expiresAt: inThirtyDays()
	});
	const { inputs } = formStore;

	function reset() {
		formStore.reset();
		formStore.setValue('expiresAt', inThirtyDays());
		created = null;
	}

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			created = await apiKeyService.create(data);
			await onCreated();
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open onOpenChangeComplete={(o) => !o && reset()}>
	<Sheet.Content title={created ? m.api_key_created() : m.create_api_key()}>
		{#if created}
			<ApiKeyReveal response={created} />
		{:else}
			<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
				<FormInput
					label={m.name()}
					placeholder="Terraform"
					description={m.api_key_name_helper()}
					bind:input={$inputs.name}
				/>
				<FormInput
					label={m.description()}
					optional
					placeholder={m.api_key_description_placeholder()}
					bind:input={$inputs.description}
				/>
				<FormInput
					label={m.expires_on()}
					description={m.api_key_expires_on_helper()}
					type="date"
					bind:input={$inputs.expiresAt}
				/>
			</form>
		{/if}
		{#snippet footer()}
			{#if created}
				<Button onclick={() => (open = false)}>{m.done()}</Button>
			{:else}
				<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button type="submit" form={formId} {isLoading}>{m.create_api_key()}</Button>
			{/if}
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
