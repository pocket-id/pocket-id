<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import { Command as CommandPrimitive } from 'cmdk-sv';
	import Search from 'lucide-svelte/icons/search';
	import type { ClassValue } from 'svelte/elements';

	type $$Props = CommandPrimitive.InputProps;

	let className: ClassValue | undefined | null = undefined;
	export { className as class };
	export let value: string = '';
</script>

<div class="flex items-center border-b px-2" data-cmdk-input-wrapper="">
	<Search class="mr-2 h-4 w-4 shrink-0 opacity-50" />
	<CommandPrimitive.Input
		class={cn(
			'placeholder:text-muted-foreground flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none disabled:cursor-not-allowed disabled:opacity-50',
			className
		)}
		{...$$restProps}
		bind:value
	/>
</div>
