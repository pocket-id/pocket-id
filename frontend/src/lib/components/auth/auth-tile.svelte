<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import { cn } from '#lib/utils/style.ts';
	import { LucideCheck } from '@lucide/svelte';

	let { success = false }: { success?: boolean } = $props();
</script>

<!-- The tile holds still while only its logo crossfades to the check, so the surface never dips -->
<div
	class="bg-surface-raised inset-ring-border mb-6 grid size-14 shrink-0 place-items-center rounded-md inset-ring"
	aria-hidden="true"
>
	<Logo
		class={cn(
			'col-start-1 row-start-1 size-8 transition-opacity duration-(--dur-base) ease-out',
			success && 'opacity-0'
		)}
	/>
	<LucideCheck
		class={cn(
			'text-success-solid transition-motion col-start-1 row-start-1 size-5 duration-(--dur-base)',
			!success && 'scale-60 opacity-0'
		)}
	/>
</div>
