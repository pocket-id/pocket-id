<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import CustomClaimService from '$lib/services/custom-claim-service';
	import type { CustomClaim } from '$lib/types/custom-claim.type';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import { onMount, type Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import AutoCompleteInput from './auto-complete-input.svelte';
	import { m } from '$lib/paraglide/messages';

	let {
		customClaims = $bindable(),
		error = $bindable(null),
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		customClaims: CustomClaim[];
		error?: string | null;
		children?: Snippet;
	} = $props();

	const limit = 20;

	const customClaimService = new CustomClaimService();

	let suggestions: string[] = $state([]);
	let filteredSuggestions: string[] = $derived(
		suggestions.filter(
			(suggestion) => !customClaims.some((customClaim) => customClaim.key === suggestion)
		)
	);

	onMount(() => {
		customClaimService.getSuggestions().then((data) => (suggestions = data));
	});
</script>

<div {...restProps}>
	<FormInput>
		<div class="flex flex-col gap-y-2">
			{#each customClaims as _, i}
				<div class="flex gap-x-2">
					<AutoCompleteInput
						placeholder={m.key()}
						suggestions={filteredSuggestions}
						bind:value={customClaims[i].key}
					/>
					<Input placeholder={m.value()} bind:value={customClaims[i].value} />
					<Button
						variant="outline"
						size="sm"
						aria-label={m.remove_custom_claim()}
						onclick={() => (customClaims = customClaims.filter((_, index) => index !== i))}
					>
						<LucideMinus class="size-4" />
					</Button>
				</div>
			{/each}
		</div>
	</FormInput>
	{#if error}
		<p class="mt-1 text-sm text-red-500">{error}</p>
	{/if}
	{#if customClaims.length < limit}
		<Button
			class="mt-2"
			variant="secondary"
			size="sm"
			onclick={() => (customClaims = [...customClaims, { key: '', value: '' }])}
		>
			<LucidePlus class="mr-1 size-4" />
			{customClaims.length === 0 ? m.add_custom_claim() : m.add_another()}
		</Button>
	{/if}
</div>
