<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import AuditLogList from '$lib/components/audit-log-list.svelte';
	import AuditLogService from '$lib/services/audit-log-service';
	import UserService from '$lib/services/user-service';
	import OIDCService from '$lib/services/oidc-service';
	import type { FilterMap } from '$lib/types/pagination.type';
	import { onMount } from 'svelte';
	import { Loader2 } from 'lucide-svelte';

	// Get data from server
	let { data } = $props();

	// Initialize Services
	const auditLogService = new AuditLogService();
	const userService = new UserService();
	const oidcService = new OIDCService();

	// Create state variables
	let auditLogs = $state(data.auditLogs);
	let requestOptions = $state(data.requestOptions);

	// Initialize selections with string types, not objects
	let selectedUserId = $state(''); // Empty string represents "All Users"
	let selectedEventType = $state(''); // Empty string represents "All Events"
	let selectedClientId = $state(''); // Empty string represents "All Clients"

	// Create promises for client-side data fetching
	const usersPromise = userService.list();
	const clientsPromise = oidcService.listClients();

	const eventTypes = $state([
		{ value: 'SIGN_IN', label: 'Sign In' },
		{ value: 'TOKEN_SIGN_IN', label: 'Token Sign In' },
		{ value: 'CLIENT_AUTHORIZATION', label: 'Client Authorization' },
		{ value: 'NEW_CLIENT_AUTHORIZATION', label: 'New Client Authorization' }
	]);

	// Initialize maps for display labels
	let userMap: Record<string, string> = $state({});
	let eventMap: Record<string, string> = $state({});
	let clientMap: Record<string, string> = $state({});

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

	// Set up event map
	onMount(() => {
		// Initialize event map
		eventMap = Object.fromEntries([
			['', 'All Events'],
			...eventTypes.map((event) => [event.value, event.label])
		]);
	});
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
				{#await usersPromise}
					<Select.Root>
						<Select.Trigger class="w-full" disabled>
							<Select.Value>
								<div class="flex items-center gap-2">
									<Loader2 class="h-4 w-4 animate-spin" />
									<span>Loading users...</span>
								</div>
							</Select.Value>
						</Select.Trigger>
					</Select.Root>
				{:then response}
					{#if response}
						{@const users = response.data.map((user) => ({
							id: user.id,
							username: user.username || `${user.firstName} ${user.lastName}`
						}))}
						{@const newUserMap = Object.fromEntries([
							['', 'All Users'],
							...users.map((user) => [user.id, user.username])
						])}
						{@const updateUserMap = () => {
							userMap = newUserMap;
							return '';
						}}
						<Select.Root
							selected={{
								value: selectedUserId,
								label: updateUserMap() || userMap[selectedUserId] || 'All Users'
							}}
							onSelectedChange={(v) => (selectedUserId = v!.value)}
						>
							<Select.Trigger class="w-full">
								<Select.Value placeholder="Filter by user..." />
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="">All Users</Select.Item>
								{#each users as user}
									<Select.Item value={user.id}>{user.username}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{/if}
				{:catch error}
					<Select.Root>
						<Select.Trigger class="text-destructive w-full" disabled>
							<Select.Value>Error loading users</Select.Value>
						</Select.Trigger>
					</Select.Root>
				{/await}
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
				{#await clientsPromise}
					<Select.Root>
						<Select.Trigger class="w-full" disabled>
							<Select.Value>
								<div class="flex items-center gap-2">
									<Loader2 class="h-4 w-4 animate-spin" />
									<span>Loading clients...</span>
								</div>
							</Select.Value>
						</Select.Trigger>
					</Select.Root>
				{:then response}
					{#if response}
						{@const clients = response.data.map((client) => ({
							id: client.id,
							name: client.name
						}))}
						{@const newClientMap = Object.fromEntries([
							['', 'All Clients'],
							...clients.map((client) => [client.id, client.name])
						])}
						{@const updateClientMap = () => {
							clientMap = newClientMap;
							return '';
						}}
						<Select.Root
							selected={{
								value: selectedClientId,
								label: updateClientMap() || clientMap[selectedClientId] || 'All Clients'
							}}
							onSelectedChange={(v) => (selectedClientId = v!.value)}
						>
							<Select.Trigger class="w-full">
								<Select.Value placeholder="Filter by client..." />
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="">All Clients</Select.Item>
								{#each clients as client}
									<Select.Item value={client.id}>{client.name}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{/if}
				{:catch error}
					<Select.Root>
						<Select.Trigger class="text-destructive w-full" disabled>
							<Select.Value>Error loading clients</Select.Value>
						</Select.Trigger>
					</Select.Root>
				{/await}
			</div>
		</div>

		<div class="mb-4 flex justify-end space-x-2">
			<Button variant="outline" size="sm" on:click={clearFilters}>Clear Filters</Button>
			<Button variant="default" size="sm" on:click={applyFilters}>Apply Filters</Button>
		</div>

		<AuditLogList isAdmin={true} {auditLogs} {requestOptions} />
	</Card.Content>
</Card.Root>
