<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, PlusIcon, UserPen, UserPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserForm from './user-form.svelte';
	import UserList from './user-list.svelte';
	import * as DropdownButton from '$lib/components/ui/dropdown-button';
	import SignupTokenModal from '$lib/components/signup-token-modal.svelte';

	let { data } = $props();
	let users = $state(data.users);
	let usersRequestOptions = $state(data.usersRequestOptions);

	let selectedCreateOptions = $state('Add User');
	let expandAddUser = $state(false);
	let signupTokenModalOpen = $state(false);

	const userService = new UserService();

	async function createUser(user: UserCreate) {
		let success = true;
		await userService
			.create(user)
			.then(() => toast.success(m.user_created_successfully()))
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		users = await userService.list(usersRequestOptions);
		return success;
	}
</script>

<svelte:head>
	<title>{m.users()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<div>
					<Card.Title>
						<UserPlus class="text-primary/80 size-5" />
						{m.create_user()}
					</Card.Title>
					<Card.Description
						>{m.add_a_new_user_to_appname({
							appName: $appConfigStore.appName
						})}.</Card.Description
					>
				</div>
				{#if !expandAddUser}
					<DropdownButton.DropdownRoot>
						<DropdownButton.Root>
							<DropdownButton.Main
								variant="default"
								disabled={false}
								onclick={() => (expandAddUser = true)}
							>
								<PlusIcon class="size-4" />
								{selectedCreateOptions}
							</DropdownButton.Main>

							<DropdownButton.DropdownTrigger>
								<DropdownButton.Trigger class="border-l" variant="default" />
							</DropdownButton.DropdownTrigger>
						</DropdownButton.Root>

						<DropdownButton.Content align="end">
							<DropdownButton.Item onclick={() => (signupTokenModalOpen = true)}>
								Create Signup Link
							</DropdownButton.Item>
						</DropdownButton.Content>
					</DropdownButton.DropdownRoot>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddUser = false)}>
						<LucideMinus class="size-5" />
					</Button>
				{/if}
			</div>
		</Card.Header>
		{#if expandAddUser}
			<div transition:slide>
				<Card.Content>
					<UserForm callback={createUser} />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<UserPen class="text-primary/80 size-5" />
				{m.manage_users()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<UserList {users} requestOptions={usersRequestOptions} />
		</Card.Content>
	</Card.Root>
</div>

<SignupTokenModal bind:open={signupTokenModalOpen} />
