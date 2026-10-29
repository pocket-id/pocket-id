<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import CustomClaimService from '#lib/services/custom-claim-service.ts';
	import type { CustomClaim } from '#lib/types/custom-claim.type.ts';
	import { LucidePlus, LucideX } from '@lucide/svelte';
	import { onMount, tick } from 'svelte';
	import AutoCompleteInput from './auto-complete-input.svelte';

	let { customClaims = $bindable() }: { customClaims: CustomClaim[] } = $props();

	const limit = 20;

	const customClaimService = new CustomClaimService();

	let editor = $state<HTMLElement>();
	let suggestions: string[] = $state([]);
	let filteredSuggestions: string[] = $derived(
		suggestions.filter(
			(suggestion) => !customClaims.some((customClaim) => customClaim.key === suggestion)
		)
	);

	onMount(() => {
		customClaimService.getSuggestions().then((data) => (suggestions = data));
	});

	// The empty state's button disappears once a row exists, so focus moves to the new row's key instead of falling back to the page
	async function add() {
		customClaims = [...customClaims, { key: '', value: '' }];
		await tick();
		const keyInputs = editor?.querySelectorAll<HTMLInputElement>('[data-claim-key] input');
		keyInputs?.item(keyInputs.length - 1).focus();
	}
</script>

<div class="flex flex-col gap-2">
	<div bind:this={editor} class="rounded-md border">
		{#if customClaims.length === 0}
			<div class="text-control text-fg-muted flex min-h-(--row-2line) items-center gap-3 px-4">
				{m.no_custom_claims()}
				<Button variant="ghost" size="sm" onclick={add}>
					<LucidePlus />
					{m.add_custom_claim()}
				</Button>
			</div>
		{:else}
			<div
				aria-hidden="true"
				class="text-caption text-fg-muted grid grid-cols-[1fr_1fr_var(--h-xs)] gap-2 px-3 pt-2.5 font-medium max-sm:hidden"
			>
				<span>{m.key()}</span>
				<span>{m.value()}</span>
			</div>
			{#each customClaims as customClaim, i (customClaim)}
				<div
					class="grid grid-cols-[1fr_var(--h-xs)] items-center gap-2 border-b px-3 py-2 sm:grid-cols-[1fr_1fr_var(--h-xs)]"
				>
					<div data-claim-key class="[&_input]:h-(--h-sm) [&_input]:font-mono">
						<AutoCompleteInput
							placeholder={m.key()}
							suggestions={filteredSuggestions}
							bind:value={customClaim.key}
						/>
					</div>
					<Input
						class="h-(--h-sm) max-sm:col-start-1 max-sm:row-start-2"
						placeholder={m.value()}
						bind:value={customClaim.value}
					/>
					<Button
						variant="ghost"
						size="icon-xs"
						class="max-sm:col-start-2 max-sm:row-start-1"
						aria-label={m.remove_custom_claim()}
						onclick={() => (customClaims = customClaims.filter((_, index) => index !== i))}
					>
						<LucideX />
					</Button>
				</div>
			{/each}
			<div class="px-2 py-1.5">
				<Button variant="ghost" size="sm" disabled={customClaims.length >= limit} onclick={add}>
					<LucidePlus />
					{m.add_another()}
				</Button>
			</div>
		{/if}
	</div>
	<p class="text-caption text-fg-muted">{m.up_to_count_claims({ count: limit })}</p>
</div>
