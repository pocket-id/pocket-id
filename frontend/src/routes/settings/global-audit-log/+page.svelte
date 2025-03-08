<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import AuditLogList from '../audit-log/audit-log-list.svelte';

	let { data } = $props();
</script>

<svelte:head>
	<title>Global Audit Log</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<Card.Title>Audit Log</Card.Title>
		<Card.Description class="mt-1">See all user activity for the last 3 months.</Card.Description>
	</Card.Header>
	<Card.Content>
		<AuditLogList isAdmin={true} auditLogs={data.auditLogs} />
	</Card.Content>
</Card.Root>
