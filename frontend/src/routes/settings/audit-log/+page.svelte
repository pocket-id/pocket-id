<script lang="ts">
	import AuditLogList from '$lib/components/audit-log-list.svelte';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import userStore from '$lib/stores/user-store';
	import { LogsIcon } from '@lucide/svelte';
	import AuditLogSwitcher from './audit-log-switcher.svelte';
</script>

<svelte:head>
	<title>{m.audit_log()}</title>
</svelte:head>

{#if $userStore?.isAdmin}
	<AuditLogSwitcher currentPage="personal" />
{/if}

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<LogsIcon class="text-primary/80 size-5" />
				{m.audit_log()}
			</Card.Title>
			<Card.Description>{m.see_your_account_activities_from_the_last_3_months()}</Card.Description>
		</Card.Header>
		<Card.Content>
			<AuditLogList />
		</Card.Content>
	</Card.Root>
</div>
