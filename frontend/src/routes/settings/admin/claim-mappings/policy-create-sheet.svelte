<script lang="ts">
	import { goto } from '$app/navigation';
	import FormInput from '#lib/components/form/form-input.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Field from '#lib/components/ui/field/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import {
		builtInClaimMappingPolicyId,
		type OidcClaimMappingPolicyMetadataDto
	} from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { preventDefault } from '#lib/utils/event-util.ts';
	import { createForm } from '#lib/utils/form-util.ts';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { z } from 'zod/v4';
	import { claimsCount, policyHref } from './claim-mapping-util';

	let { open = $bindable() }: { open: boolean } = $props();

	const policyService = new OidcClaimMappingPolicyService();
	const formId = $props.id();

	let isLoading = $state(false);
	let policies = $state<OidcClaimMappingPolicyMetadataDto[]>([]);
	// A copy of the built-in policy by default, so a new policy starts from what Pocket ID has always sent
	let from = $state(builtInClaimMappingPolicyId);

	const formStore = createForm(z.object({ name: z.string().min(1).max(255) }), { name: '' });
	const { inputs } = formStore;

	const fromLabel = $derived(policies.find((p) => p.id === from)?.name ?? m.no_claims());

	$effect(() => {
		if (!open) return;
		untrack(() => {
			formStore.reset();
			from = builtInClaimMappingPolicyId;
			policyService
				.listClaimMappingPolicies({
					pagination: { page: 1, limit: 100 },
					sort: { column: 'name', direction: 'asc' }
				})
				.then((res) => (policies = res.data))
				.catch(axiosErrorToast);
		});
	});

	async function onSubmit() {
		const data = formStore.validate();
		if (!data) return;
		isLoading = true;
		try {
			const claimMappings = from
				? (await policyService.getClaimMappingPolicy(from)).claimMappings
				: [];
			const policy = await policyService.createClaimMappingPolicy({
				name: data.name,
				isDefault: false,
				claimMappings
			});
			open = false;
			toast.success(m.name_added({ name: policy.name }));
			goto(policyHref(policy.id));
		} catch (e) {
			axiosErrorToast(e);
		} finally {
			isLoading = false;
		}
	}
</script>

{#snippet option(label: string, description: string)}
	<span class="flex min-w-0 flex-col">
		<span class="truncate">{label}</span>
		<span class="text-caption text-fg-muted truncate">{description}</span>
	</span>
{/snippet}

<Sheet.Root bind:open>
	<Sheet.Content title={m.add_policy()}>
		<form id={formId} onsubmit={preventDefault(onSubmit)} class="flex flex-col gap-6">
			<FormInput
				label={m.name()}
				placeholder={m.policy_name_placeholder()}
				description={m.policy_name_helper()}
				autocomplete="off"
				bind:input={$inputs.name}
			/>
			<Field.Field>
				<Field.Label for="policy-start-from">{m.start_from()}</Field.Label>
				<Select.Root type="single" bind:value={from}>
					<Select.Trigger id="policy-start-from" class="w-full">{fromLabel}</Select.Trigger>
					<Select.Content>
						<Select.Item value="" label={m.no_claims()}>
							{@render option(m.no_claims(), m.an_empty_policy())}
						</Select.Item>
						{#each policies as policy (policy.id)}
							<Select.Item value={policy.id} label={policy.name}>
								{@render option(policy.name, claimsCount(policy.claimCount))}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
				<Field.Description>{m.start_from_helper()}</Field.Description>
			</Field.Field>
		</form>
		{#snippet footer()}
			<Button variant="ghost" onclick={() => (open = false)}>{m.cancel()}</Button>
			<Button type="submit" form={formId} {isLoading}>{m.add_policy()}</Button>
		{/snippet}
	</Sheet.Content>
</Sheet.Root>
