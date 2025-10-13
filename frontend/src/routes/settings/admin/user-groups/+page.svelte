<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import type { UserGroupCreate } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, UserCog, UserPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserGroupForm from './user-group-form.svelte';
	import UserGroupList from './user-group-list.svelte';

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

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<div>
					<Card.Title>
						<UserPlus class="text-primary/80 size-5" />
						{m.create_user_group()}
					</Card.Title>
					<Card.Description>{m.create_a_new_group_that_can_be_assigned_to_users()}</Card.Description
					>
				</div>
				{#if !expandAddUserGroup}
					<Button onclick={() => (expandAddUserGroup = true)}>{m.add_group()}</Button>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddUserGroup = false)}>
						<LucideMinus class="size-5" />
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
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<UserCog class="text-primary/80 size-5" />
				{m.manage_user_groups()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<UserGroupList />
		</Card.Content>
	</Card.Root>
</div>
