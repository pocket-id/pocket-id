<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import * as Card from './ui/card';
	import { m } from '$lib/paraglide/messages';
	import { onMount } from 'svelte';

	let {
		children,
		showAlternativeSignInMethodButton = false
	}: {
		children: Snippet;
		showAlternativeSignInMethodButton?: boolean;
	} = $props();

	let mounted = $state(false);

	onMount(() => {
		mounted = true;
	});
</script>

{#if mounted}
	<!-- Desktop with sliding reveal animation -->
	<div class="hidden h-screen items-center overflow-hidden text-center lg:flex">
		<!-- Content area that fades in after background slides -->
		<div
			class="animate-delayed-fade relative z-10 flex h-full min-w-[650px] p-16 {showAlternativeSignInMethodButton
				? 'pb-0'
				: ''}"
		>
			<div class="flex h-full w-full flex-col">
				<div class="flex flex-grow flex-col items-center justify-center">
					{@render children()}
				</div>
				{#if showAlternativeSignInMethodButton}
					<div
						class="animate-fade-in mb-4 flex items-center justify-center"
						style="animation-delay: 1000ms;"
					>
						<a
							href={page.url.pathname == '/login'
								? '/login/alternative'
								: `/login/alternative?redirect=${encodeURIComponent(
										page.url.pathname + page.url.search
									)}`}
							class="text-muted-foreground text-xs transition-colors hover:underline"
						>
							{m.dont_have_access_to_your_passkey()}
						</a>
					</div>
				{/if}
			</div>
		</div>

		<!-- Background image with slide animation -->
		<div class="animate-slide-bg-container absolute bottom-0 right-0 top-0 z-0">
			<img
				src="/api/application-configuration/background-image"
				class="h-full rounded-l-[60px] object-cover"
				alt={m.login_background()}
			/>
		</div>
	</div>

	<!-- Mobile -->
	<div
		class="flex h-screen items-center justify-center bg-[url('/api/application-configuration/background-image')] bg-cover bg-center text-center lg:hidden"
	>
		<Card.Root class="animate-fade-in mx-3 w-full max-w-md" style="animation-delay: 200ms;">
			<Card.CardContent
				class="px-4 py-10 sm:p-10 {showAlternativeSignInMethodButton ? 'pb-3 sm:pb-3' : ''}"
			>
				{@render children()}
				{#if showAlternativeSignInMethodButton}
					<a
						href={page.url.pathname == '/login'
							? '/login/alternative'
							: `/login/alternative?redirect=${encodeURIComponent(
									page.url.pathname + page.url.search
								)}`}
						class="text-muted-foreground animate-fade-in mt-7 flex justify-center text-xs transition-colors hover:underline"
						style="animation-delay: 400ms;"
					>
						{m.dont_have_access_to_your_passkey()}
					</a>
				{/if}
			</Card.CardContent>
		</Card.Root>
	</div>
{/if}

<style>
	/* Animation for the container that holds the background image */
	@keyframes slide-bg-container {
		0% {
			width: 100%;
			left: 0;
		}
		100% {
			width: calc(100% - 650px);
			left: 650px;
		}
	}

	.animate-slide-bg-container {
		animation: slide-bg-container 1.2s cubic-bezier(0.33, 1, 0.68, 1) forwards;
	}

	/* Fade in for content after the slide is mostly complete */
	@keyframes delayed-fade {
		0%,
		40% {
			opacity: 0;
		}
		100% {
			opacity: 1;
		}
	}

	.animate-delayed-fade {
		animation: delayed-fade 1.5s ease-out forwards;
	}
</style>
