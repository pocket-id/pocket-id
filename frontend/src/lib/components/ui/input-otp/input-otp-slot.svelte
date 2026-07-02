<script lang="ts">
	import { PinInput as InputOTPPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

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
		'w-10 h-13 rounded-sm bg-input/50 border-input data-active:border-ring data-active:ring-ring/30 data-active:aria-invalid:ring-destructive/20 dark:data-active:aria-invalid:ring-destructive/40 aria-invalid:border-destructive border text-sm transition-all outline-none data-active:ring-3 relative flex items-center justify-center data-active:z-10',
		className
	)}
	{...restProps}
>
	{cell.char}
	{#if cell.hasFakeCaret}
		<div
			class="cn-input-otp-caret pointer-events-none absolute inset-0 flex items-center justify-center"
		>
			<div
				class="animate-caret-blink bg-foreground h-4 w-px duration-1000 bg-foreground h-4 w-px"
			></div>
		</div>
	{/if}
</InputOTPPrimitive.Cell>
