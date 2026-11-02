<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import { Command as CommandPrimitive } from 'bits-ui';

	export type CommandRootApi = CommandPrimitive.Root;

	let {
		api = $bindable(null),
		ref = $bindable(null),
		value = $bindable(''),
		// Inside a popover the list mounts before it is positioned, so scrolling the first item into view would scroll the page
		disableInitialScroll = true,
		class: className,
		...restProps
	}: CommandPrimitive.RootProps & {
		api?: CommandRootApi | null;
	} = $props();
</script>

<CommandPrimitive.Root
	bind:this={api}
	bind:value
	bind:ref
	{disableInitialScroll}
	data-slot="command"
	class={cn('text-fg flex size-full flex-col overflow-hidden p-1', className)}
	{...restProps}
/>
