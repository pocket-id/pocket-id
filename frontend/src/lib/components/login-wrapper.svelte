<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { cachedBackgroundImage } from '$lib/utils/cached-image-util';
	import { cn } from '$lib/utils/style';
	import type { Snippet } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import * as Card from './ui/card';

	let {
		children,
		showAlternativeSignInMethodButton = false,
		animate = false
	}: {
		children: Snippet;
		showAlternativeSignInMethodButton?: boolean;
		animate?: boolean;
	} = $props();

	const isDesktop = new MediaQuery('min-width: 1024px');
	let alternativeSignInButton = $state({
		href: '/login/alternative',
		label: m.alternative_sign_in_methods()
	});

	appConfigStore.subscribe((config) => {
		if (config.emailOneTimeAccessAsUnauthenticatedEnabled) {
			alternativeSignInButton.href = '/login/alternative';
			alternativeSignInButton.label = m.alternative_sign_in_methods();
		} else {
			alternativeSignInButton.href = '/login/alternative/code';
			alternativeSignInButton.label = m.sign_in_with_login_code();
		}

		if (page.url.pathname != '/login') {
			alternativeSignInButton.href = `${alternativeSignInButton.href}?redirect=${encodeURIComponent(page.url.pathname + page.url.search)}`;
		}
	});
</script>

{#if isDesktop.current}
	<div class="h-screen items-center overflow-hidden text-center">
		<div
			class="relative z-10 flex h-full w-[650px] p-16 {cn(
				showAlternativeSignInMethodButton && 'pb-0',
				animate && 'animate-delayed-fade'
			)}"
		>
			<div class="flex h-full w-full flex-col overflow-hidden">
				<div class="relative flex flex-grow flex-col items-center justify-center overflow-auto">
					{@render children()}
				</div>
				{#if showAlternativeSignInMethodButton}
					<div
						class="mb-4 flex items-center justify-center"
						style={animate ? 'animation-delay: 500ms;' : ''}
					>
						<a
							href={alternativeSignInButton.href}
							class="text-muted-foreground text-xs transition-colors hover:underline"
						>
							{alternativeSignInButton.label}
						</a>
					</div>
				{/if}
			</div>
		</div>

		<!-- Background image with slide animation -->
		<div class="{cn(animate && 'animate-slide-bg-container')} absolute top-0 right-0 bottom-0 z-0">
			<img
				src={cachedBackgroundImage.getUrl()}
				class="h-screen rounded-l-[60px] object-cover {animate
					? 'w-full'
					: 'w-[calc(100vw-650px)]'}"
				alt={m.login_background()}
			/>
		</div>
	</div>
{:else}
	<div
		class="flex h-screen items-center justify-center bg-cover bg-center text-center"
		style="background-image: url({cachedBackgroundImage.getUrl()});"
	>
		<Card.Root class="mx-3 w-full max-w-md" style={animate ? 'animation-delay: 200ms;' : ''}>
			<Card.CardContent
				class="px-4 py-10 sm:p-10 {showAlternativeSignInMethodButton ? 'pb-3 sm:pb-3' : ''}"
			>
				{@render children()}
				{#if showAlternativeSignInMethodButton}
					<a
						href={alternativeSignInButton.href}
						class="text-muted-foreground mt-7 flex justify-center text-xs transition-colors hover:underline"
					>
						{alternativeSignInButton.label}
					</a>
				{/if}
			</Card.CardContent>
		</Card.Root>
	</div>
{/if}
