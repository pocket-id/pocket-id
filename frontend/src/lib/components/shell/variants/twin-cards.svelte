<script lang="ts">
	import { PersistedState } from 'runed';
	import PhoneNav from '../phone-nav.svelte';
	import ClassicSidebar from './classic-sidebar.svelte';
	import type { ShellProps } from './registry.ts';

	let { groups, versionInformation, showUpdate, children }: ShellProps = $props();

	const collapsed = new PersistedState('sidebar-collapsed', false);
</script>

<div class="twin-cards flex min-h-dvh gap-2 sm:p-2">
	<ClassicSidebar
		{groups}
		{versionInformation}
		{showUpdate}
		bind:collapsed={collapsed.current}
		class="border-border top-2 h-[calc(100dvh-1rem)] rounded-xl border"
	/>
	<div
		class="bg-surface sm:border-border flex min-w-0 flex-1 flex-col sm:rounded-xl sm:border sm:shadow-(--card-shadow)"
	>
		<PhoneNav
			{groups}
			version="Pocket ID {versionInformation.currentVersion}"
			newestVersion={showUpdate ? versionInformation.newestVersion : null}
		/>
		{@render children()}
	</div>
</div>

<style>
	/* The body carries the shell color so the overscroll bounce reveals it instead of the canvas */
	:global(body:has(.twin-cards)) {
		background: var(--shell-bg);
	}

	.twin-cards {
		--card-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
	}

	:global(.dark) .twin-cards {
		--card-shadow: none;
	}

	/* Outranks the sidebar's hairline, which would sit inside the card's edge */
	.twin-cards :global([data-slot='app-sidebar']) {
		box-shadow: var(--card-shadow);
	}
</style>
