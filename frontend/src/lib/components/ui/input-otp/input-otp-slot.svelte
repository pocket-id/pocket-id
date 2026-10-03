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
		'bg-input/50 border-input data-active:border-ring data-active:ring-ring/30 data-active:aria-invalid:ring-destructive/20 dark:data-active:aria-invalid:ring-destructive/40 aria-invalid:border-destructive relative flex h-13 w-10 items-center justify-center rounded-sm border text-sm transition-all outline-none data-active:z-10 data-active:ring-3',
		className
	)}
	{...restProps}
>
	{cell.char}
	{#if cell.hasFakeCaret}
		<div
			class="cn-input-otp-caret pointer-events-none absolute inset-0 flex items-center justify-center"
		>
			<div class="animate-caret-blink bg-foreground bg-foreground h-4 w-px duration-1000"></div>
		</div>
	{/if}
</InputOTPPrimitive.Cell>
