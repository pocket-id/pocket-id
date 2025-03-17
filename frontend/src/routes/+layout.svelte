<script lang="ts">
	import { browser } from '$app/environment';
	import ConfirmDialog from '$lib/components/confirm-dialog/confirm-dialog.svelte';
	import Error from '$lib/components/error.svelte';
	import Header from '$lib/components/header/header.svelte';
	import { Toaster } from '$lib/components/ui/sonner';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { ModeWatcher } from 'mode-watcher';
	import type { Snippet } from 'svelte';
	import '../app.css';
	import type { LayoutData } from './$types';

	let {
		data,
		children
	}: {
		data: LayoutData;
		children: Snippet;
	} = $props();

	const { user, appConfig } = data;

	if (browser && user) {
		userStore.setUser(user);
	}
	if (appConfig) {
		appConfigStore.set(appConfig);
	}
</script>

{#if !appConfig}
	<Error
		message="A critical error occurred. Please contact your administrator."
		showButton={false}
	/>
{:else}
	<Header />
	{@render children()}
{/if}
<Toaster />
<ConfirmDialog />
<ModeWatcher />
