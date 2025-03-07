<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import { Button as ButtonPrimitive } from 'bits-ui';
	import LoaderCircle from 'lucide-svelte/icons/loader-circle';
	import type { ClassNameValue } from 'tailwind-merge';
	import { type Events, type Props, buttonVariants } from './index.js';

	type $$Props = Props;
	type $$Events = Events;

	let className: $$Props['class'] = undefined;
	export let variant: $$Props['variant'] = 'default';
	export let size: $$Props['size'] = 'default';
	export let disabled: boolean | undefined | null = false;
	export let isLoading: $$Props['isLoading'] = false;
	export let builders: $$Props['builders'] = [];
	export { className as class };
</script>

<ButtonPrimitive.Root
	{builders}
	disabled={isLoading || disabled}
	class={cn(buttonVariants({ variant, size, className: className as ClassNameValue }))}
	type="button"
	{...$$restProps}
	on:click
	on:keydown
>
	{#if isLoading}
		<LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
	{/if}
	<slot />
</ButtonPrimitive.Root>
