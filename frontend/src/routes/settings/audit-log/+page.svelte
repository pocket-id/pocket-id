<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import AuditLogList from './audit-log-list.svelte';

	let { data } = $props();
	let { auditLogs } = data;
	let auditLogsRequestOptions = $state(data.auditLogsRequestOptions);
</script>

<svelte:head>
	<title>{m.audit_log()}</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.audit_log()}</Card.Title>
		<Card.Description class="mt-1"
			>{m.see_your_account_activities_from_the_last_3_months()}</Card.Description
		>
	</Card.Header>
	<Card.Content>
		<AuditLogList auditLogs={data.auditLogs} requestOptions={auditLogsRequestOptions} />
	</Card.Content>
</Card.Root>
