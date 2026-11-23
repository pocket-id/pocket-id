<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as Tabs from '@pocket-id/ui/tabs';
	import type { ComponentProps } from 'svelte';

	let { value = $bindable(''), ...restProps }: ComponentProps<typeof Tabs.Root> = $props();

	// Follows the hash on load and on later hash changes, so that links like `#credentials` elsewhere on the page can switch the tab
	// Tab changes write the hash with a shallow goto, so when Back restores such an entry page.url has no hash and the browser's own hash is the fallback
	$effect(() => {
		const hash = page.url.hash || location.hash;
		if (hash) {
			value = hash.substring(1);
		}
	});

	async function onTabChange(newValue: string) {
		if (page.url.hash.substring(1) !== newValue) {
			await goto(location.pathname + location.search + `#${newValue}`, {
				shallow: true,
				replace: true,
				state: page.state
			});
		}
	}
</script>

<Tabs.Root bind:value onValueChange={onTabChange} {...restProps} />
