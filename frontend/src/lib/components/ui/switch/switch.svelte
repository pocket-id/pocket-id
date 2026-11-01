<script lang="ts">
	import { Switch as SwitchPrimitive } from 'bits-ui';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		checked = $bindable(false),
		...restProps
	}: WithoutChildrenOrChild<SwitchPrimitive.RootProps> = $props();
</script>

<!-- The on-track and on-thumb colors are layers that crossfade, so only opacity and transform animate -->
<SwitchPrimitive.Root
	bind:ref
	bind:checked
	data-slot="switch"
	class={cn(
		'peer group/switch bg-fill-selected before:bg-primary-control focus-visible:outline-ring relative isolate inline-flex h-6 w-10 shrink-0 items-center rounded-full shadow-[inset_0_0_0_1px_var(--border-control)] before:absolute before:inset-0 before:rounded-full before:opacity-0 before:transition-opacity before:duration-(--dur-base) before:ease-in-out after:absolute after:-inset-x-3 after:-inset-y-2 hover:shadow-[inset_0_0_0_1px_var(--fg-muted)] focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-45 aria-invalid:shadow-[inset_0_0_0_1px_var(--danger-solid)] data-checked:before:opacity-100 data-checked:hover:shadow-[inset_0_0_0_1px_var(--primary-control)]',
		className
	)}
	{...restProps}
>
	<SwitchPrimitive.Thumb
		data-slot="switch-thumb"
		class="bg-fg-muted shadow-thumb before:bg-primary-control-foreground pointer-events-none relative ms-0.5 block size-5 rounded-full transition-transform duration-(--dur-base) ease-in-out before:absolute before:inset-0 before:rounded-full before:opacity-0 before:transition-opacity before:duration-(--dur-base) before:ease-in-out data-checked:translate-x-4 data-checked:before:opacity-100 rtl:data-checked:-translate-x-4"
	/>
</SwitchPrimitive.Root>
