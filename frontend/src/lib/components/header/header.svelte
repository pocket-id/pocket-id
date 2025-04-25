<script lang="ts">
	import { page } from '$app/state';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import Logo from '../logo.svelte';
	import HeaderAvatar from './header-avatar.svelte';

	const authUrls = [/^\/authorize$/, /^\/device$/, /^\/login(?:\/.*)?$/, /^\/logout$/];

	let isAuthPage = $derived(
		!page.error && authUrls.some((pattern) => pattern.test(page.url.pathname))
	);
</script>

<div class=" w-full {isAuthPage ? 'absolute top-0 z-10 mt-4' : 'border-b'}">
	<div
		class="{!isAuthPage
			? 'max-w-[1640px]'
			: ''} mx-auto flex w-full items-center justify-between px-4 md:px-10"
	>
		<div class="flex h-16 items-center">
			{#if !isAuthPage}
				<a
					href="/settings/account"
					class="flex items-center gap-3 transition-opacity hover:opacity-80"
				>
					<Logo class="h-8 w-8" />
					<h1 class="text-lg font-semibold tracking-tight" data-testid="application-name">
						{$appConfigStore.appName}
					</h1>
				</a>
			{/if}
		</div>
		<div class="flex items-center justify-between gap-4">
			{#if $userStore?.id}
				<HeaderAvatar />
			{/if}
		</div>
	</div>
</div>
