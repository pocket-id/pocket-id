<script lang="ts">
	interface Props {
		[key: string]: any
	}

	let { ...rest }: Props = $props();
</script>

<svg
	class="checkmark block stroke-green-600 stroke-[5px] {rest.class}"
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 52 52"
>
	<path class="checkmark__check" fill="none" d="M5,26 L20,41 L47,14" />
</svg>

<style>
	.checkmark {
		stroke-miterlimit: 10;
		stroke-linecap: round;
	}

	.checkmark__check {
		transform-origin: 50% 50%;
		stroke-dasharray: 72;
		stroke-dashoffset: 72;
		animation: stroke 0.3s cubic-bezier(0.65, 0, 0.45, 1) 0.2s forwards;
	}

	@keyframes stroke {
		100% {
			stroke-dashoffset: 0;
		}
	}
</style>
