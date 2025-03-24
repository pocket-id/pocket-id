<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import { LogsIcon } from 'lucide-svelte';
	import AuditLogList from './audit-log-list.svelte';
	import { onMount } from 'svelte';

	let { data } = $props();
	let { auditLogs } = data;
	let auditLogsRequestOptions = $state(data.auditLogsRequestOptions);
	let mounted = $state(false);

	onMount(() => {
		mounted = true;
	});
</script>

<svelte:head>
	<title>{m.audit_log()}</title>
</svelte:head>

{#if mounted}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					<LogsIcon class="text-primary/80 h-5 w-5" />
					{m.audit_log()}
				</Card.Title>
				<Card.Description class="mt-1"
					>{m.see_your_account_activities_from_the_last_3_months()}</Card.Description
				>
			</Card.Header>
			<Card.Content class="bg-muted/20 pt-5">
				<AuditLogList auditLogs={data.auditLogs} requestOptions={auditLogsRequestOptions} />
			</Card.Content>
		</Card.Root>
	</div>
{/if}
