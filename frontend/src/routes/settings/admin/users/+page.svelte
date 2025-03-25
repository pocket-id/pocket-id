<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import UserService from '$lib/services/user-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type { UserCreate } from '$lib/types/user.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, UserPen, UserPlus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import UserForm from './user-form.svelte';
	import UserList from './user-list.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let users = $state(data.users);
	let usersRequestOptions = $state(data.usersRequestOptions);

	let expandAddUser = $state(false);
	let mounted = $state(false);

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

<div class="animate-fade-in" style="animation-delay: 100ms;">
	<Card.Root>
		<Card.Header class="border-b">
			<div class="flex items-center justify-between">
				<div>
					<Card.Title class="flex items-center gap-2 text-xl font-semibold">
						<UserPlus class="text-primary/80 h-5 w-5" />
						{m.create_user()}
					</Card.Title>
					<Card.Description
						>{m.add_a_new_user_to_appname({
							appName: $appConfigStore.appName
						})}.</Card.Description
					>
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
				<Card.Content class="bg-muted/20 pt-5">
					<UserForm callback={createUser} />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div class="animate-fade-in" style="animation-delay: 200ms;">
	<Card.Root>
		<Card.Header class="border-b">
			<Card.Title class="flex items-center gap-2 text-xl font-semibold">
				<UserPen class="text-primary/80 h-5 w-5" />
				{m.manage_users()}
			</Card.Title>
		</Card.Header>
		<Card.Content class="bg-muted/20 pt-5">
			<UserList {users} requestOptions={usersRequestOptions} />
		</Card.Content>
	</Card.Root>
</div>
