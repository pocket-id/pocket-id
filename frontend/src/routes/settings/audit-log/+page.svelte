<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import { LogsIcon } from 'lucide-svelte';
	import AuditLogList from '$lib/components/audit-log-list.svelte';

	let { data } = $props();
	let auditLogsRequestOptions = $state(data.auditLogsRequestOptions);
</script>

<svelte:head>
	<title>{m.audit_log()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<LogsIcon class="text-primary/80 h-5 w-5" />
				{m.audit_log()}
			</Card.Title>
			<Card.Description>{m.see_your_account_activities_from_the_last_3_months()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<AuditLogList auditLogs={data.auditLogs} requestOptions={auditLogsRequestOptions} />
		</Card.Content>
	</Card.Root>
</div>
