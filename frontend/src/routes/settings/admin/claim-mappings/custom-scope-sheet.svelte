<script lang="ts">
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '@pocket-id/ui/button';
	import * as Sheet from '@pocket-id/ui/sheet';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type { CustomScope } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';

	let {
		open = $bindable(),
		scope,
		onSaved
	}: {
		open: boolean;
		// The scope to edit, or undefined to create a new one
		scope?: CustomScope;
		onSaved: () => unknown;
	} = $props();

	const policyService = new OidcClaimMappingPolicyService();
	const formId = $props.id();

	let isLoading = $state(false);

	const formStore = createForm(
		z.object({
			name: z.string().min(1).max(255),
			// RFC 6749 scope-token characters, which is what the backend accepts
			key: z
				.string()
				.min(1)
				.max(128)
				.regex(/^[\x21\x23-\x5B\x5D-\x7E]+$/, m.scope_key_invalid()),
			description: z.string().min(1).max(1000)
		}),
		{ name: '', key: '', description: '' }
	);
	const { inputs } = formStore;

	$effect(() => {
		if (!open) return;
		untrack(() => {
			formStore.commit({
				name: scope?.name ?? '',
				key: scope?.key ?? '',
				description: scope?.description ?? ''
			});
			formStore.reset();
		});
	});

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const saved = scope
				? await policyService.updateCustomScope(scope.id, data)
				: await policyService.createCustomScope(data);
			open = false;
			toast.success(scope ? m.changes_saved() : m.name_added({ name: saved.name }));
			await onSaved();
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

<Sheet.Root bind:open>
	<Sheet.Content title={scope ? scope.name : m.add_scope()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<FormInput
				label={m.name()}
				placeholder={m.scope_name_placeholder()}
				description={m.scope_name_helper()}
				autocomplete="off"
				bind:input={$inputs.name}
			/>
			<FormInput
				label={m.scope()}
				placeholder={m.scope_key_placeholder()}
				description={m.scope_key_helper()}
				autocomplete="off"
				inputClass="font-mono"
				disabled={!!scope}
				bind:input={$inputs.key}
			/>
			<FormInput
				label={m.description()}
				placeholder={m.scope_description_placeholder()}
				description={m.scope_description_helper()}
				autocomplete="off"
				bind:input={$inputs.description}
			/>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{scope ? m.save() : m.add_scope()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
