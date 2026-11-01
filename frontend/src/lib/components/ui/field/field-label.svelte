<script lang="ts">
	import { Label } from '#lib/components/ui/label/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		optional = false,
		children,
		...restProps
	}: ComponentProps<typeof Label> & {
		optional?: boolean;
	} = $props();
</script>

<Label
	bind:ref
	data-slot="field-label"
	class={cn(
		'group/field-label peer/field-label flex w-fit gap-2 font-semibold group-data-[disabled=true]/field:opacity-45',
		'has-[>[data-slot=field]]:bg-surface hover:has-[>[data-slot=field]]:bg-fill-row-hover has-data-checked:border-border-selected has-[>[data-slot=field]]:has-data-checked:bg-surface-raised hover:has-[>[data-slot=field]]:has-data-checked:bg-surface-raised has-data-checked:shadow-selected has-[>[data-slot=field]]:w-full has-[>[data-slot=field]]:flex-col has-[>[data-slot=field]]:rounded-md has-[>[data-slot=field]]:border has-[>[data-slot=field]]:font-medium has-[>[data-slot=field]]:has-disabled:cursor-not-allowed *:data-[slot=field]:px-4 *:data-[slot=field]:py-3 has-disabled:**:data-[slot=field-content]:opacity-45',
		className
	)}
	{...restProps}
>
	{@render children?.()}
	{#if optional}
		<span class="text-fg-muted -ms-1 font-normal">{m.field_optional()}</span>
	{/if}
</Label>
