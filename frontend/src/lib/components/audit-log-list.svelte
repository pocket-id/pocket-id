<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { m } from '$lib/paraglide/messages';
	import { translateAuditLogEvent } from '$lib/utils/audit-log-translator';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import type { ColumnSpec } from '$lib/components/pocket-id-table';

	let {
		auditLogs,
		isAdmin = false,
		requestOptions
	}: {
		auditLogs: Paginated<AuditLog>;
		isAdmin?: boolean;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const auditLogService = new AuditLogService();

	const columns = [
		{ title: m.time(), accessorKey: 'createdAt', sortable: true, cell: TimeCell },
		...(isAdmin ? [{ title: 'Username', cell: UsernameCell }] : []),
		{ title: m.event(), accessorKey: 'event', sortable: true, cell: EventCell },
		{ title: m.approximate_location(), accessorKey: 'city', sortable: true, cell: LocationCell },
		{ title: m.ip_address(), accessorKey: 'ipAddress', sortable: true, cell: IpCell },
		{ title: m.device(), accessorKey: 'device', sortable: true, cell: DeviceCell },
		{ title: m.client(), cell: ClientCell }
	] satisfies ColumnSpec<AuditLog>[];
</script>

{#snippet TimeCell({ value }: { value: unknown })}
	{new Date(Number(value)).toLocaleString()}
{/snippet}

{#snippet UsernameCell({ item }: { item: AuditLog })}
	{#if item.username}
		{item.username}
	{:else}
		{m.unknown()}
	{/if}
{/snippet}

{#snippet EventCell({ item }: { item: AuditLog })}
	<Badge class="rounded-full" variant="outline">{translateAuditLogEvent(item.event)}</Badge>
{/snippet}

{#snippet LocationCell({ item }: { item: AuditLog })}
			{#if item.city && item.country}
				{item.city}, {item.country}
			{:else if item.country}
				{item.country}
			{:else}
				{m.unknown()}
			{/if}
{/snippet}

{#snippet IpCell({ item }: { item: AuditLog })}
	{item.ipAddress}
{/snippet}

{#snippet DeviceCell({ item }: { item: AuditLog })}
	{item.device}
{/snippet}

{#snippet ClientCell({ item }: { item: AuditLog })}
	{item.data?.clientName}
{/snippet}

<PocketIdTable
	items={auditLogs}
	bind:requestOptions
	onRefresh={async (options) =>
		isAdmin
			? (auditLogs = await auditLogService.listAllLogs(options))
			: (auditLogs = await auditLogService.list(options))}
	{columns}
	persistKey="pocket-id-audit-logs"
	withoutSearch
	selectionDisabled={true}
/>
