<script lang="ts">
	import { replaceState } from '$app/navigation';
	import { page } from '$app/state';
	import { cn } from '$lib/utils/style.js';
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
	$effect(() => {
		if (useHash && page.url.hash) {
			value = page.url.hash.substring(1);
		}
	});

	function onTabChange(newValue: string) {
		if (useHash && page.url.hash.substring(1) !== newValue) {
			replaceState(location.pathname + location.search + `#${newValue}`, page.state);
		}
	}
</script>

<TabsPrimitive.Root
	bind:ref
	bind:value
	onValueChange={onTabChange}
	data-slot="tabs"
	class={cn('gap-2 group/tabs flex data-[orientation=horizontal]:flex-col', className)}
	{...restProps}
/>
