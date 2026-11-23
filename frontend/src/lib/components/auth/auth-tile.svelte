<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import { cn } from '@pocket-id/ui/utils';
	import CheckIcon from '~icons/solar/check-bold';
	import CrossIcon from '~icons/solar/close-bold';

	let { success = false, error = false }: { success?: boolean; error?: boolean } = $props();

	const outcome = $derived(success || error);
</script>

<!-- An outcome ends on the same colored box as the authorize tiles: the color floods out from the middle while the logo gives way to the glyph -->
<!-- Each color has its own layer, so a box that empties again keeps its color until it is gone -->
<div
	class="bg-surface-raised inset-ring-border relative mb-6 grid size-14 shrink-0 place-items-center rounded-md inset-ring"
	aria-hidden="true"
>
	{#each [{ on: success, fill: 'bg-success-soft' }, { on: error, fill: 'bg-danger-soft' }] as layer (layer.fill)}
		<div
			class={cn(
				'absolute inset-0 rounded-[inherit] transition-[clip-path] duration-(--dur-slow) ease-(--ease-out)',
				layer.fill,
				layer.on ? '[clip-path:circle(75%)]' : '[clip-path:circle(0%)]'
			)}
		></div>
	{/each}
	<Logo
		class={cn(
			'transition-motion relative col-start-1 row-start-1 size-8 duration-(--dur-base)',
			outcome && 'scale-60 opacity-0'
		)}
		animate={false}
	/>
	<CheckIcon
		class={cn(
			'text-success-solid transition-motion relative col-start-1 row-start-1 size-7 duration-(--dur-base)',
			success ? 'delay-(--dur-fast)' : 'scale-60 opacity-0'
		)}
	/>
	<CrossIcon
		class={cn(
			'text-danger-solid transition-motion relative col-start-1 row-start-1 size-5.5 duration-(--dur-base)',
			error ? 'delay-(--dur-fast)' : 'scale-60 opacity-0'
		)}
	/>
</div>
