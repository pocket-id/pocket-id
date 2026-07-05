<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import { LucideMinus, LucidePlus } from '@lucide/svelte';

	let {
		urls = $bindable(),
		error = null,
		testIdPrefix = 'url'
	}: {
		urls: string[];
		error?: string | null;
		testIdPrefix?: string;
	} = $props();
</script>

<div>
	<div class="flex flex-col gap-y-2">
		{#each urls as _, i}
			<div class="flex gap-x-2">
				<Input
					aria-invalid={!!error}
					data-testid={`${testIdPrefix}-${i + 1}`}
					type="text"
					inputmode="url"
					autocomplete="url"
					bind:value={urls[i]}
				/>
				<Button
					variant="outline"
					size="sm"
					onclick={() => (urls = urls.filter((_, index) => index !== i))}
				>
					<LucideMinus class="size-4" />
				</Button>
			</div>
		{/each}
	</div>
	<Button class="mt-2" variant="secondary" size="sm" onclick={() => (urls = [...urls, ''])}>
		<LucidePlus class="mr-1 size-4" />
		{urls.length === 0 ? m.add() : m.add_another()}
	</Button>
</div>
