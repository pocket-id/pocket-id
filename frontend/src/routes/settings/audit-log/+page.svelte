<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import AuditLogList from './audit-log-list.svelte';

	let { data } = $props();
	let { auditLogs } = data;
	let auditLogsRequestOptions = $state(data.auditLogsRequestOptions);
</script>

<svelte:head>
	<title>Audit Log</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<Card.Title>Audit Log</Card.Title>
		<Card.Description class="mt-1"
			>See your account activities from the last 3 months.</Card.Description
		>
	</Card.Header>
	<Card.Content>
		<AuditLogList auditLogs={data.auditLogs} requestOptions={auditLogsRequestOptions} />
	</Card.Content>
</Card.Root>
