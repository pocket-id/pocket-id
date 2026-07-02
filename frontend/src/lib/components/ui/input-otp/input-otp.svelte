<script lang="ts">
	import { PinInput as InputOTPPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		value = $bindable(''),
		...restProps
	}: InputOTPPrimitive.RootProps = $props();
</script>

<InputOTPPrimitive.Root
	bind:ref
	bind:value
	data-slot="input-otp"
	spellcheck={false}
	data-bwignore="true"
	data-lpignore="true"
	data-1p-ignore="true"
	class={cn(
		'cn-input-otp-input gap-2 flex items-center disabled:cursor-not-allowed has-disabled:opacity-50',
		className
	)}
	{...restProps}
	inputmode="text"
/>
