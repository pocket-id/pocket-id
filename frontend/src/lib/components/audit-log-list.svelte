<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { m } from '$lib/paraglide/messages';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { AuditLog, AuditLogFilter } from '$lib/types/audit-log.type';
	import { translateAuditLogEvent } from '$lib/utils/audit-log-translator';

	let {
		isAdmin = false,
		filters
	}: {
		isAdmin?: boolean;
		filters?: AuditLogFilter;
	} = $props();

	const auditLogService = new AuditLogService();
	let tableRef: AdvancedTable<AuditLog>;

	const columns: AdvancedTableColumn<AuditLog>[] = [
		{
			label: m.time(),
			column: 'createdAt',
			sortable: true,
			value: (item) => new Date(item.createdAt).toLocaleString()
		},
		{
			label: m.username(),
			column: 'username',
			hidden: !isAdmin,
			value: (item) => item.username ?? m.unknown()
		},
		{
			label: m.event(),
			column: 'event',
			sortable: true,
			cell: EventCell
		},
		{
			label: m.approximate_location(),
			key: 'location',
			sortable: true,
			value: (item) => formatLocation(item)
		},
		{
			label: m.ip_address(),
			column: 'ipAddress',
			sortable: true
		},
		{
			label: m.device(),
			column: 'device',
			sortable: true
		},
		{
			label: m.client(),
			key: 'client',
			value: (item) => item.data?.clientName
		}
	];

	$effect(() => {
		if (filters) {
			tableRef?.refresh();
		}
	});

	export async function refresh() {
		await tableRef.refresh();
	}

	function formatLocation(log: AuditLog) {
		if (log.city && log.country) {
			return `${log.city}, ${log.country}`;
		} else if (log.country) {
			return log.country;
		} else {
			return m.unknown();
		}
	}

	function wrapFilters(filters?: Record<string, string>) {
		if (!filters) return undefined;
		return Object.fromEntries(
			Object.entries(filters)
				.filter(([_, value]) => value !== undefined && value !== null && value !== '')
				.map(([key, value]) => [key, [value]])
		);
	}
</script>

{#snippet EventCell({ item }: { item: AuditLog })}
	<Badge class="rounded-full" variant="outline">
		{translateAuditLogEvent(item.event)}
	</Badge>
{/snippet}

<AdvancedTable
	id="audit-log-list-{isAdmin ? 'admin' : 'user'}"
	bind:this={tableRef}
	fetchCallback={async (options) =>
		isAdmin
			? await auditLogService.listAllLogs({
					...options,
					filters: wrapFilters(filters)
				})
			: await auditLogService.list(options)}
	defaultSort={{ column: 'createdAt', direction: 'desc' }}
	withoutSearch
	{columns}
/>
