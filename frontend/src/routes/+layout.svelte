<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ConfirmDialog from '$lib/components/confirm-dialog/confirm-dialog.svelte';
	import Error from '$lib/components/error.svelte';
	import Header from '$lib/components/header/header.svelte';
	import { Toaster } from '$lib/components/ui/sonner';
	import { m } from '$lib/paraglide/messages';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import userStore from '$lib/stores/user-store';
	import { getAuthRedirectPath } from '$lib/utils/redirection-util';
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

	const redirectPath = getAuthRedirectPath(page.url.pathname, user);
	if (redirectPath) {
		goto(redirectPath);
	}

	if (user) {
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
<Toaster
	toastOptions={{
		classes: {
			toast: 'border border-primary/30!',
			title: 'text-foreground',
			description: 'text-muted-foreground',
			actionButton: 'bg-primary text-primary-foreground hover:bg-primary/90',
			cancelButton: 'bg-muted text-muted-foreground hover:bg-muted/80',
			closeButton: 'text-muted-foreground hover:text-foreground'
		}
	}}
/>
<ConfirmDialog />
<ModeWatcher />
