<script lang="ts">
	import AuditLogList from '$lib/components/audit-log-list.svelte';
	import SearchableSelect from '$lib/components/form/searchable-select.svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLogFilter } from '$lib/types/audit-log.type';
	import AuditLogSwitcher from '../audit-log-switcher.svelte';

	let { data } = $props();

	const auditLogService = new AuditLogService();

	let auditLogs = $state(data.auditLogs);
	let requestOptions = $state(data.requestOptions);

	let filters: AuditLogFilter = $state({
		userId: '',
		event: '',
		location: '',
		clientName: ''
	});

	const locationTypes = $state({
		external: 'External Networks',
		internal: 'Internal Networks'
	});

	const eventTypes = $state({
		SIGN_IN: m.sign_in(),
		TOKEN_SIGN_IN: m.token_sign_in(),
		CLIENT_AUTHORIZATION: m.client_authorization(),
		NEW_CLIENT_AUTHORIZATION: m.new_client_authorization()
	});

	$effect(() => {
		auditLogService.listAllLogs(requestOptions, filters).then((response) => (auditLogs = response));
	});
</script>

<svelte:head>
	<title>{m.global_audit_log()}</title>
</svelte:head>

<AuditLogSwitcher currentPage="global" />

<Card.Root>
	<Card.Header>
		<Card.Title>{m.global_audit_log()}</Card.Title>
		<Card.Description class="mt-1"
			>{m.see_all_account_activities_from_the_last_3_months()}</Card.Description
		>
	</Card.Header>
	<Card.Content>
		<div class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-4">
			<div>
				{#await auditLogService.listUsers()}
					<Select.Root type="single">
						<Select.Trigger class="w-full" disabled>
							{m.all_users()}
						</Select.Trigger>
					</Select.Root>
				{:then users}
					<SearchableSelect
						class="w-full"
						items={[
							{ value: '', label: m.all_users() },
							...Object.entries(users).map(([id, username]) => ({
								value: id,
								label: username
							}))
						]}
						bind:value={filters.userId}
					/>
				{/await}
			</div>
			<div>
				<SearchableSelect
					class="w-full"
					items={[
						{ value: '', label: m.all_events() },
						...Object.entries(eventTypes).map(([value, label]) => ({
							value,
							label
						}))
					]}
					bind:value={filters.event}
				/>
			</div>
			<div>
				<SearchableSelect
					class="w-full"
					items={[
						{ value: '', label: m.all_locations() },
						...Object.entries(locationTypes).map(([value, label]) => ({
							value,
							label
						}))
					]}
					bind:value={filters.location}
				/>
			</div>
			<div>
				{#await auditLogService.listClientNames()}
					<Select.Root
						type="single"
						value={filters.clientName}
						onValueChange={(v) => (filters.clientName = v)}
					>
						<Select.Trigger class="w-full" disabled>
							{m.all_clients()}
						</Select.Trigger>
					</Select.Root>
				{:then clientNames}
					<SearchableSelect
						class="w-full"
						items={[
							{ value: '', label: m.all_clients() },
							...clientNames.map((name) => ({
								value: name,
								label: name
							}))
						]}
						bind:value={filters.clientName}
					/>
				{/await}
			</div>
		</div>

		<AuditLogList isAdmin={true} {auditLogs} {requestOptions} />
	</Card.Content>
</Card.Root>
