<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import ConfirmDialog from '#lib/components/confirm-dialog/confirm-dialog.svelte';
	import Error from '#lib/components/error.svelte';
	import NavigationProgress from '#lib/components/navigation-progress.svelte';
	import { Toaster } from '#lib/components/ui/sonner/index.ts';
	import UnsavedChangesBar from '#lib/components/unsaved-changes-bar.svelte';
	import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { startPageTrace } from '#lib/utils/tracing-util.ts';
	import { ModeWatcher } from 'mode-watcher';
	import { type Snippet } from 'svelte';
	import '../app.css';
	import type { LayoutData } from './$types';

	// Start a new page-level trace on each navigation so a page view and the API calls it triggers are correlated as a single trace.
	beforeNavigate((nav) => {
		if (!nav.shallow) startPageTrace(nav.to?.url.pathname);
	});

	let {
		data,
		children
	}: {
		data: LayoutData;
		children: Snippet;
	} = $props();

	const { appConfig } = data;

	// Toasts sit above the phone bottom bar, and while the unsaved bar shows they rise above it too (its height plus 8px)
	const barOpen = $derived(unsavedChanges.hasChanges || !!unsavedChanges.status);
	const phoneClearance = $derived(
		barOpen ? 136 : page.url.pathname.startsWith('/settings') ? 76 : 16
	);
</script>

{#if !appConfig}
	<Error message={m.critical_error_occurred_contact_administrator()} showButton={false} />
{:else}
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
<NavigationProgress />
<ConfirmDialog />
<UnsavedChangesBar />
<ModeWatcher disableTransitions={false} />
