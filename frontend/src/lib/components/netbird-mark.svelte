<script lang="ts">
	import { cn } from '@pocket-id/ui/utils';
	import { onMount } from 'svelte';

	let { class: className }: { class?: string } = $props();

	let run = $state(0);

	// The bird first flaps a second after the page loads, so it isn't lost in the page's entrance, and then again at an irregular pace, so it reads as alive rather than as a busy indicator
	onMount(() => {
		let timer: ReturnType<typeof setTimeout>;
		const schedule = (delay: number) => {
			timer = setTimeout(() => {
				run++;
				schedule(4000 + Math.random() * 4000);
			}, delay);
		};
		schedule(1000);
		return () => clearTimeout(timer);
	});
</script>

{#key run}
	<svg
		viewBox="0 0 31 23"
		class={cn('size-5 overflow-visible', className)}
		class:flap={run > 0}
		aria-hidden="true"
	>
		<path
			class="wing right"
			d="M21.4631 0.523438C17.8173 0.857913 16.0028 2.95675 15.3171 4.01871L4.66406 22.4734H17.5163L30.1929 0.523438H21.4631Z"
		/>
		<path
			class="wing left"
			d="M17.5265 22.4737L0 3.88525C0 3.88525 19.8177 -1.44128 21.7493 15.1738L17.5265 22.4737Z"
		/>
		<path
			class="body"
			d="M14.9236 4.70563L9.54688 14.0208L17.5158 22.4747L21.7385 15.158C21.0696 9.44682 18.2851 6.32784 14.9236 4.69727"
		/>
	</svg>
{/key}

<style>
	.wing {
		fill: #f68330;
	}

	.body {
		fill: #f05252;
	}

	/* Both wings hinge on the tail tip where they meet, the right one swings clockwise to flatten and the left one the other way */
	.wing {
		transform-box: view-box;
		transform-origin: 17.5px 22.47px;
	}

	.flap .wing {
		animation: flap calc(var(--dur-unfold) * 1.5) var(--ease-in-out);
	}

	.right {
		--s: 1;
	}

	.left {
		--s: -1;
	}

	/* Two beats, the second one smaller */
	@keyframes flap {
		30% {
			rotate: calc(10deg * var(--s));
		}

		60% {
			rotate: 0deg;
		}

		80% {
			rotate: calc(5deg * var(--s));
		}
	}
</style>
