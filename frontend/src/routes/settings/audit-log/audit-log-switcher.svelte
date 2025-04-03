<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';

	let { currentPage }: { currentPage: 'personal' | 'global' } = $props();
</script>

<div class="flex justify-end no-fade">
	<Tabs.Root value={currentPage} >
		<Tabs.List>
			<Tabs.Trigger onclick={() => goto('/settings/audit-log')} value="personal"
				>{m.personal()}</Tabs.Trigger
			>
			<Tabs.Trigger onclick={() => goto('/settings/audit-log/global')} value="global"
				>{m.global()}</Tabs.Trigger
			>
		</Tabs.List>
	</Tabs.Root>
</div>
