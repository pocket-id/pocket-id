<script lang="ts">
	import Logo from '#lib/components/logo.svelte';
	import { cn } from '@pocket-id/ui/utils';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cachedLogoMask, loadLogoMask, renderQr } from './qr-render.ts';

	let {
		value,
		size = 200,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		value: string | null;
		size?: number;
	} = $props();

	let mask = $state(cachedLogoMask());
	$effect(() => {
		if (!mask) void loadLogoMask().then((m) => (mask = m));
	});

	const drawing = $derived(value ? renderQr(value, mask) : undefined);
	const pct = (v: number) => `${(v / drawing!.n) * 100}%`;
</script>

<!-- Scanners need dark modules on a light quiet zone, so the code sits on a white tile in both themes -->
<div
	{...restProps}
	class={cn(
		'bg-qr-bg box-content aspect-square w-(--qr-size) rounded-md p-2 text-[#121212]',
		restProps.class
	)}
	style:--qr-size="{size}px"
>
	{#if drawing}
		<div class="relative size-full">
			<svg viewBox="0 0 {drawing.n} {drawing.n}" class="block size-full" aria-hidden="true">
				<path fill="currentColor" d={drawing.modules} />
				<path fill="currentColor" fill-rule="evenodd" d={drawing.eyes} />
			</svg>
			<div
				class="absolute"
				style:left={pct(drawing.logo.start)}
				style:top={pct(drawing.logo.start)}
				style:width={pct(drawing.logo.size)}
				style:height={pct(drawing.logo.size)}
			>
				<Logo colorScheme="light" animate={false} class="size-full" alt="" />
			</div>
		</div>
	{/if}
</div>
