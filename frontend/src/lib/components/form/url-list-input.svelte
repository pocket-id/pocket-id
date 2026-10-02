<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import { Input } from '#lib/components/ui/input/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucidePlus, LucideX } from '@lucide/svelte';

	let {
		urls = $bindable(),
		error = null,
		testIdPrefix = 'url',
		disabled = false,
		keepAtLeastOne = false,
		addLabel
	}: {
		urls: string[];
		error?: string | null;
		testIdPrefix?: string;
		disabled?: boolean;
		keepAtLeastOne?: boolean;
		// Names what the button adds, instead of the generic "Add" and "Add another"
		addLabel?: string;
	} = $props();

	function removeUrl(index: number) {
		if (keepAtLeastOne && urls.length === 1) {
			urls = [''];
			return;
		}

		urls = urls.filter((_, urlIndex) => urlIndex !== index);
	}
</script>

<div>
	<div class="flex flex-col gap-y-2">
		{#each urls as url, i (i)}
			<div class="flex gap-x-2">
				<Input
					class="font-mono text-[13px] md:text-[13px]"
					aria-invalid={!!error}
					data-testid={`${testIdPrefix}-${i + 1}`}
					type="text"
					inputmode="url"
					autocomplete="url"
					bind:value={urls[i]}
					{disabled}
				/>
				<Button
					variant="ghost"
					size="icon"
					class="text-muted-foreground size-9"
					aria-label={m.remove_url({ identifier: url || i + 1 })}
					onclick={() => removeUrl(i)}
					{disabled}
				>
					<LucideX class="size-4" />
				</Button>
			</div>
		{/each}
	</div>
	<Button
		class="text-muted-foreground mt-1 -ml-2.5"
		variant="ghost"
		size="sm"
		onclick={() => (urls = [...urls, ''])}
		{disabled}
	>
		<LucidePlus class="mr-1 size-4" />
		{addLabel ?? (urls.length === 0 ? m.add() : m.add_another())}
	</Button>
</div>
