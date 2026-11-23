<script lang="ts">
	import { PersistedState } from 'runed';
	import PhoneNav from '../phone-nav.svelte';
	import ClassicSidebar from './classic-sidebar.svelte';
	import type { ShellProps } from './registry.ts';

	let { groups, versionInformation, showUpdate, children }: ShellProps = $props();

	const collapsed = new PersistedState('sidebar-collapsed', false);
</script>

<div class="bg-surface flex min-h-dvh">
	<ClassicSidebar {groups} {versionInformation} {showUpdate} bind:collapsed={collapsed.current} />
	<div class="flex min-w-0 flex-1 flex-col">
		<PhoneNav
			{groups}
			version="Pocket ID {versionInformation.currentVersion}"
			newestVersion={showUpdate ? versionInformation.newestVersion : null}
		/>
		{@render children()}
	</div>
</div>
