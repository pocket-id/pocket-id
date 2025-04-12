<script lang="ts">
	import QRCode from 'qrcode';
	import { onMount } from 'svelte';

	let { value }: { value: string | null } = $props();

	onMount(() => {
		const canvas = document.getElementById('login-qrcode') as HTMLCanvasElement;
		if (value && canvas) {
			QRCode.toCanvas(canvas, value).catch((error: Error) => {
				console.error('Error generating QR Code:', error);
			});
		}
	});
</script>

<div>
	<canvas id="login-qrcode"></canvas>
</div>

<style>
	div {
		margin: auto;
	}
</style>
