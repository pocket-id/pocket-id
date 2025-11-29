<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		label,
		description,
		callbackURLs = $bindable(),
		error = $bindable(null),
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		label: string;
		description: string;
		callbackURLs: string[];
		error?: string | null;
		children?: Snippet;
	} = $props();
</script>

<div {...restProps}>
	<FormInput {label} {description}>
		<div class="flex flex-col gap-y-2">
			{#each callbackURLs as _, i}
				<div class="flex gap-x-2">
					<Input
						aria-invalid={!!error}
						data-testid={`callback-url-${i + 1}`}
						bind:value={callbackURLs[i]}
					/>
					<Button
						variant="outline"
						size="sm"
						onclick={() => (callbackURLs = callbackURLs.filter((_, index) => index !== i))}
					>
						<LucideMinus class="size-4" />
					</Button>
				</div>
			{/each}
		</div>
	</FormInput>
	{#if error}
		<Field.Error>{error}</Field.Error>
	{/if}
	<Button
		class="mt-2"
		variant="secondary"
		size="sm"
		onclick={() => (callbackURLs = [...callbackURLs, ''])}
	>
		<LucidePlus class="mr-1 size-4" />
		{callbackURLs.length === 0 ? m.add() : m.add_another()}
	</Button>
</div>
