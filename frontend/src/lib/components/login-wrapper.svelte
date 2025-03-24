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
	<!-- Desktop -->
	<div class="hidden h-screen items-center text-center lg:flex">
		<div
			class="animate-fade-in h-full min-w-[650px] p-16 {showAlternativeSignInMethodButton
				? 'pb-0'
				: ''}"
			style="animation-delay: 100ms;"
		>
			<div class="flex h-full flex-col">
				<div class="flex flex-grow flex-col items-center justify-center">
					{@render children()}
				</div>
				{#if showAlternativeSignInMethodButton}
					<div class="animate-fade-in mb-4 flex justify-center" style="animation-delay: 400ms;">
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
		<img
			src="/api/application-configuration/background-image"
			class="animate-fade-in h-screen w-[calc(100vw-650px)] rounded-l-[60px] object-cover"
			style="animation-delay: 300ms;"
			alt={m.login_background()}
		/>
	</div>

	<!-- Mobile -->
	<div
		class="flex h-screen items-center justify-center bg-[url('/api/application-configuration/background-image')] bg-cover bg-center text-center lg:hidden"
	>
		<Card.Root class="animate-fade-in mx-3" style="animation-delay: 200ms;">
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
