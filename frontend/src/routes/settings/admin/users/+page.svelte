<script lang="ts">
	import SignupTokenListModal from '$lib/components/signup/signup-token-list-modal.svelte';
	import SignupTokenModal from '$lib/components/signup/signup-token-modal.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as ButtonGroup from '$lib/components/ui/button-group';
	import * as Card from '$lib/components/ui/card';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { ChevronDown, LucideMinus, UserPen, UserPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import type { PageProps } from './$types';
	import UserForm from './user-form.svelte';
	import UserList from './user-list.svelte';

	let { data }: PageProps = $props();

	let selectedCreateOptions = $state(m.add_user());
	let expandAddUser = $state(false);
	let signupTokenModalOpen = $state(false);
	let signupTokenListModalOpen = $state(false);

	let userListRef: UserList;
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

		await userListRef.refresh();
		return success;
	}
</script>

<svelte:head>
	<title>{m.users()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex flex-wrap items-center justify-between md:flex-nowrap gap-4">
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
						<ButtonGroup.Root>
							<Button onclick={() => (expandAddUser = true)}>
								{selectedCreateOptions}
							</Button>
							<DropdownMenu.Root>
								<DropdownMenu.Trigger>
									{#snippet child({ props })}
										<Button {...props} size="icon" aria-label="Create options">
											<ChevronDown />
										</Button>
									{/snippet}
								</DropdownMenu.Trigger>
								<DropdownMenu.Content align="end">
									<DropdownMenu.Item onclick={() => (signupTokenModalOpen = true)}>
										{m.create_signup_token()}
									</DropdownMenu.Item>
									<DropdownMenu.Item onclick={() => (signupTokenListModalOpen = true)}>
										{m.view_active_signup_tokens()}
									</DropdownMenu.Item>
								</DropdownMenu.Content>
							</DropdownMenu.Root>
						</ButtonGroup.Root>
					{:else}
						<Button class="w-full md:w-auto" onclick={() => (expandAddUser = true)}
							>{m.add_user()}</Button
						>
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
					<UserForm
						callback={createUser}
						emailsVerifiedPerDefault={data.emailsVerifiedPerDefault}
					/>
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
			<UserList bind:this={userListRef} />
		</Card.Content>
	</Card.Root>
</div>

<SignupTokenModal bind:open={signupTokenModalOpen} />
<SignupTokenListModal bind:open={signupTokenListModalOpen} />
