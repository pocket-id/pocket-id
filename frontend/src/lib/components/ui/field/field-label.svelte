<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import { cn } from '$lib/utils/style.js';
	import type { ComponentProps } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		required = false,
		children,
		...restProps
	}: ComponentProps<typeof Label> & {
		required?: boolean;
	} = $props();
</script>

<Label
	bind:ref
	data-slot="field-label"
	class={cn(
		'has-data-checked:bg-input/30 gap-2 leading-snug group-data-[disabled=true]/field:opacity-50 has-[>[data-slot=field]]:rounded-2xl has-[>[data-slot=field]]:border *:data-[slot=field]:p-4 group/field-label peer/field-label flex w-fit leading-snug',
		'has-[>[data-slot=field]]:w-full has-[>[data-slot=field]]:flex-col',
		required && "after:text-red-500 after:content-['*']",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</Label>
