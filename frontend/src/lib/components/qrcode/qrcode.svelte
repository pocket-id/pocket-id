<script lang="ts">
	import { cn } from '#lib/utils/style.ts';
	import QRCode from 'qrcode';
	import type { HTMLAttributes } from 'svelte/elements';

	let canvasEl: HTMLCanvasElement | null;
	let {
		value,
		size = 200,
		...restProps
	}: HTMLAttributes<HTMLCanvasElement> & {
		value: string | null;
		size?: number;
	} = $props();

	$effect(() => {
		if (value && canvasEl) {
			QRCode.toCanvas(canvasEl, value, {
				width: size,
				margin: 0,
				color: { dark: '#121212', light: '#ffffff' }
			})
				.then(() => {
					canvasEl?.style.removeProperty('height');
					canvasEl?.style.removeProperty('width');
				})
				.catch((error: Error) => {
					console.error('Error generating QR Code:', error);
				});
		}
	});
</script>

<!-- Scanners need dark modules on a light quiet zone, so the code sits on a white tile in both themes -->
<canvas
	{...restProps}
	bind:this={canvasEl}
	class={cn('bg-qr-bg box-content rounded-md p-2', restProps.class)}
></canvas>
