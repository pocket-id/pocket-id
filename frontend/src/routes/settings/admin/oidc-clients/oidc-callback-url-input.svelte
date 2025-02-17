<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { LucideMinus, LucidePlus } from 'lucide-svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		label,
		callbackURLs = $bindable(),
		error = $bindable(null),
		allowEmpty = false,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		label: string;
		callbackURLs: string[];
		error?: string | null;
		allowEmpty?: boolean;
		children?: Snippet;
	} = $props();

	const limit = 20;
</script>

<div {...restProps}>
	<FormInput {label}>
		<div class="flex flex-col gap-y-2">
			{#each callbackURLs as _, i}
				<div class="flex gap-x-2">
					<Input data-testid={`callback-url-${i + 1}`} bind:value={callbackURLs[i]} />
					{#if callbackURLs.length > 1 || allowEmpty}
						<Button
							variant="outline"
							size="sm"
							on:click={() => (callbackURLs = callbackURLs.filter((_, index) => index !== i))}
						>
							<LucideMinus class="h-4 w-4" />
						</Button>
					{/if}
				</div>
			{/each}
		</div>
	</FormInput>
	{#if error}
		<p class="mt-1 text-sm text-red-500">{error}</p>
	{/if}
	{#if callbackURLs.length < limit}
		<Button
			class="mt-2"
			variant="secondary"
			size="sm"
			on:click={() => (callbackURLs = [...callbackURLs, ''])}
		>
			<LucidePlus class="mr-1 h-4 w-4" />
			{callbackURLs.length === 0 ? 'Add' : 'Add another'}
		</Button>
	{/if}
</div>
