<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils/style';
	import type { Snippet } from 'svelte';
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
</script>

<!-- Desktop with sliding reveal animation -->
<div class="hidden h-screen items-center overflow-hidden text-center lg:flex">
	<!-- Content area that fades in after background slides -->
	<div
		class="relative z-10 flex h-full w-[650px] p-16 {cn(
			showAlternativeSignInMethodButton && 'pb-0',
			animate && 'animate-delayed-fade'
		)}"
	>
		<div class="flex h-full w-full flex-col overflow-hidden">
			<div class="relative flex grow flex-col items-center justify-center overflow-auto">
				{@render children()}
			</div>
			{#if showAlternativeSignInMethodButton}
				<div
					class="mb-4 flex items-center justify-center"
					style={animate ? 'animation-delay: 1000ms;' : ''}
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
	<div class="{cn(animate && 'animate-slide-bg-container')} absolute top-0 right-0 bottom-0 z-0">
		<img
			src="/api/application-configuration/background-image"
			class="h-screen rounded-l-[60px] object-cover {animate ? 'w-full' : 'w-[calc(100vw-650px)]'}"
			alt={m.login_background()}
		/>
	</div>
</div>

<!-- Mobile -->
<div
	class="flex h-screen items-center justify-center bg-[url('/api/application-configuration/background-image')] bg-cover bg-center text-center lg:hidden"
>
	<Card.Root class="mx-3 w-full max-w-md" style={animate ? 'animation-delay: 200ms;' : ''}>
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
					class="text-muted-foreground mt-7 flex justify-center text-xs transition-colors hover:underline"
				>
					{m.dont_have_access_to_your_passkey()}
				</a>
			{/if}
		</Card.CardContent>
	</Card.Root>
</div>
