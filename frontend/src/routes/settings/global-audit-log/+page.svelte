<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import AuditLogList from '../audit-log/audit-log-list.svelte';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { Paginated } from '$lib/types/pagination.type';

	let { data } = $props();

	// Store audit logs in local state
	let auditLogs = $state<Paginated<AuditLog>>(data.auditLogs);

	// Create a reference to the AuditLogList component
	let auditLogListComponent: any;

	// Initialize selections with any existing filter values
	let selectedUserId = $state<string | null>(data.filters.userId || '');
	let selectedEventType = $state<string | null>(data.filters.eventType || '');
	let selectedClientId = $state<string | null>(data.filters.clientId || '');

	// Create request options object
	let requestOptions = $state({
		sort: {
			column: 'createdAt',
			direction: 'desc'
		},
		pagination: {
			page: 1,
			limit: 10
		},
		filters: {} as Record<string, any>
	});

	// Apply filters directly without URL navigation
	async function applyFilters() {
		// Clear existing filters
		requestOptions.filters = {};

		// Add filters if they exist
		if (selectedUserId && selectedUserId !== '') {
			requestOptions.filters.userId =
				typeof selectedUserId === 'object' && selectedUserId.value
					? selectedUserId.value
					: String(selectedUserId);
		}

		if (selectedEventType && selectedEventType !== '') {
			requestOptions.filters.event =
				typeof selectedEventType === 'object' && selectedEventType.value
					? selectedEventType.value
					: String(selectedEventType);
		}

		if (selectedClientId && selectedClientId !== '') {
			requestOptions.filters.clientId =
				typeof selectedClientId === 'object' && selectedClientId.value
					? selectedClientId.value
					: String(selectedClientId);
		}

		// Reset pagination to first page when applying filters
		requestOptions.pagination.page = 1;

		// Call the refreshAuditLogs function on the AuditLogList component
		if (auditLogListComponent?.refreshAuditLogs) {
			auditLogs = await auditLogListComponent.refreshAuditLogs(requestOptions);
		}
	}

	// Clear all filters
	function clearFilters() {
		selectedUserId = '';
		selectedEventType = '';
		selectedClientId = '';

		// Reset filters and refresh data
		requestOptions.filters = {};
		if (auditLogListComponent?.refreshAuditLogs) {
			auditLogListComponent.refreshAuditLogs(requestOptions);
		}
	}
</script>

<svelte:head>
	<title>Global Audit Log</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<Card.Title>Global Audit Log</Card.Title>
		<Card.Description class="mt-1">See all user activity for the last 3 months.</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
			<div>
				<Select.Root
					selected={selectedUserId}
					onSelectedChange={(value) => (selectedUserId = value)}
				>
					<Select.Trigger class="w-full">
						<Select.Value placeholder="Filter by user..." />
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="">All Users</Select.Item>
						{#each data.users as user}
							<Select.Item value={user.id}>{user.username}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div>
				<Select.Root
					selected={selectedEventType}
					onSelectedChange={(value) => (selectedEventType = value)}
				>
					<Select.Trigger class="w-full">
						<Select.Value placeholder="Filter by event type..." />
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="">All Events</Select.Item>
						{#each data.eventTypes as eventType}
							<Select.Item value={eventType.value}>{eventType.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div>
				<Select.Root
					selected={selectedClientId}
					onSelectedChange={(value) => (selectedClientId = value)}
				>
					<Select.Trigger class="w-full">
						<Select.Value placeholder="Filter by client..." />
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="">All Clients</Select.Item>
						{#each data.clients as client}
							<Select.Item value={client.id}>{client.name}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
		</div>

		<div class="mb-4 flex justify-end space-x-2">
			<Button variant="outline" size="sm" on:click={clearFilters}>Clear Filters</Button>
			<Button variant="default" size="sm" on:click={applyFilters}>Apply Filters</Button>
		</div>

		<AuditLogList bind:this={auditLogListComponent} isAdmin={true} {auditLogs} {requestOptions} />
	</Card.Content>
</Card.Root>
