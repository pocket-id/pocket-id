<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import AuditLogList from '../audit-log/audit-log-list.svelte';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import UserService from '$lib/services/user-service';
	import OIDCService from '$lib/services/oidc-service';
	import { onMount } from 'svelte';
	import type { AuditLog } from '$lib/types/audit-log.type';

	let { data } = $props();

	// Initialize with empty string values to represent "All" options
	let selectedUserId = $state<string | null>('');
	let selectedEventType = $state<string | null>('');
	let selectedClientId = $state<string | null>('');

	let users = $state<Array<{ id: string; username: string }>>([]);
	let eventTypes = $state<Array<{ value: string; label: string }>>([]);
	let clients = $state<Array<{ id: string; name: string }>>([]);

	let auditLogList: any; // Reference to the AuditLogList component
	let requestOptions: any; // Object to store request options

	const userService = new UserService();
	const oidcService = new OIDCService();

	// Fetch all users and clients for the dropdown
	onMount(async () => {
		// Get users for dropdown
		const userResponse = await userService.list();
		users = userResponse.data.map((user) => ({
			id: user.id,
			username: user.username || user.firstName + ' ' + user.lastName
		}));

		// Get clients for dropdown
		const clientResponse = await oidcService.listClients();
		clients = clientResponse.data.map((client) => ({
			id: client.id,
			name: client.name
		}));

		// Add event types (these are the standard event types in your system)
		eventTypes = [
			{ value: 'SIGN_IN', label: 'Sign In' },
			{ value: 'TOKEN_SIGN_IN', label: 'Token Sign In' },
			{ value: 'CLIENT_AUTHORIZATION', label: 'Client Authorization' },
			{ value: 'NEW_CLIENT_AUTHORIZATION', label: 'New Client Authorization' }
		];
	});

	// Apply filters and refresh the audit log
	async function applyFilters() {
		// Only include non-empty values in filters
		const filters: Record<string, string> = {};

		// Only add to filters if not empty/null and not "All" (empty string)
		if (selectedUserId && selectedUserId !== '') {
			filters.userId =
				typeof selectedUserId === 'object' && selectedUserId.value
					? selectedUserId.value
					: String(selectedUserId);
		}

		if (selectedEventType && selectedEventType !== '') {
			filters.event =
				typeof selectedEventType === 'object' && selectedEventType.value
					? selectedEventType.value
					: String(selectedEventType);
		}

		if (selectedClientId && selectedClientId !== '') {
			filters.clientId =
				typeof selectedClientId === 'object' && selectedClientId.value
					? selectedClientId.value
					: String(selectedClientId);
		}

		// Update your stored request options object
		requestOptions = {
			sort: {
				column: 'createdAt',
				direction: 'desc'
			},
			pagination: {
				page: 1,
				limit: 10
			},
			filters: Object.keys(filters).length > 0 ? filters : undefined
		};

		if (auditLogList && auditLogList.refreshAuditLogs) {
			await auditLogList.refreshAuditLogs(requestOptions);
		}
		console.log('applying', selectedUserId, selectedEventType, selectedClientId);
	}

	// Clear all filters
	function clearFilters() {
		selectedUserId = '';
		selectedEventType = '';
		selectedClientId = '';

		// Reset request options without filters
		requestOptions = {
			sort: {
				column: 'createdAt',
				direction: 'desc'
			},
			pagination: {
				page: 1,
				limit: 10
			}
		};

		if (auditLogList && auditLogList.refreshAuditLogs) {
			auditLogList.refreshAuditLogs(requestOptions);
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
						{#each users as user}
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
						{#each eventTypes as eventType}
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
						{#each clients as client}
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

		<AuditLogList bind:this={auditLogList} isAdmin={true} auditLogs={data.auditLogs} />
	</Card.Content>
</Card.Root>
