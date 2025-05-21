<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';
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
</script>

<div {...restProps}>
	<FormInput {label} description={m.callback_url_description()}>
		<div class="flex flex-col gap-y-2">
			{#each callbackURLs as _, i}
				<div class="flex gap-x-2">
					<Input
						aria-invalid={!!error}
						data-testid={`callback-url-${i + 1}`}
						bind:value={callbackURLs[i]}
					/>
					{#if callbackURLs.length > 1 || allowEmpty}
						<Button
							variant="outline"
							size="sm"
							onclick={() => (callbackURLs = callbackURLs.filter((_, index) => index !== i))}
						>
							<LucideMinus class="size-4" />
						</Button>
					{/if}
				</div>
			{/each}
		</div>
	</FormInput>
	{#if error}
		<p class="text-destructive mt-1 text-xs">{error}</p>
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
