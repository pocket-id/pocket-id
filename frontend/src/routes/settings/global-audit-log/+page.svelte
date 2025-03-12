<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import AuditLogList from '$lib/components/audit-log-list.svelte';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { FilterMap } from '$lib/types/pagination.type';

	// Get data from server
	let { data } = $props();

	// Initialize the service
	const auditLogService = new AuditLogService();

	// Create state variables
	let auditLogs = $state(data.auditLogs);
	let requestOptions = $state(data.requestOptions);

	// Initialize selections with empty values
	let selectedUserId = $state('');
	let selectedEventType = $state('');
	let selectedClientId = $state('');

	// Helper function to safely extract value from select options
	function extractValueFromSelect(value: any): string {
		if (!value) return '';
		if (typeof value === 'string') return value;
		if (typeof value === 'object' && value !== null && 'value' in value) {
			return String(value.value);
		}
		return '';
	}

	// Apply filters directly
	async function applyFilters() {
		// Initialize filters as an empty map
		const filters: FilterMap = {};

		// Add filters if they exist, extracting just the string value
		const userId = extractValueFromSelect(selectedUserId);
		if (userId) {
			filters.userId = userId;
		}

		const eventType = extractValueFromSelect(selectedEventType);
		if (eventType) {
			filters.event = eventType;
		}

		const clientId = extractValueFromSelect(selectedClientId);
		if (clientId) {
			filters.clientId = clientId;
		}

		// Set the filters on the request options
		requestOptions.filters = filters;

		// Fetch the audit logs with the updated filters
		const result = await auditLogService.listAllLogs(requestOptions);
		auditLogs = { ...result }; // Create a new object to ensure reactivity
	}

	// Clear all filters
	async function clearFilters() {
		selectedUserId = '';
		selectedEventType = '';
		selectedClientId = '';

		// Reset filters to empty object
		requestOptions.filters = {};

		// Fetch the audit logs with cleared filters
		const result = await auditLogService.listAllLogs(requestOptions);
		auditLogs = { ...result }; // Create a new object to ensure reactivity
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

		<AuditLogList isAdmin={true} {auditLogs} {requestOptions} />
	</Card.Content>
</Card.Root>
