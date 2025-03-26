<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserForm from './user-form.svelte';
	import UserList from './user-list.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let users = $state(data.users);
	let usersRequestOptions = $state(data.usersRequestOptions);

	let expandAddUser = $state(false);

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

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>{m.create_user()}</Card.Title>
				<Card.Description>{m.add_a_new_user_to_appname({ appName: $appConfigStore.appName })}.</Card.Description>
			</div>
			{#if !expandAddUser}
				<Button on:click={() => (expandAddUser = true)}>{m.add_user()}</Button>
			{:else}
				<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddUser = false)}>
					<LucideMinus class="h-5 w-5" />
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

<Card.Root>
	<Card.Header>
		<Card.Title>{m.manage_users()}</Card.Title>
	</Card.Header>
	<Card.Content>
		<UserList {users} requestOptions={usersRequestOptions} />
	</Card.Content>
</Card.Root>
