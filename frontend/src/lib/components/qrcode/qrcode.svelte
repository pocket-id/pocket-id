<script lang="ts">
	import { cn } from '$lib/utils/style';
	import QRCode from 'qrcode';
	import { onMount } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let canvasEl: HTMLCanvasElement | null;
	let {
		value,
		size = 200,
		color = '#000000',
		backgroundColor = '#FFFFFF',
		...restProps
	}: HTMLAttributes<HTMLCanvasElement> & {
		value: string | null;
		size?: number;
		color?: string;
		backgroundColor?: string;
	} = $props();

	onMount(() => {
		if (value && canvasEl) {
			// Convert "transparent" to a valid value for the QR code library
			const lightColor = backgroundColor === 'transparent' ? '#00000000' : backgroundColor;

			const options = {
				width: size,
				margin: 0,
				color: {
					dark: color,
					light: lightColor
				}
			};

			QRCode.toCanvas(canvasEl, value, options).catch((error: Error) => {
				console.error('Error generating QR Code:', error);
			});
		}
	});
</script>

<canvas {...restProps} bind:this={canvasEl} class={cn('rounded-lg', restProps.class)}></canvas>
