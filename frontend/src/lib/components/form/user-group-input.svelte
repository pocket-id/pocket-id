<script lang="ts">
	import MultiSelect from '#lib/components/form/multi-select.svelte';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { latestRequest } from '#lib/utils/latest-request-util.ts';
	import { onMount } from 'svelte';

	let {
		selectedGroupIds = $bindable()
	}: {
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

	let userGroups = $state<{ value: string; label: string }[]>([]);
	let isLoading = $state(false);

	const latestLoad = latestRequest();

	function loadUserGroups(search?: string) {
		return latestLoad(async (signal) => {
			userGroups = (
				await userGroupService.list(
					{ search, sort: { column: 'friendlyName', direction: 'asc' } },
					signal
				)
			).data.map((group) => ({
				value: group.id,
				label: group.friendlyName
			}));

			// Ensure selected groups are still in the list
			for (const selectedGroupId of selectedGroupIds) {
				if (!userGroups.some((g) => g.value === selectedGroupId)) {
					const group = await userGroupService.get(selectedGroupId);
					if (signal.aborted) return;
					userGroups.push({ value: group.id, label: group.friendlyName });
				}
			}
		});
	}

	const onUserGroupSearch = debounced(
		(search: string) => loadUserGroups(search),
		300,
		(loading) => (isLoading = loading)
	);

	onMount(() => loadUserGroups());
</script>

<MultiSelect
	id="default-groups"
	items={userGroups}
	oninput={(e) => onUserGroupSearch(e.currentTarget.value)}
	bind:selectedItems={selectedGroupIds}
	{isLoading}
	placeholder={m.select_items()}
/>
