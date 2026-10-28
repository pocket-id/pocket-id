<script lang="ts">
	import AuditLogDateFilter from '#lib/components/audit-log-date-filter.svelte';
	import AuditLogList from '#lib/components/audit-log-list.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import AdvancedTableFilter from '#lib/components/table/advanced-table-filter.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import AuditLogService from '#lib/services/audit-log-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type { AuditLogDateRange, AuditLogFilter } from '#lib/types/audit-log.type.ts';
	import { eventTypes } from '#lib/utils/audit-log-translator.ts';
	import { revealFocused } from '#lib/utils/event-util.ts';

	const auditLogService = new AuditLogService();

	const noFilters: AuditLogFilter = { userID: '', event: '', location: '', clientName: '' };
	let filters = $state({ ...noFilters });

	let dateRange: AuditLogDateRange = $state({});
	let filterRow = $state<HTMLElement>();
	let userOptions = $state<{ label: string; value: string }[]>([]);
	let clientOptions = $state<{ label: string; value: string }[]>([]);

	auditLogService.listUsers().then((users) => {
		userOptions = Object.entries(users)
			.map(([value, label]) => ({ value, label }))
			.sort((a, b) => a.label.localeCompare(b.label));
	});
	auditLogService.listClientNames().then((names) => {
		clientOptions = names.sort().map((name) => ({ value: name, label: name }));
	});

	type Facet = {
		key: keyof AuditLogFilter;
		title: string;
		options: { label: string; value: string }[];
	};

	const facets: Facet[] = $derived([
		{ key: 'userID', title: m.user(), options: userOptions },
		{
			key: 'event',
			title: m.event(),
			options: Object.entries(eventTypes).map(([value, label]) => ({ value, label }))
		},
		{ key: 'clientName', title: m.app(), options: clientOptions },
		{
			key: 'location',
			title: m.network(),
			options: [
				{ value: 'external', label: m.external_networks() },
				{ value: 'internal', label: m.internal_networks() }
			]
		}
	]);
</script>

<svelte:head>
	<title>{m.audit_log()}</title>
</svelte:head>

<PageHeader
	title={m.audit_log()}
	description={m.audit_log_page_description({ appName: $appConfigStore.appName })}
/>

<div
	bind:this={filterRow}
	role="group"
	aria-label={m.filters()}
	onfocusin={revealFocused}
	class="scroll-fade-x -mx-1 mt-6 -mb-1 flex scroll-px-1 items-center gap-1 p-1 max-sm:[scrollbar-width:none] max-sm:overflow-x-auto sm:flex-wrap"
>
	<AuditLogDateFilter bind:range={dateRange} />
	{#each facets as facet (facet.key)}
		<AdvancedTableFilter
			title={facet.title}
			options={facet.options}
			value={filters[facet.key] || undefined}
			onChange={(value) => (filters[facet.key] = (value as string | undefined) ?? '')}
		/>
	{/each}
</div>

<!-- The date range is never cleared, since the date filter keeps its own selection -->
<!-- Focus moves to the filters because the action removes itself -->
<AuditLogList
	isAdmin
	{filters}
	{dateRange}
	onClearFilters={Object.values(filters).some(Boolean)
		? () => {
				filters = { ...noFilters };
				filterRow?.querySelector('button')?.focus();
			}
		: undefined}
/>
