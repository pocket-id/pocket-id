<script lang="ts">
	import QRCode from 'qrcode';
	import { onMount } from 'svelte';

	let {
		value,
		size = 200,
		margin = 0,
		color = '#000000',
		backgroundColor = '#FFFFFF'
	}: {
		value: string | null;
		size?: number;
		margin?: number;
		color?: string;
		backgroundColor?: string;
	} = $props();

	onMount(() => {
		const canvas = document.getElementById('qr-code-canvas') as HTMLCanvasElement;
		if (value && canvas) {
			// Convert "transparent" to a valid value for the QR code library
			const lightColor = backgroundColor === 'transparent' ? '#00000000' : backgroundColor;

			const options = {
				width: size,
				margin: margin,
				color: {
					dark: color,
					light: lightColor
				}
			};

			QRCode.toCanvas(canvas, value, options).catch((error: Error) => {
				console.error('Error generating QR Code:', error);
			});
		}
	});
</script>

<div
	class="qrcode-container"
	style="--bg-color: {backgroundColor === 'transparent' ? 'transparent' : backgroundColor};"
>
	<canvas id="qr-code-canvas" class="rounded-lg"></canvas>
</div>

<style>
	.qrcode-container {
		display: flex;
		justify-content: center;
		align-items: center;
		padding: 0.75rem;
		border-radius: 1rem;
		background: var(--background, transparent);
		border: 1px solid var(--border, rgba(0, 0, 0, 0.1));
		margin: 0.5rem 0;
	}
</style>
