<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Avatar from '$lib/components/ui/avatar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { m } from '$lib/paraglide/messages';
	import WebAuthnService from '$lib/services/webauthn-service';
	import userStore from '$lib/stores/user-store';
	import { cachedProfilePicture } from '$lib/utils/cached-image-util';
	import { LucideLogOut, LucideUser } from '@lucide/svelte';

	const webauthnService = new WebAuthnService();

	async function logout() {
		await webauthnService.logout();
		goto('/login');
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger
		><Avatar.Root class="size-9">
			<Avatar.Image src={cachedProfilePicture.getUrl($userStore!.id)} />
		</Avatar.Root></DropdownMenu.Trigger
	>
	<DropdownMenu.Content class="min-w-40" align="end">
		<DropdownMenu.Label class="font-normal">
			<div class="flex flex-col space-y-1">
				<p class="text-sm leading-none font-medium">
					{$userStore?.firstName}
					{$userStore?.lastName}
				</p>
				<p class="text-muted-foreground text-xs leading-none">{$userStore?.email}</p>
			</div>
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		<DropdownMenu.Group>
			<DropdownMenu.Item onclick={() => goto('/settings/account')}
				><LucideUser class="mr-2 size-4" /> {m.my_account()}</DropdownMenu.Item
			>
			<DropdownMenu.Item onclick={logout}
				><LucideLogOut class="mr-2 size-4" /> {m.logout()}</DropdownMenu.Item
			>
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
