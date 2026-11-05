<script lang="ts">
	import SettingRow from '#lib/components/setting-row.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import ApiKeyService from '#lib/services/api-key-service.ts';
	import ChevronRightIcon from '~icons/solar/alt-arrow-right-linear';
	import KeyIcon from '~icons/solar/key-bold-duotone';
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

<SettingRow title={m.api_keys()} description={countLabel} icon={KeyIcon}>
	{#snippet control()}
		<!-- The chevron glyph carries empty space beside it, so a tighter end padding looks even with the start -->
		<Button variant="secondary" size="sm" class="pe-2" href="/settings/account/api-keys">
			{m.manage()}
			<ChevronRightIcon />
		</Button>
	{/snippet}
</SettingRow>
