<script lang="ts">
	import SignupTokenListModal from '$lib/components/signup/signup-token-list-modal.svelte';
	import SignupTokenModal from '$lib/components/signup/signup-token-modal.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as DropdownButton from '$lib/components/ui/dropdown-button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, UserPen, UserPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserForm from './user-form.svelte';
	import UserList from './user-list.svelte';

	let { data } = $props();
	let users = $state(data.users);
	let usersRequestOptions = $state(data.usersRequestOptions);
	let signupTokens = $state(data.signupTokens);
	let signupTokensRequestOptions = $state(data.signupTokensRequestOptions);

	let selectedCreateOptions = $state(m.add_user());
	let expandAddUser = $state(false);
	let signupTokenModalOpen = $state(false);
	let signupTokenListModalOpen = $state(false);

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

	async function refreshSignupTokens() {
		signupTokens = await userService.listSignupTokens(signupTokensRequestOptions);
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
					{#if $appConfigStore.allowUserSignups !== 'disabled'}
						<DropdownButton.DropdownRoot>
							<DropdownButton.Root>
								<DropdownButton.Main disabled={false} onclick={() => (expandAddUser = true)}>
									{selectedCreateOptions}
								</DropdownButton.Main>

								<DropdownButton.DropdownTrigger>
									<DropdownButton.Trigger class="border-l" />
								</DropdownButton.DropdownTrigger>
							</DropdownButton.Root>

							<DropdownButton.Content align="end">
								<DropdownButton.Item onclick={() => (signupTokenModalOpen = true)}>
									{m.create_signup_token()}
								</DropdownButton.Item>
								<DropdownButton.Item onclick={() => (signupTokenListModalOpen = true)}>
									{m.view_active_signup_tokens()}
								</DropdownButton.Item>
							</DropdownButton.Content>
						</DropdownButton.DropdownRoot>
					{:else}
						<Button onclick={() => (expandAddUser = true)}>{m.add_user()}</Button>
					{/if}
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

<SignupTokenModal bind:open={signupTokenModalOpen} onTokenCreated={refreshSignupTokens} />
<SignupTokenListModal
	bind:open={signupTokenListModalOpen}
	bind:signupTokens
	{signupTokensRequestOptions}
	onTokenDeleted={refreshSignupTokens}
/>
