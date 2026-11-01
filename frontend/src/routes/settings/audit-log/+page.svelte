<script lang="ts">
	import AuditLogDateFilter from '#lib/components/audit-log-date-filter.svelte';
	import AuditLogList from '#lib/components/audit-log-list.svelte';
	import HashTabs from '#lib/components/hash-tabs.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Tabs from '@pocket-id/ui/tabs';
	import { m } from '#lib/paraglide/messages.js';
	import type { AuditLogDateRange } from '#lib/types/audit-log.type.ts';
	import SessionList from './session-list.svelte';

	let { data } = $props();

	let sessions = $state(data.sessions);
	let dateRange: AuditLogDateRange = $state({});
	let tab = $state('devices');
</script>

<svelte:head>
	<title>{m.activity()}</title>
</svelte:head>

<div class="flex flex-col gap-8">
	<PageHeader title={m.activity()} description={m.activity_description()} />

	<HashTabs bind:value={tab} class="gap-6">
		<Tabs.List variant="line" class="max-sm:gap-4" aria-label={m.activity_sections()}>
			<Tabs.Trigger value="devices">
				{m.devices()}
				<span class="text-fg-muted tabular-nums">{sessions.length}</span>
			</Tabs.Trigger>
			<Tabs.Trigger value="history">{m.history()}</Tabs.Trigger>
		</Tabs.List>

		<Tabs.Content value="devices">
			<SessionList bind:sessions />
		</Tabs.Content>

		<Tabs.Content value="history">
			<div role="group" aria-label={m.filters()} class="-mb-1 flex items-center gap-1 py-1">
				<AuditLogDateFilter bind:range={dateRange} />
			</div>
			<AuditLogList {dateRange} />
		</Tabs.Content>
	</HashTabs>
</div>
