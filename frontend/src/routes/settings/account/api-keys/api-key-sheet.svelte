<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import userStore from '#lib/stores/user-store.ts';
	import type { ApiKey, ApiKeyResponse } from '#lib/types/api-key.type.ts';
	import { grantableScopes } from '#lib/utils/api-key-scopes.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { emptyToUndefined } from '#lib/utils/zod-util.ts';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import ApiKeyReveal from './api-key-reveal.svelte';
	import ApiKeyScopesInput from './api-key-scopes-input.svelte';

	// Setting apiKey opens the sheet to edit that key, otherwise it creates a new one
	let {
		open = $bindable(),
		apiKey = $bindable(),
		onSaved
	}: { open: boolean; apiKey: ApiKey | null; onSaved: () => unknown } = $props();

	const apiKeyService = new ApiKeyService();
	const formId = $props.id();
	const isAdmin = $derived(!!$userStore?.isAdmin);

	let isLoading = $state(false);
	let created = $state<ApiKeyResponse | null>(null);

	function inThirtyDays() {
		const date = new Date();
		date.setDate(date.getDate() + 30);
		date.setHours(0, 0, 0, 0);
		return date;
	}

	// Editing leaves the expiry alone, because renewing is what changes it
	const formSchema = z.object({
		name: z.string().min(3).max(50),
		description: emptyToUndefined(z.string()),
		expiresAt: z.date().refine((date) => !!apiKey || date > new Date(), {
			message: m.expiration_date_must_be_in_the_future()
		}),
		scopes: z.array(z.string()).min(1, m.choose_at_least_one_permission())
	});
	const formStore = createForm(formSchema, {
		name: '',
		description: '',
		expiresAt: inThirtyDays(),
		scopes: grantableScopes(!!$userStore?.isAdmin)
	});
	const { inputs } = formStore;

	// The values are filled in before the sheet opens, so the permissions picker starts from the key's preset
	$effect(() => {
		const key = apiKey;
		if (!key) return;
		untrack(() => {
			formStore.setValue('name', key.name);
			formStore.setValue('description', key.description ?? '');
			formStore.setValue('scopes', [...key.scopes]);
			open = true;
		});
	});

	function reset() {
		formStore.reset();
		formStore.setValue('expiresAt', inThirtyDays());
		formStore.setValue('scopes', grantableScopes(isAdmin));
		created = null;
		apiKey = null;
	}

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			if (apiKey) {
				const { name, description, scopes } = data;
				await apiKeyService.update(apiKey.id, { name, description, scopes });
				toast.success(m.api_key_updated());
				open = false;
			} else {
				created = await apiKeyService.create(data);
			}
			await onSaved();
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open onOpenChangeComplete={(o) => !o && reset()}>
	<Sheet.Content
		title={created ? m.api_key_created() : apiKey ? m.edit_api_key() : m.create_api_key()}
	>
		{#if created}
			<ApiKeyReveal response={created} />
		{:else}
			<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
				<FormInput label={m.name()} placeholder="Terraform" bind:input={$inputs.name} />
				<FormInput
					label={m.description()}
					optional
					placeholder={m.api_key_description_placeholder()}
					bind:input={$inputs.description}
				/>
				{#if !apiKey}
					<FormInput label={m.expires_on()} type="date" bind:input={$inputs.expiresAt} />
				{/if}
				<ApiKeyScopesInput bind:input={$inputs.scopes} {isAdmin} />
			</form>
		{/if}
		{#snippet footer()}
			{#if created}
				<Button onclick={() => (open = false)}>{m.done()}</Button>
			{:else}
				<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
				<Button type="submit" form={formId} {isLoading}>
					{apiKey ? m.save() : m.create_api_key()}
				</Button>
			{/if}
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
