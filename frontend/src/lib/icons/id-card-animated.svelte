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
	<path d="M16 10h2" />
	<path d="M16 14h2" />
	<g class="avatar">
		<path d="M6.17 15a3 3 0 0 1 5.66 0" />
		<circle cx="9" cy="11" r="2" />
	</g>
	<rect x="2" y="5" width="20" height="14" rx="2" />
</svg>

<style>
	svg {
		overflow: visible;
	}

	.animate .avatar {
		animation: nod 0.45s cubic-bezier(0.34, 1.56, 0.64, 1);
	}

	@keyframes nod {
		45% {
			transform: translateY(-1.5px);
		}
	}
</style>
