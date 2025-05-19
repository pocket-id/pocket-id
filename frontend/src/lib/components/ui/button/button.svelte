<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import { Button as ButtonPrimitive } from 'bits-ui';
	import LoaderCircle from 'lucide-svelte/icons/loader-circle';
	import type { ClassNameValue } from 'tailwind-merge';
	import { type Events, type Props, buttonVariants } from './index.js';

	type $$Props = Props;
	type $$Events = Events;

	interface Props_1 {
		class?: $$Props['class'];
		variant?: $$Props['variant'];
		size?: $$Props['size'];
		disabled?: boolean | undefined | null;
		isLoading?: $$Props['isLoading'];
		builders?: $$Props['builders'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let {
		class: className = undefined,
		variant = 'default',
		size = 'default',
		disabled = false,
		isLoading = false,
		builders = [],
		children,
		...rest
	}: Props_1 = $props();
	
</script>

<ButtonPrimitive.Root
	{builders}
	disabled={isLoading || disabled}
	class={cn(buttonVariants({ variant, size, className: className as ClassNameValue }))}
	type="button"
	{...rest}
	on:click
	on:keydown
>
	{#if isLoading}
		<LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
	{/if}
	{@render children?.()}
</ButtonPrimitive.Root>
