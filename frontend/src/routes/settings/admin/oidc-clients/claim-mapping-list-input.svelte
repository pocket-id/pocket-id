<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import ClaimMappingInput from './claim-mapping-input.svelte';
	import { m } from '$lib/paraglide/messages';
	import type { OidcClientClaimMapping } from '$lib/types/oidc.type';
	import { LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { z } from 'zod/v4';

	let {
		claimMappings = $bindable([]),
		errors,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		claimMappings: OidcClientClaimMapping[];
		errors?: z.core.$ZodIssue[];
		children?: Snippet;
	} = $props();

	function addClaimMapping() {
		claimMappings = [
			...claimMappings,
			{
				claimName: '',
				sourceType: 'user_field',
				sourceValue: ''
			}
		];
	}

	function removeClaimMapping(index: number) {
		claimMappings = claimMappings.filter((_, i) => i !== index);
	}
</script>

<div {...restProps}>
	<FormInput label={m.claim_mappings()} description={m.claim_mappings_description()}>
		<div class="space-y-4">
			{#each claimMappings as _, i (i)}
				<ClaimMappingInput
					idx={i}
					bind:claimMapping={claimMappings[i]}
					onRemove={removeClaimMapping}
					errors={errors?.filter((e) => e.path[0] === i)}
				/>
			{/each}
		</div>
	</FormInput>

	<Button class="mt-3" variant="secondary" size="sm" onclick={addClaimMapping} type="button">
		<LucidePlus class="mr-1 size-4" />
		{claimMappings.length === 0 ? m.add_claim_mapping() : m.add_another_claim_mapping()}
	</Button>
</div>
