<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import Logo from '../logo.svelte';
	import Separator from '../ui/separator/separator.svelte';
	import HeaderAvatar from './header-avatar.svelte';
	import ModeSwitcher from './mode-switcher.svelte';

	const authUrls = [
		/^\/interaction$/,
		/^\/interaction\/error$/,
		/^\/device$/,
		/^\/login(?:\/.*)?$/,
		/^\/logout$/,
		/^\/signup(?:\/.*)?$/
	];

	let isAuthPage = $derived(
		!page.error && authUrls.some((pattern) => pattern.test(page.url.pathname))
	);
</script>

<div
	class=" w-full {isAuthPage
		? 'absolute top-0 z-10 mt-3 lg:mt-8 pr-2 lg:pr-3'
		: 'pt-3 bg-muted/40 dark:bg-background '}"
>
	<div
		class="{!isAuthPage
			? 'max-w-[1720px]'
			: ''} mx-auto flex w-full items-center justify-between px-4 md:px-10"
	>
		<div class="flex h-16 items-center">
			{#if !isAuthPage}
				<a href="/" class="flex items-center transition-opacity hover:opacity-80">
					<Logo class="size-8" />
					<Separator orientation="vertical" class="h-5! bg-neutral-600 ml-2 mr-3" />
					<h1 class="text-2xl font-gloock" data-testid="application-name">
						{m.settings()}
					</h1>
				</a>
			{/if}
		</div>
		<div class="flex items-center justify-between gap-4">
			{#if !isAuthPage}
				<ModeSwitcher />
			{/if}
			{#if $userStore?.id}
				<HeaderAvatar />
			{/if}
		</div>
	</div>
</div>
