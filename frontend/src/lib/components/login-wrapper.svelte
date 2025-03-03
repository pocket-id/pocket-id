<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import * as Card from './ui/card';

	let {
		children,
		showAlternativeSignInMethodButton = false
	}: {
		children: Snippet;
		showAlternativeSignInMethodButton?: boolean;
	} = $props();
</script>

<!-- Desktop -->
<div class="hidden h-screen items-center text-center lg:flex">
	<div class="h-full min-w-[650px] p-16 {showAlternativeSignInMethodButton ? 'pb-0' : ''}">
		<div class="flex h-full flex-col">
			<div class="flex flex-grow flex-col items-center justify-center">
				{@render children()}
			</div>
			{#if showAlternativeSignInMethodButton}
				<div class="mb-4 flex justify-center">
					<a
						href="/login/alternative?redirect={encodeURIComponent(
							page.url.pathname + page.url.search
						)}"
						class="text-muted-foreground text-xs"
					>
						Don't have access to your passkey?
					</a>
				</div>
			{/if}
		</div>
	</div>
	<img
		src="/api/application-configuration/background-image"
		class="h-screen w-[calc(100vw-650px)] rounded-l-[60px] object-cover"
		alt="Login background"
	/>
</div>

<!-- Mobile -->
<div
	class="flex h-screen items-center justify-center bg-[url('/api/application-configuration/background-image')] bg-cover bg-center text-center lg:hidden"
>
	<Card.Root class="mx-3">
		<Card.CardContent
			class="px-4 py-10 sm:p-10 {showAlternativeSignInMethodButton ? 'pb-3 sm:pb-3' : ''}"
		>
			{@render children()}
			{#if showAlternativeSignInMethodButton}
				<a
					href="/login/alternative?redirect={encodeURIComponent(
						page.url.pathname + page.url.search
					)}"
					class="text-muted-foreground mt-5 text-xs"
				>
					Don't have access to your passkey?
				</a>
			{/if}
		</Card.CardContent>
	</Card.Root>
</div>
