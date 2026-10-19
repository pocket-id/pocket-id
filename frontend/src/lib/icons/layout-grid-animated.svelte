<script lang="ts">
	import type { IconProps } from '@jis3r/icons';

	let { size = 24, strokeWidth = 2, animate = false }: IconProps = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width={strokeWidth}
	stroke-linecap="round"
	stroke-linejoin="round"
	class:animate
>
	<rect width="7" height="7" x="3" y="3" rx="1" />
	<rect width="7" height="7" x="14" y="3" rx="1" class="lifted" />
	<rect width="7" height="7" x="14" y="14" rx="1" />
	<rect width="7" height="7" x="3" y="14" rx="1" />
</svg>

<style>
	svg {
		overflow: visible;
	}

	.animate .lifted {
		animation: lift 0.45s cubic-bezier(0.34, 1.56, 0.64, 1);
	}

	@keyframes lift {
		45% {
			transform: translate(2px, -2px);
		}
	}
</style>
