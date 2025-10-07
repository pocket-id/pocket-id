<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import AuditLogService from '$lib/services/audit-log-service';
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

	export async function refresh() {
		await tableRef.refresh();
	}
</script>

<AdvancedTable
	id="audit-log-list"
	bind:this={tableRef}
	fetchCallback={async (options) =>
		isAdmin
			? await auditLogService.listAllLogs(options, filters)
			: await auditLogService.list(options)}
	defaultSort={{ column: 'createdAt', direction: 'desc' }}
	columns={[
		{ label: m.time(), sortColumn: 'createdAt' },
		...(isAdmin ? [{ label: 'Username' }] : []),
		{ label: m.event(), sortColumn: 'event' },
		{ label: m.approximate_location(), sortColumn: 'city' },
		{ label: m.ip_address(), sortColumn: 'ipAddress' },
		{ label: m.device(), sortColumn: 'device' },
		{ label: m.client() }
	]}
	withoutSearch
>
	{#snippet rows({ item })}
		<Table.Cell>{new Date(item.createdAt).toLocaleString()}</Table.Cell>
		{#if isAdmin}
			<Table.Cell>
				{#if item.username}
					{item.username}
				{:else}
					Unknown User
				{/if}
			</Table.Cell>
		{/if}
		<Table.Cell>
			<Badge class="rounded-full" variant="outline">{translateAuditLogEvent(item.event)}</Badge>
		</Table.Cell>
		<Table.Cell
			>{item.city && item.country ? `${item.city}, ${item.country}` : m.unknown()}</Table.Cell
		>
		<Table.Cell>{item.ipAddress}</Table.Cell>
		<Table.Cell>{item.device}</Table.Cell>
		<Table.Cell>{item.data.clientName}</Table.Cell>
	{/snippet}
</AdvancedTable>
