<script lang="ts">
	import { page } from '$app/state';
	import { cn } from '$lib/utils/style.js';
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { onMount } from 'svelte';

	let {
		ref = $bindable(null),
		value = $bindable(''),
		useHash = false,
		class: className,
		...restProps
	}: TabsPrimitive.RootProps & {
		useHash?: boolean;
	} = $props();

	onMount(() => {
		if (useHash && page.url.hash) {
			value = page.url.hash.substring(1);
		}
	});

	function onTabChange(newValue: string) {
		if (useHash && page.url.hash !== newValue) {
			window.location.hash = newValue;
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
