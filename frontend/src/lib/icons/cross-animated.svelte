<script lang="ts">
	interface Props {
		[key: string]: any;
	}

	let { ...rest }: Props = $props();
</script>

<svg
	class="stroke-red-600 stroke-[13px] {rest.class}"
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 85 85"
>
	<path class="line1" d="M5,5 L80,80"></path>
	<path class="line2" d="M80,5 L5,80"></path>
</svg>

<style>
	.line1 {
		stroke-dasharray: 107px;
		stroke-dashoffset: 107px;
		stroke-linecap: round;
		animation: ani-error-line 0.15s 0.3s linear both;
	}
	.line2 {
		stroke-dasharray: 107px;
		stroke-dashoffset: 107px;
		stroke-linecap: round;
		animation: ani-error-line 0.2s 0.6s linear both;
	}

	@keyframes ani-error-line {
		to {
			stroke-dashoffset: 0;
		}
	}
</style>
