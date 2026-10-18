<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { cn } from '#lib/utils/style.js';
	import { Tabs as TabsPrimitive } from 'bits-ui';

	let {
		ref = $bindable(null),
		value = $bindable(''),
		useHash = false,
		class: className,
		...restProps
	}: TabsPrimitive.RootProps & {
		useHash?: boolean;
	} = $props();

	// Follows the hash on load and on later hash changes, so that links like `#credentials` elsewhere on the page can switch the tab
	// Tab changes write the hash with a shallow goto, so when Back restores such an entry page.url has no hash and the browser's own hash is the fallback
	$effect(() => {
		const hash = page.url.hash || location.hash;
		if (useHash && hash) {
			value = hash.substring(1);
		}
	});

	async function onTabChange(newValue: string) {
		if (useHash && page.url.hash.substring(1) !== newValue) {
			await goto(location.pathname + location.search + `#${newValue}`, {
				shallow: true,
				replace: true,
				state: page.state
			});
		}
	}
</script>

<TabsPrimitive.Root
	bind:ref
	bind:value
	onValueChange={onTabChange}
	data-slot="tabs"
	class={cn('group/tabs flex gap-2 data-[orientation=horizontal]:flex-col', className)}
	{...restProps}
/>
