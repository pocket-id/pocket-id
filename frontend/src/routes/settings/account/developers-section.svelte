<script lang="ts">
	import AdvancedDisclosure from '#lib/components/advanced-disclosure.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import { LucideChevronRight, LucideKey } from '@lucide/svelte';
	import { onMount } from 'svelte';

	let count = $state<number>();

	onMount(async () => {
		const res = await new ApiKeyService().list({ pagination: { page: 1, limit: 1 } });
		count = res.pagination.totalItems;
	});

	const countLabel = $derived(
		count === undefined
			? ''
			: count === 0
				? m.no_api_keys()
				: count === 1
					? m.api_keys_count_one()
					: m.api_keys_count({ count })
	);
</script>

<AdvancedDisclosure title={m.developers()} summary={m.developers_summary()}>
	<SettingRow title={m.api_keys()} description={countLabel} icon={LucideKey}>
		{#snippet control()}
			<Button variant="secondary" size="sm" href="/settings/account/api-keys">
				{m.manage()}
				<LucideChevronRight />
			</Button>
		{/snippet}
	</SettingRow>
</AdvancedDisclosure>
