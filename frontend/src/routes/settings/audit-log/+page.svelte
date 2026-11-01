<script lang="ts">
	import AuditLogDateFilter from '#lib/components/audit-log-date-filter.svelte';
	import AuditLogList from '#lib/components/audit-log-list.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type { AuditLogDateRange } from '#lib/types/audit-log.type.ts';

	let dateRange: AuditLogDateRange = $state({});
</script>

<svelte:head>
	<title>{m.activity()}</title>
</svelte:head>

<PageHeader title={m.activity()} description={m.activity_description()} />

<div role="group" aria-label={m.filters()} class="mt-6 -mb-1 flex items-center gap-1 py-1">
	<AuditLogDateFilter bind:range={dateRange} />
</div>

<AuditLogList {dateRange} />
