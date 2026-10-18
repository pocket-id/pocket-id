<script lang="ts">
	import { PinInput as InputOTPPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		cell,
		class: className,
		...restProps
	}: InputOTPPrimitive.CellProps = $props();
</script>

<InputOTPPrimitive.Cell
	{cell}
	bind:ref
	data-slot="input-otp-slot"
	class={cn(
		'text-code-cell text-fg before:bg-danger-soft aria-invalid:text-danger-fg relative isolate grid h-18 max-w-14 min-w-[30px] flex-1 place-items-center font-mono uppercase tabular-nums before:absolute before:inset-0 before:-z-1 before:opacity-0 before:transition-opacity before:duration-(--dur-fast) before:ease-out after:pointer-events-none after:absolute after:inset-0 after:opacity-0 after:shadow-[inset_0_0_0_1px_var(--ring),inset_0_0_0_calc(1px+var(--halo-width))_var(--ring-halo)] first:after:rounded-l-sm last:after:rounded-r-sm aria-invalid:before:opacity-100 data-active:z-2 data-active:after:opacity-100 [&+&]:shadow-[inset_1px_0_0_var(--border-strong)]',
		className
	)}
	{...restProps}
>
	{cell.char}
	{#if cell.hasFakeCaret}
		<div class="pointer-events-none absolute inset-0 flex items-center justify-center">
			<div class="animate-caret-blink bg-fg h-5.5 w-px [animation-duration:var(--t-caret)]"></div>
		</div>
	{/if}
</InputOTPPrimitive.Cell>
