<script lang="ts">
	import AuditLogList from '#lib/components/audit-log-list.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { m } from '#lib/paraglide/messages.js';
</script>

<svelte:head>
	<title>{m.activity()}</title>
</svelte:head>

<PageHeader title={m.activity()} description={m.activity_description()} />

<div class="mt-4">
	<AuditLogList />
</div>
