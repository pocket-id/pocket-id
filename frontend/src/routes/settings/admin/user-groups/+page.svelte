<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserGroupService from '$lib/services/user-group-service';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { UserGroupCreate, UserGroupWithUserCount } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, UserCog, UserPlus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserGroupForm from './user-group-form.svelte';
	import UserGroupList from './user-group-list.svelte';
	import { m } from '$lib/paraglide/messages';
	import { onMount } from 'svelte';

	let { data } = $props();
	let userGroups = $state(data.userGroups);
	let userGroupsRequestOptions = $state(data.userGroupsRequestOptions);
	let expandAddUserGroup = $state(false);
	let mounted = $state(false);

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

	onMount(() => {
		mounted = true;
	});
</script>

<svelte:head>
	<title>{m.user_groups()}</title>
</svelte:head>

{#if mounted}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<div class="flex items-center justify-between">
					<div>
						<Card.Title class="flex items-center gap-2 text-xl font-semibold">
							<UserPlus class="text-primary/80 h-5 w-5" />
							{m.create_user_group()}
						</Card.Title>
						<Card.Description
							>{m.create_a_new_group_that_can_be_assigned_to_users()}</Card.Description
						>
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
					<Card.Content class="bg-muted/20 pt-5">
						<UserGroupForm callback={createUserGroup} />
					</Card.Content>
				</div>
			{/if}
		</Card.Root>
	</div>

	<div class="animate-fade-in" style="animation-delay: 200ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					<UserCog class="text-primary/80 h-5 w-5" />
					{m.manage_user_groups()}
				</Card.Title>
			</Card.Header>
			<Card.Content class="bg-muted/20 pt-5">
				<UserGroupList {userGroups} requestOptions={userGroupsRequestOptions} />
			</Card.Content>
		</Card.Root>
	</div>
{/if}
