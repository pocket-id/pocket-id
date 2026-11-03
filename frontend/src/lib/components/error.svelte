<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '@pocket-id/ui/button';
	import * as Empty from '@pocket-id/ui/empty';
	import { m } from '#lib/paraglide/messages.js';
	import CompassIcon from '~icons/solar/compass-bold-duotone';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold-duotone';

	let { message, showButton = true }: { message: string; showButton?: boolean } = $props();

	const title = $derived(page.status === 404 ? m.page_not_found_title() : m.something_went_wrong());
</script>

<svelte:head>
	<title>{title}</title>
</svelte:head>

<Empty.Root class="mx-auto min-h-[70dvh] items-center justify-center px-4 text-center">
	<Empty.Header class="items-center">
		<Empty.Media variant="icon">
			{#if page.status === 404}
				<CompassIcon />
			{:else}
				<TriangleAlertIcon />
			{/if}
		</Empty.Media>
		<h1 class="font-display text-display-s text-fg">{title}</h1>
		<Empty.Description class="wrap-anywhere">{message}</Empty.Description>
	</Empty.Header>
	{#if showButton}
		<Button variant="secondary" href="/">{m.go_back_to_home()}</Button>
	{/if}
</Empty.Root>
