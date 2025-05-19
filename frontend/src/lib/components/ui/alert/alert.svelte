<script lang="ts">
	import { cn } from '$lib/utils/style.js';
	import { LucideX } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { type Variant, alertVariants } from './index.js';

	type $$Props = HTMLAttributes<HTMLDivElement> & {
		variant?: Variant;
		dismissibleId?: string;
	};

	interface Props {
		class?: $$Props['class'];
		variant?: $$Props['variant'];
		dismissibleId?: $$Props['dismissibleId'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let {
		class: className = undefined,
		variant = 'default',
		dismissibleId = undefined,
		children,
		...rest
	}: Props = $props();
	

	let isVisible = $state(!dismissibleId);

	onMount(() => {
		if (dismissibleId) {
			const dismissedAlerts = JSON.parse(localStorage.getItem('dismissed-alerts') || '[]');
			isVisible = !dismissedAlerts.includes(dismissibleId);
		}
	});

	function dismiss() {
		if (dismissibleId) {
			const dismissedAlerts = JSON.parse(localStorage.getItem('dismissed-alerts') || '[]');
			localStorage.setItem('dismissed-alerts', JSON.stringify([...dismissedAlerts, dismissibleId]));
			isVisible = false;
		}
	}
</script>

{#if isVisible}
	<div class={cn(alertVariants({ variant }), className)} {...rest} role="alert">
		{@render children?.()}
		{#if dismissibleId}
			<button onclick={dismiss} class="absolute top-0 right-0 m-3 text-black dark:text-white"
				><LucideX class="w-4" /></button
			>
		{/if}
	</div>
{/if}
