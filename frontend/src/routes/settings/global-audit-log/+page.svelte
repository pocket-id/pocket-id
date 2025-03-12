<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import AuditLogList from '$lib/components/audit-log-list.svelte';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { FilterMap } from '$lib/types/pagination.type';

	// Get data from server
	let { data } = $props();

	// Initialize the service
	const auditLogService = new AuditLogService();

	// Create state variables
	let auditLogs = $state(data.auditLogs);
	let requestOptions = $state(data.requestOptions);

	// Initialize selections with string types, not objects
	let selectedUserId = $state(''); // Empty string represents "All Users"
	let selectedEventType = $state(''); // Empty string represents "All Events"
	let selectedClientId = $state(''); // Empty string represents "All Clients"

	// Create mapping objects for display purposes
	const userMap = Object.fromEntries([
		['', 'All Users'],
		...data.users.map((user) => [user.id, user.username])
	]);

	const eventMap = Object.fromEntries([
		['', 'All Events'],
		...data.eventTypes.map((event) => [event.value, event.label])
	]);

	const clientMap = Object.fromEntries([
		['', 'All Clients'],
		...data.clients.map((client) => [client.id, client.name])
	]);

	// Apply filters directly
	async function applyFilters() {
		// Initialize filters as an empty map
		const filters: FilterMap = {};

		// Add filters if they exist
		if (selectedUserId) {
			filters.userId = selectedUserId;
		}

		if (selectedEventType) {
			filters.event = selectedEventType;
		}

		if (selectedClientId) {
			filters.clientId = selectedClientId;
		}

		requestOptions.filters = filters;

		const result = await auditLogService.listAllLogs(requestOptions);
		auditLogs = { ...result };
	}

	// Clear all filters
	async function clearFilters() {
		selectedUserId = '';
		selectedEventType = '';
		selectedClientId = '';

		requestOptions.filters = {};

		const result = await auditLogService.listAllLogs(requestOptions);
		auditLogs = { ...result };
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
					selected={{
						value: selectedUserId,
						label: userMap[selectedUserId] || 'All Users'
					}}
					onSelectedChange={(v) => (selectedUserId = v!.value)}
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
					selected={{
						value: selectedEventType,
						label: eventMap[selectedEventType] || 'All Events'
					}}
					onSelectedChange={(v) => (selectedEventType = v!.value)}
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
					selected={{
						value: selectedClientId,
						label: clientMap[selectedClientId] || 'All Clients'
					}}
					onSelectedChange={(v) => (selectedClientId = v!.value)}
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
