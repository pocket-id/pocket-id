<script lang="ts">
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import AuditLogList from './audit-log-list.svelte';
	import { fly } from 'svelte/transition';
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
	<div in:fly={{ y: -20, duration: 300, delay: 100 }}>
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
	</div>
{/if}
