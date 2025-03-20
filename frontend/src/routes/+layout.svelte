<script lang="ts">
	import { browser } from '$app/environment';
	import ConfirmDialog from '$lib/components/confirm-dialog/confirm-dialog.svelte';
	import Error from '$lib/components/error.svelte';
	import Header from '$lib/components/header/header.svelte';
	import { Toaster } from '$lib/components/ui/sonner';
	import { m } from '$lib/paraglide/messages';
	import { setLocale } from '$lib/paraglide/runtime';
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
		if (user.locale){
			setLocale(user.locale, {reload: false});
		}
		userStore.setUser(user);
	}
	if (appConfig) {
		appConfigStore.set(appConfig);
	}
</script>

{#if !appConfig}
	<Error message={m.critical_error_occurred_contact_administrator()} showButton={false} />
{:else}
	<Header />
	{@render children()}
{/if}
<Toaster />
<ConfirmDialog />
<ModeWatcher />
