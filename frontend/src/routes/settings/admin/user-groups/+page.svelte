<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserGroupService from '$lib/services/user-group-service';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { UserGroupCreate, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserGroupForm from './user-group-form.svelte';
	import UserGroupList from './user-group-list.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let userGroups = $state(data.userGroups);
	let userGroupsRequestOptions = $state(data.userGroupsRequestOptions);
	let expandAddUserGroup = $state(false);

	const userGroupService = new UserGroupService();

	async function createUserGroup(userGroup: UserGroupCreate) {
		let success = true;
		await userGroupService
			.create(userGroup)
			.then((createdUserGroup) => {
				toast.success(m.user_group_created_successfully());
				goto(`/settings/admin/user-groups/${createdUserGroup.id}`);
			})
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});
		return success;
	}
</script>

<svelte:head>
	<title>{m.user_groups()}</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>{m.create_user_group()}</Card.Title>
				<Card.Description>{m.create_a_new_group_that_can_be_assigned_to_users()}</Card.Description>
			</div>
			{#if !expandAddUserGroup}
				<Button on:click={() => (expandAddUserGroup = true)}>{m.add_group()}</Button>
			{:else}
				<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddUserGroup = false)}>
					<LucideMinus class="h-5 w-5" />
				</Button>
			{/if}
		</div>
	</Card.Header>
	{#if expandAddUserGroup}
		<div transition:slide>
			<Card.Content>
				<UserGroupForm callback={createUserGroup} />
			</Card.Content>
		</div>
	{/if}
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.manage_user_groups()}</Card.Title>
	</Card.Header>
	<Card.Content>
		<UserGroupList {userGroups} requestOptions={userGroupsRequestOptions} />
	</Card.Content>
</Card.Root>
