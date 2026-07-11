<script lang="ts">
	import ConfirmDialog from '$lib/components/confirm-dialog/confirm-dialog.svelte';
	import Error from '$lib/components/error.svelte';
	import Header from '$lib/components/header/header.svelte';
	import { Toaster } from '$lib/components/ui/sonner';
	import { m } from '$lib/paraglide/messages';
	import { startPageTrace } from '$lib/utils/tracing-util';
	import { beforeNavigate } from '$app/navigation';
	import { ModeWatcher } from 'mode-watcher';
	import { type Snippet } from 'svelte';
	import '../app.css';
	import type { LayoutData } from './$types';

	// Start a new page-level trace on each navigation so a page view and the API calls it triggers are correlated as a single trace.
	beforeNavigate((nav) => startPageTrace(nav.to?.url.pathname));

	let {
		data,
		children
	}: {
		data: LayoutData;
		children: Snippet;
	} = $props();

	const { appConfig } = data;
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
<ModeWatcher disableTransitions={false} />
