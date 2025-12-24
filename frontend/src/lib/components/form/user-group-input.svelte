<script lang="ts">
	import SearchableMultiSelect from '$lib/components/form/searchable-multi-select.svelte';
	import UserGroupService from '$lib/services/user-group-service';
	import { debounced } from '$lib/utils/debounce-util';
	import { onMount } from 'svelte';

	let {
		selectedGroupIds = $bindable()
	}: {
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

	let userGroups = $state<{ value: string; label: string }[]>([]);
	let isLoading = $state(false);

	async function loadUserGroups(search?: string) {
		userGroups = (await userGroupService.list({ search })).data.map((group) => ({
			value: group.id,
			label: group.name
		}));

		// Ensure selected groups are still in the list
		for (const selectedGroupId of selectedGroupIds) {
			if (!userGroups.some((g) => g.value === selectedGroupId)) {
				const group = await userGroupService.get(selectedGroupId);
				userGroups.push({ value: group.id, label: group.name });
			}
		}
	}

	const onUserGroupSearch = debounced(
		async (search: string) => await loadUserGroups(search),
		300,
		(loading) => (isLoading = loading)
	);

	onMount(() => loadUserGroups());
</script>

<SearchableMultiSelect
	id="default-groups"
	items={userGroups}
	oninput={(e) => onUserGroupSearch(e.currentTarget.value)}
	selectedItems={selectedGroupIds}
	onSelect={(selected) => (selectedGroupIds = selected)}
	{isLoading}
	disableInternalSearch
/>
